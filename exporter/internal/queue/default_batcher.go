// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
)

// DefaultBatcher continuously reads from the queue and flushes asynchronously if size limit is met or on timeout.
type DefaultBatcher struct {
	BaseBatcher
	currentBatchMu sync.Mutex
	currentBatch   *batch
	timer          *time.Timer
	shutdownCh     chan bool
}

func (qb *DefaultBatcher) resetTimer() {
	if qb.batchCfg.FlushTimeout != 0 {
		qb.timer.Reset(qb.batchCfg.FlushTimeout)
	}
}

// startReadingFlushingGoroutine starts a goroutine that reads and then flushes.
func (qb *DefaultBatcher) startReadingFlushingGoroutine() {
	qb.stopWG.Add(1)
	go func() {
		defer qb.stopWG.Done()
		for {
			// Read() blocks until the queue is non-empty or until the queue is stopped.
			idx, ctx, req, ok := qb.queue.Read(context.Background())
			if !ok {
				qb.shutdownCh <- true
				return
			}

			qb.currentBatchMu.Lock()
			if qb.currentBatch == nil || qb.currentBatch.req == nil {
				qb.resetTimer()
				qb.currentBatch = &batch{
					req:     req,
					ctx:     ctx,
					idxList: []uint64{idx}}
			} else {
				mergedReq, mergeErr := qb.currentBatch.req.Merge(qb.currentBatch.ctx, req)
				if mergeErr != nil {
					qb.queue.OnProcessingFinished(idx, mergeErr)
					qb.currentBatchMu.Unlock()
					continue
				}
				qb.currentBatch = &batch{
					req:     mergedReq,
					ctx:     qb.currentBatch.ctx,
					idxList: append(qb.currentBatch.idxList, idx)}
			}

			if qb.currentBatch.req.ItemsCount() > qb.batchCfg.MinSizeItems {
				batchToFlush := *qb.currentBatch
				qb.currentBatch = nil
				qb.currentBatchMu.Unlock()

				// flushAsync() blocks until successfully started a goroutine for flushing.
				qb.flushAsync(batchToFlush)
				qb.resetTimer()
			} else {
				qb.currentBatchMu.Unlock()
			}
		}
	}()
}

// startTimeBasedFlushingGoroutine starts a goroutine that flushes on timeout.
func (qb *DefaultBatcher) startTimeBasedFlushingGoroutine() {
	qb.stopWG.Add(1)
	go func() {
		defer qb.stopWG.Done()
		for {
			select {
			case <-qb.shutdownCh:
				return
			case <-qb.timer.C:
				qb.currentBatchMu.Lock()
				if qb.currentBatch == nil || qb.currentBatch.req == nil {
					qb.currentBatchMu.Unlock()
					continue
				}
				batchToFlush := *qb.currentBatch
				qb.currentBatch = nil
				qb.currentBatchMu.Unlock()

				// flushAsync() blocks until successfully started a goroutine for flushing.
				qb.flushAsync(batchToFlush)
				qb.resetTimer()
			}
		}
	}()
}

// Start starts the goroutine that reads from the queue and flushes asynchronously.
func (qb *DefaultBatcher) Start(_ context.Context, _ component.Host) error {
	qb.startWorkerPool()
	qb.shutdownCh = make(chan bool, 1)

	if qb.batchCfg.FlushTimeout == 0 {
		qb.timer = time.NewTimer(math.MaxInt)
		qb.timer.Stop()
	} else {
		qb.timer = time.NewTimer(qb.batchCfg.FlushTimeout)
	}

	qb.startReadingFlushingGoroutine()
	qb.startTimeBasedFlushingGoroutine()

	return nil
}
