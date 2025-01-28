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
type DefaultBatcher[K internal.Request] struct {
	BaseBatcher[K]
	currentBatchMu sync.Mutex
	currentBatch   *batch[K]
	timer          *time.Timer
	shutdownCh     chan bool
}

func (qb *DefaultBatcher[K]) resetTimer() {
	if qb.batchCfg.FlushTimeout != 0 {
		qb.timer.Reset(qb.batchCfg.FlushTimeout)
	}
}

// startReadingFlushingGoroutine starts a goroutine that reads and then flushes.
func (qb *DefaultBatcher[K]) startReadingFlushingGoroutine() {
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
				var reqList []K
				var mergeSplitErr error
				if qb.currentBatch == nil {
					qb.resetTimer()
					reqList, mergeSplitErr = qb.merger.MergeSplit(req, nil, qb.batchCfg.MaxSizeConfig)
				} else {
					reqList, mergeSplitErr = qb.merger.MergeSplit(qb.currentBatch.req, req, qb.batchCfg.MaxSizeConfig)
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
						qb.flush(batch[K]{
							req:     reqList[i],
							ctx:     ctx,
							idxList: []uint64{idx},
						})
						// TODO: handle partial failure
					}
					qb.resetTimer()
				} else {
					qb.currentBatch = &batch[K]{
						req:     reqList[0],
						ctx:     ctx,
						idxList: []uint64{idx},
					}
					qb.currentBatchMu.Unlock()
				}
			} else {
				if qb.currentBatch == nil {
					qb.resetTimer()
					qb.currentBatch = &batch[K]{
						req:     req,
						ctx:     ctx,
						idxList: []uint64{idx},
					}
				} else {
					// TODO: consolidate implementation for the cases where MaxSizeConfig is specified and the case where it is not specified
					mergedReq, mergeErr := qb.merger.MergeSplit(qb.currentBatch.req, req, qb.batchCfg.MaxSizeConfig)
					if mergeErr != nil {
						qb.queue.OnProcessingFinished(idx, mergeErr)
						qb.currentBatchMu.Unlock()
						continue
					}
					qb.currentBatch = &batch[K]{
						req:     mergedReq[0],
						ctx:     qb.currentBatch.ctx,
						idxList: append(qb.currentBatch.idxList, idx),
					}
				}

				if qb.currentBatch.req.ItemsCount() >= qb.batchCfg.MinSizeItems {
					batchToFlush := *qb.currentBatch
					qb.currentBatch = nil
					qb.currentBatchMu.Unlock()

					// flush() blocks until successfully started a goroutine for flushing.
					qb.flush(batchToFlush)
					qb.resetTimer()
				} else {
					qb.currentBatchMu.Unlock()
				}
			}
		}
	}()
}

// startTimeBasedFlushingGoroutine starts a goroutine that flushes on timeout.
func (qb *DefaultBatcher[K]) startTimeBasedFlushingGoroutine() {
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
func (qb *DefaultBatcher[K]) Start(_ context.Context, _ component.Host) error {
	// maxWorker being -1 means batcher is disabled. This is for testing queue sender metrics.
	if qb.maxWorkers == -1 {
		return nil
	}

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
func (qb *DefaultBatcher[K]) flushCurrentBatchIfNecessary() {
	qb.currentBatchMu.Lock()
	if qb.currentBatch == nil {
		qb.currentBatchMu.Unlock()
		return
	}
	batchToFlush := *qb.currentBatch
	qb.currentBatch = nil
	qb.currentBatchMu.Unlock()

	// flush() blocks until successfully started a goroutine for flushing.
	qb.flush(batchToFlush)
	qb.resetTimer()
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *DefaultBatcher[K]) Shutdown(_ context.Context) error {
	qb.flushCurrentBatchIfNecessary()
	qb.stopWG.Wait()
	return nil
}
