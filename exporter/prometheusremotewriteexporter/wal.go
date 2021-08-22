// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusremotewriteexporter

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gogo/protobuf/proto"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/prometheus/prometheus/prompb"
	"github.com/tidwall/wal"
)

type prweWAL struct {
	mu        sync.Mutex // mu protects the fields below.
	wal       *wal.Log
	walConfig *walConfig
	walPath   string

	exportSink func(ctx context.Context, reqL []*prompb.WriteRequest) []error

	stopOnce  sync.Once
	stopChan  chan struct{}
	rWALIndex uint64
	wWALIndex uint64
}

const (
	defaultWALBufferSize        = 300
	defaultWALTruncateFrequency = 1 * time.Minute
)

type walConfig struct {
	// Note: These variable names are meant to closely mirror what Prometheus' WAL uses for field names per
	// https://docs.google.com/document/d/1cCcoFgjDFwU2n823tKuMvrIhzHty4UDyn0IcfUHiyyI/edit#heading=h.mlf37ibqjgov
	// but also we are using underscores "_" instead of dashes "-".
	Directory         string        `mapstructure:"directory"`
	BufferSize        int           `mapstructure:"buffer_size"`
	TruncateFrequency time.Duration `mapstructure:"truncate_frequency"`
}

func (wc *walConfig) bufferSize() int {
	if wc.BufferSize > 0 {
		return wc.BufferSize
	}
	return defaultWALBufferSize
}

func (wc *walConfig) truncateFrequency() time.Duration {
	if wc.TruncateFrequency > 0 {
		return wc.TruncateFrequency
	}
	return defaultWALTruncateFrequency
}

func newWAL(walConfig *walConfig, exportSink func(context.Context, []*prompb.WriteRequest) []error) (*prweWAL, error) {
	if walConfig == nil {
		// There are cases for which the WAL can be disabled.
		// TODO: Perhaps log that the WAL wasn't enabled.
		return nil, errNilConfig
	}

	return &prweWAL{
		exportSink: exportSink,
		walConfig:  walConfig,
		stopChan:   make(chan struct{}),
	}, nil
}

func (wc *walConfig) createWAL() (*wal.Log, string, error) {
	walPath := filepath.Join(wc.Directory, "prom_remotewrite")
	wal, err := wal.Open(walPath, &wal.Options{
		SegmentCacheSize: wc.bufferSize(),
		NoCopy:           true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("prometheusremotewriteexporter: failed to open WAL: %w", err)
	}
	return wal, walPath, nil
}

var (
	errAlreadyClosed = errors.New("already closed")
	errNilWAL        = errors.New("wal is nil")
	errNilConfig     = errors.New("expecting a non-nil configuration")
)

// retrieveWALIndices queries the WriteAheadLog for its current first and last indices.
func (prwe *prweWAL) retrieveWALIndices(context.Context) (err error) {
	prwe.mu.Lock()
	defer prwe.mu.Unlock()

	prwe.closeWAL()
	wal, walPath, err := prwe.walConfig.createWAL()
	if err != nil {
		return err
	}

	prwe.wal = wal
	prwe.walPath = walPath

	prwe.rWALIndex, err = prwe.wal.FirstIndex()
	if err != nil {
		return fmt.Errorf("prometheusremotewriteexporter: failed to retrieve the last WAL index: %w", err)
	}

	prwe.wWALIndex, err = prwe.wal.LastIndex()
	if err != nil {
		return fmt.Errorf("prometheusremotewriteexporter: failed to retrieve the first WAL index: %w", err)
	}
	return nil
}

func (prwe *prweWAL) stop() error {
	err := errAlreadyClosed
	prwe.stopOnce.Do(func() {
		prwe.mu.Lock()
		defer prwe.mu.Unlock()

		close(prwe.stopChan)
		prwe.closeWAL()
		err = nil
	})
	return err
}

// start begins reading from the WAL until prwe.stopChan is closed.
func (prwe *prweWAL) start(ctx context.Context) error {
	shutdownCtx, cancel := context.WithCancel(ctx)
	if err := prwe.retrieveWALIndices(shutdownCtx); err != nil {
		cancel()
		return err
	}

	go func() {
		<-prwe.stopChan
		cancel()
	}()

	return nil
}

func (prwe *prweWAL) run(ctx context.Context) (err error) {
	if err := prwe.start(ctx); err != nil {
		return err
	}

	// Start the process of exporting but wait until the exporting has started.
	waitUntilStartedCh := make(chan bool)
	go func() {
		prwe.continuallyPopWALThenExport(ctx, func() { close(waitUntilStartedCh) })
	}()
	<-waitUntilStartedCh
	return nil
}

// continuallyPopWALThenExport reads a prompb.WriteRequest proto encoded blob from the WAL, and moves
// the WAL's front index forward until either the read buffer period expires or the maximum
// buffer size is exceeded. When either of the two conditions are matched, it then exports
// the requests to the Remote-Write endpoint, and then truncates the head of the WAL to where
// it last read from.
func (prwe *prweWAL) continuallyPopWALThenExport(ctx context.Context, signalStart func()) (err error) {
	var reqL []*prompb.WriteRequest
	defer func() {
		// Keeping it within a closure to ensure that the later
		// updated value of reqL is always flushed to disk.
		if errL := prwe.exportSink(ctx, reqL); len(errL) != 0 && err == nil {
			err = multierror.Append(nil, errL...)
		}
	}()

	freshTimer := func() *time.Timer {
		return time.NewTimer(prwe.walConfig.truncateFrequency())
	}

	timer := freshTimer()
	defer func() {
		// Added in a closure to ensure we capture the later
		// updated value of timer when changed in the loop below.
		timer.Stop()
	}()

	signalStart()

	maxCountPerUpload := prwe.walConfig.bufferSize()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-prwe.stopChan:
			return nil
		default:
		}

		req, err := prwe.readPrompbFromWAL(ctx, atomic.LoadUint64(&prwe.rWALIndex))
		if err != nil {
			return err
		}
		reqL = append(reqL, req)

		// Now increment the WAL's read index.
		atomic.AddUint64(&prwe.rWALIndex, 1)

		shouldExport := false
		select {
		case <-timer.C:
			shouldExport = true
			timer.Stop()
			timer = freshTimer()
		default:
			shouldExport = len(reqL) >= maxCountPerUpload
		}

		if !shouldExport {
			continue
		}

		// Otherwise, it is time to export, flush and then truncate the WAL, but also to kill the timer!
		timer.Stop()
		if err := prwe.exportThenFrontTruncateWAL(ctx, reqL); err != nil {
			return err
		}
		// Reset but reuse the write requests slice.
		reqL = reqL[:0]
		timer = freshTimer()
	}
}

