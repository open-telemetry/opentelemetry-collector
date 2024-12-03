// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/internal"
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

			if qb.batchCfg.MaxSizeItems > 0 {
				var reqList []internal.Request
				var mergeSplitErr error
				if qb.currentBatch == nil || qb.currentBatch.req == nil {
					qb.resetTimer()
					reqList, mergeSplitErr = req.MergeSplit(ctx, qb.batchCfg.MaxSizeConfig, nil)
				} else {
					reqList, mergeSplitErr = qb.currentBatch.req.MergeSplit(ctx, qb.batchCfg.MaxSizeConfig, req)
				}

				if mergeSplitErr != nil || reqList == nil {
					qb.queue.OnProcessingFinished(idx, mergeSplitErr)
					qb.currentBatchMu.Unlock()
					continue
				}

				// If there was a split, we flush everything immediately.
				if reqList[0].ItemsCount() >= qb.batchCfg.MinSizeItems || len(reqList) > 1 {
					qb.currentBatch = nil
					qb.currentBatchMu.Unlock()
					for i := 0; i < len(reqList); i++ {
						qb.flushAsync(batch{
							req:     reqList[i],
							ctx:     ctx,
							idxList: []uint64{idx}})
						// TODO: handle partial failure
					}
					qb.resetTimer()
				} else {
					qb.currentBatch = &batch{
						req:     reqList[0],
						ctx:     ctx,
						idxList: []uint64{idx}}
					qb.currentBatchMu.Unlock()
				}
			} else {
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

				if qb.currentBatch.req.ItemsCount() >= qb.batchCfg.MinSizeItems {
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
				qb.flushCurrentBatchIfNecessary()
			}
		}
	}()
}

// Start starts the goroutine that reads from the queue and flushes asynchronously.
func (qb *DefaultBatcher) Start(_ context.Context, _ component.Host) error {
	// maxWorker being -1 means batcher is disabled. This is for testing queue sender metrics.
	if qb.maxWorkers == -1 {
		return nil
	}

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

// flushCurrentBatchIfNecessary sends out the current request batch if it is not nil
func (qb *DefaultBatcher) flushCurrentBatchIfNecessary() {
	qb.currentBatchMu.Lock()
	if qb.currentBatch == nil || qb.currentBatch.req == nil {
		qb.currentBatchMu.Unlock()
		return
	}
	batchToFlush := *qb.currentBatch
	qb.currentBatch = nil
	qb.currentBatchMu.Unlock()

	// flushAsync() blocks until successfully started a goroutine for flushing.
	qb.flushAsync(batchToFlush)
	qb.resetTimer()
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *DefaultBatcher) Shutdown(_ context.Context) error {
	qb.flushCurrentBatchIfNecessary()
	qb.stopWG.Wait()
	return nil
}
