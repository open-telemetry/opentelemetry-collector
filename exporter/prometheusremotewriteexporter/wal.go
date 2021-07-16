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
	"time"

	"github.com/gogo/protobuf/proto"
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
		SegmentCacheSize: wc.nBeforeTruncation(),
		NoCopy:           true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("prometheusremotewriteexporter: failed to open WAL: %w", err)
	}
	return wal, walPath, nil
}

type walConfig struct {
	// Note: These variable names are meant to closely mirror what Prometheus' WAL uses for field names per
	// https://docs.google.com/document/d/1cCcoFgjDFwU2n823tKuMvrIhzHty4UDyn0IcfUHiyyI/edit#heading=h.mlf37ibqjgov
	// but also we are using underscores "_" instead of dashes "-".
	Directory         string        `mapstructure:"directory"`
	NBeforeTruncation int           `mapstructure:"n_before_truncation"`
	TruncateFrequency time.Duration `mapstructure:"truncate_frequency"`
}

func (wc *walConfig) nBeforeTruncation() int {
	if wc.NBeforeTruncation <= 0 {
		return 300
	}
	return wc.NBeforeTruncation
}

var (
	errAlreadyClosed = errors.New("already closed")
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

func (prwe *prweWAL) closeWAL() {
	if prwe.wal != nil {
		prwe.wal.Close()
		prwe.wal = nil
	}
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
		prwe.wWALIndex++
		batch.Write(prwe.wWALIndex, protoBlob)
	}

	prwe.mu.Lock()
	defer prwe.mu.Unlock()

	return prwe.wal.WriteBatch(batch)
}

func (prwe *prweWAL) readPrompbFromWAL(_ context.Context, index uint64) (wreq *prompb.WriteRequest, err error) {
	var protoBlob []byte
	for i := 0; i < 12; i++ {
		if index <= 0 {
			index = 1
		}
		protoBlob, err = prwe.wal.Read(index)
		if errors.Is(err, wal.ErrNotFound) {
			time.Sleep(time.Duration(1<<i) * time.Millisecond)
			continue
		}
		if err != nil {
			return nil, err
		}

		req := new(prompb.WriteRequest)
		if err = proto.Unmarshal(protoBlob, req); err != nil {
			return nil, err
		}
		return req, nil
	}
	return nil, err
}