func (prwe *prweWAL) closeWAL() {
	if prwe.wal != nil {
		prwe.wal.Close()
		prwe.wal = nil
	}
}

func (prwe *prweWAL) syncAndTruncateFront() error {
	prwe.mu.Lock()
	defer prwe.mu.Unlock()

	if prwe.wal == nil {
		return errNilWAL
	}

	// Save all the entries that aren't yet committed, to the tail of the WAL.
	if err := prwe.wal.Sync(); err != nil {
		return err
	}
	// Truncate the WAL from the front for the entries that we already
	// read from the WAL and had already exported.
	if err := prwe.wal.TruncateFront(atomic.LoadUint64(&prwe.rWALIndex)); err != nil && err != wal.ErrOutOfRange {
		return err
	}
	return nil
}

func (prwe *prweWAL) exportThenFrontTruncateWAL(ctx context.Context, reqL []*prompb.WriteRequest) error {
	if len(reqL) == 0 {
		return nil
	}
	if cErr := ctx.Err(); cErr != nil {
		return nil
	}

	if errL := prwe.exportSink(ctx, reqL); len(errL) != 0 {
		return multierror.Append(nil, errL...)
	}
	if err := prwe.syncAndTruncateFront(); err != nil {
		return err
	}
	// Reset by retrieving the respective read and write WAL indices.
	return prwe.retrieveWALIndices(ctx)
}

// persistToWAL is the routine that'll be hooked into the exporter's receiving side and it'll
// write them to the Write-Ahead-Log so that shutdowns won't lose data, and that the routine that
// reads from the WAL can then process the previously serialized requests.
func (prwe *prweWAL) persistToWAL(requests []*prompb.WriteRequest) error {
	// Write all the requests to the WAL in a batch.
	batch := new(wal.Batch)
	for _, req := range requests {
		protoBlob, err := proto.Marshal(req)
		if err != nil {
			return err
		}
		wIndex := atomic.AddUint64(&prwe.wWALIndex, 1)
		batch.Write(wIndex, protoBlob)
	}

	prwe.mu.Lock()
	defer prwe.mu.Unlock()

	return prwe.wal.WriteBatch(batch)
}

func (prwe *prweWAL) readPrompbFromWAL(ctx context.Context, index uint64) (wreq *prompb.WriteRequest, err error) {
	prwe.mu.Lock()
	defer prwe.mu.Unlock()

	var protoBlob []byte
	for i := 0; i < 12; i++ {
		// Firstly check if we've been terminated, then exit if so.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if index <= 0 {
			index = 1
		}

		protoBlob, err = prwe.wal.Read(index)
		if err == nil { // The read succeeded.
			req := new(prompb.WriteRequest)
			if err = proto.Unmarshal(protoBlob, req); err != nil {
				return nil, err
			}
			return req, nil
		}

		if !errors.Is(err, wal.ErrNotFound) {
			return nil, err
		}

		if index <= 1 {
			// This could be the very first attempted read, so try again, after a small sleep.
			time.Sleep(time.Duration(1<<i) * time.Millisecond)
			continue
		}

		// Otherwise, we couldn't find the record, let's try watching
		// the WAL file until perhaps there is a write to it.
		walWatcher, werr := fsnotify.NewWatcher()
		if werr != nil {
			return nil, werr
		}
		if werr = walWatcher.Add(prwe.walPath); werr != nil {
			return nil, werr
		}

		// Watch until perhaps there is a write to the WAL file.
		watchCh := make(chan error)
		wErr := err
		go func() {
			defer func() {
				watchCh <- wErr
				close(watchCh)
				// Close the file watcher.
				walWatcher.Close()
			}()

			select {
			case <-ctx.Done(): // If the context was cancelled, bail out ASAP.
				wErr = ctx.Err()
				return

			case event, ok := <-walWatcher.Events:
				if !ok {
					return
				}
				switch event.Op {
				case fsnotify.Remove:
					// The file got deleted.
					// TODO: Add capabilities to search for the updated file.
				case fsnotify.Rename:
					// Renamed, we don't have information about the renamed file's new name.
				case fsnotify.Write:
					// Finally a write, let's try reading again, but after some watch.
					wErr = nil
				}

			case eerr, ok := <-walWatcher.Errors:
				if ok {
					wErr = eerr
				}
			}
		}()

		if gerr := <-watchCh; gerr != nil {
			return nil, gerr
		}

		// Otherwise a write occurred might have occurred,
		// and we can sleep for a little bit then try again.
		time.Sleep(time.Duration(1<<i) * time.Millisecond)
	}
	return nil, err
}
