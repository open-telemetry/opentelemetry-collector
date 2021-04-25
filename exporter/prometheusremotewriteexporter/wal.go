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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/prometheus/prompb"
	"github.com/tidwall/wal"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

type walConfig struct {
	// Note: These variable names mirror what Prometheus' WAL uses for field names per
	// https://docs.google.com/document/d/1cCcoFgjDFwU2n823tKuMvrIhzHty4UDyn0IcfUHiyyI/edit#heading=h.mlf37ibqjgov
	// except that we are using underscores "_" instead of dashes "-".
	Dir               string        `mapstructure:"directory"`
	NBeforeTruncation int           `mapstructure:"cache_size"`
	RefreshDuration   time.Duration `mapstructure:"truncate_frequency"`
}

func (wc *walConfig) createWAL() (*wal.Log, error) {
	if wc == nil {
		// There are cases for which the WAL can be disabled.
		// TODO: Perhaps log that the WAL wasn't enabled.
		return nil, nil
	}
	wal, err := wal.Open(filepath.Join(wc.Dir, "prom_rw"), &wal.Options{
		SegmentCacheSize: wc.nBeforeTruncation(),
		NoCopy:           true,
	})
	if err != nil {
		return nil, fmt.Errorf("prometheusremotewriteexporter: failed to open WAL: %w", err)
	}
	return wal, nil
}

func (wc *walConfig) nBeforeTruncation() int {
	if wc == nil {
		return 0
	}
	if wc.NBeforeTruncation <= 0 {
		return 300
	}
	return wc.NBeforeTruncation
}

func (wc *walConfig) refreshDuration() time.Duration {
	if wc == nil {
		return 0
	}
	if wc.RefreshDuration <= 0 {
		// Following what Prometheus's WAL defaults.
		return 1 * time.Minute
	}
	return wc.RefreshDuration
}

func (prwe *PrwExporter) walEnabled() bool { return prwe.walConfig != nil }

var errWALDisabled = errors.New("WAL is disabled")

func (prwe *PrwExporter) retrieveWALIndices(context.Context) (err error) {
	if !prwe.walEnabled() {
		return errWALDisabled
	}

	prwe.walMu.Lock()
	defer prwe.walMu.Unlock()

	wal, err := prwe.walConfig.createWAL()
	if err != nil {
		return err
	}
	prwe.wal = wal

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

// exportThenTruncateWAL uploads the serialized data to the remote write endpoint,
// and then truncates and retrieves the new indices for the WAL.
func (prwe *PrwExporter) exportThenTruncateWAL(ctx context.Context, reqL []*prompb.WriteRequest) error {
	if len(reqL) == 0 {
		return nil
	}
	// Otherwise, time to flush.
	if errs := prwe.exportWriteRequests(ctx, reqL); len(errs) != 0 {
		return consumererror.Combine(errs)
	}
	// Save all the entries that aren't yet committed, to the tail of the WAL.
	if err := prwe.wal.Sync(); err != nil {
		return err
	}
	// Truncate the WAL from the front for the entries that we already
	// read from the WAL and then exported.
	if err := prwe.wal.TruncateFront(prwe.rWALIndex); err != nil && err != wal.ErrOutOfRange {
		return err
	}
	// Reset by retrieving the respective read and write WAL indices.
	return prwe.retrieveWALIndices(ctx)
}

func (prwe *PrwExporter) turnOnWALIfEnabled() error {
	if !prwe.walEnabled() {
		return nil
	}

	// From this point down below, the Write-Ahead-Log is enabled.
	shutdownCtx, cancel := context.WithCancel(context.Background())
	if err := prwe.retrieveWALIndices(shutdownCtx); err != nil {
		cancel()
		return err
	}

	go func() {
		<-prwe.closeChan
		cancel()
	}()

	prwe.wg.Add(1)
	go func() {
		defer prwe.wg.Done()

		_ = prwe.readFromWALThenExport(shutdownCtx)
	}()

	return nil
}

func (prwe *PrwExporter) closeWAL() {
	prwe.walMu.Lock()
	defer prwe.walMu.Unlock()

	if prwe.wal != nil {
		prwe.wal.Close()
		prwe.wal = nil
	}
}

func (prwe *PrwExporter) writeToWAL(requests []*prompb.WriteRequest) error {
	if !prwe.walEnabled() {
		return errWALDisabled
	}

	// Write in batch.
	batch := new(wal.Batch)
	for _, req := range requests {
		protoBlob, err := proto.Marshal(req)
		if err != nil {
			return err
		}
		prwe.wWALIndex++
		batch.Write(prwe.wWALIndex, protoBlob)
	}

	return prwe.wal.WriteBatch(batch)
}

func (prwe *PrwExporter) readFromWALThenExport(ctx context.Context) error {
	var reqL []*prompb.WriteRequest

	defer func() {
		// Keeping it within a closure to ensure that the later
		// updated value of reqL is always flushed to disk.
		prwe.exportThenTruncateWAL(ctx, reqL)
	}()

	freshTimer := func() *time.Timer {
		return time.NewTimer(prwe.walConfig.refreshDuration())
	}

	timer := freshTimer()
	maxCountPerUpload := prwe.walConfig.nBeforeTruncation()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		protoBlob, err := prwe.wal.Read(prwe.rWALIndex)
		if err == wal.ErrNotFound {
			// Perhaps the WAL is not yet ready.
			time.Sleep(1 * time.Second)
			continue
		}
		if err != nil {
			return err
		}
		req := new(prompb.WriteRequest)
		if err := proto.Unmarshal(protoBlob, req); err != nil {
			return err
		}

		reqL = append(reqL, req)
		prwe.rWALIndex++

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

		// Otherwise, it is time to export, flush and then truncate the WAL!
		if err := prwe.exportThenTruncateWAL(ctx, reqL); err != nil {
			return err
		}
		// Reset but reuse the write requests slice.
		reqL = reqL[:0]
	}
}
