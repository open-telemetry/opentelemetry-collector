// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

var _ Batcher[request.Request] = (*partitionBatcher)(nil)

type batch struct {
	ctx  context.Context
	req  request.Request
	done multiDone
}

// partitionBatcher continuously batch incoming requests and flushes asynchronously if minimum size limit is met or on timeout.
type partitionBatcher struct {
	cfg            BatchConfig
	wp             *workerPool
	sizer          request.Sizer[request.Request]
	consumeFunc    sender.SendFunc[request.Request]
	stopWG         sync.WaitGroup
	currentBatchMu sync.Mutex
	currentBatch   *batch
	timer          *time.Timer
	shutdownCh     chan struct{}
	logger         *zap.Logger
}

func newPartitionBatcher(
	cfg BatchConfig,
	sizer request.Sizer[request.Request],
	wp *workerPool,
	next sender.SendFunc[request.Request],
	logger *zap.Logger,
) *partitionBatcher {
	return &partitionBatcher{
		cfg:         cfg,
		wp:          wp,
		sizer:       sizer,
		consumeFunc: next,
		shutdownCh:  make(chan struct{}, 1),
		logger:      logger,
	}
}

func (qb *partitionBatcher) resetTimer() {
	if qb.cfg.FlushTimeout > 0 {
		qb.timer.Reset(qb.cfg.FlushTimeout)
	}
}

func (qb *partitionBatcher) Consume(ctx context.Context, req request.Request, done queue.Done) {
	qb.currentBatchMu.Lock()

	if qb.currentBatch == nil {
		reqList, mergeSplitErr := req.MergeSplit(ctx, int(qb.cfg.MaxSize), qb.cfg.Sizer, nil)
		if mergeSplitErr != nil {
			// Do not return in case of error if there are data, try to export as much as possible.
			qb.logger.Warn("Failed to split request.", zap.Error(mergeSplitErr))
		}

		if len(reqList) == 0 {
			done.OnDone(mergeSplitErr)
			qb.currentBatchMu.Unlock()
			return
		}

		// If more than one flush is required for this request, call done only when all flushes are done.
		numRefs := len(reqList)
		// Need to also inform about the mergeSplitErr, consider the errored data as 1 batch.
		if mergeSplitErr != nil {
			numRefs++
		}
		if numRefs > 1 {
			done = newRefCountDone(done, int64(numRefs))
			if mergeSplitErr != nil {
				done.OnDone(mergeSplitErr)
			}
		}

		// We have at least one result in the reqList. Last in the list may not have enough data to be flushed.
		// Find if it has at least MinSize, and if it does then move that as the current batch.
		lastReq := reqList[len(reqList)-1]
		if qb.sizer.Sizeof(lastReq) < qb.cfg.MinSize {
			// Do not flush the last item and add it to the current batch.
			reqList = reqList[:len(reqList)-1]
			qb.currentBatch = &batch{
				ctx:  ctx,
				req:  lastReq,
				done: multiDone{done},
			}
			qb.resetTimer()
		}

		qb.currentBatchMu.Unlock()
		for i := 0; i < len(reqList); i++ {
			qb.flush(ctx, reqList[i], done)
		}

		return
	}

	reqList, mergeSplitErr := qb.currentBatch.req.MergeSplit(ctx, int(qb.cfg.MaxSize), qb.cfg.Sizer, req)
	// If failed to merge signal all Done callbacks from the current batch as well as the current request and reset the current batch.
	if mergeSplitErr != nil {
		// Do not return in case of error if there are data, try to export as much as possible.
		qb.logger.Warn("Failed to split request.", zap.Error(mergeSplitErr))
	}

	if len(reqList) == 0 {
		done.OnDone(mergeSplitErr)
		qb.currentBatchMu.Unlock()
		return
	}

	// If more than one flush is required for this request, call done only when all flushes are done.
	numRefs := len(reqList)
	// Need to also inform about the mergeSplitErr, consider the errored data as 1 batch.
	if mergeSplitErr != nil {
		numRefs++
	}
	fmt.Printf("[DEBUG] Partition batcher flush analysis: reqList_len=%d, numRefs=%d, has_mergeSplitErr=%t, done_type=%T\n",
		len(reqList), numRefs, mergeSplitErr != nil, done)
	if numRefs > 1 {
		fmt.Printf("[DEBUG] Creating refCountDone: ref_count=%d, original_done_type=%T\n",
			len(reqList), done)
		done = newRefCountDone(done, int64(len(reqList)))
		fmt.Printf("[DEBUG] Created refCountDone: new_done_type=%T\n", done)
		if mergeSplitErr != nil {
			fmt.Printf("[DEBUG] Calling OnDone for mergeSplitErr: err=%v\n", mergeSplitErr)
			done.OnDone(mergeSplitErr)
		}
	}

	// We have at least one result in the reqList, if more results here is what that means:
	// - First result will contain items from the current batch + some results from the current request.
	// - All other results except first will contain items only from the current request.
	// - Last result may not have enough data to be flushed.

	// Logic on how to deal with the current batch:
	qb.currentBatch.req = reqList[0]
	qb.currentBatch.done = append(qb.currentBatch.done, done)
	qb.currentBatch.ctx = contextWithMergedLinks(qb.currentBatch.ctx, ctx)

	// Save the "currentBatch" if we need to flush it, because we want to execute flush without holding the lock, and
	// cannot unlock and re-lock because we are not done processing all the responses.
	var firstBatch *batch
	// Need to check the currentBatch if more than 1 result returned or if 1 result return but larger than MinSize.
	if len(reqList) > 1 || qb.sizer.Sizeof(qb.currentBatch.req) >= qb.cfg.MinSize {
		firstBatch = qb.currentBatch
		qb.currentBatch = nil
	}
	// At this moment we dealt with the first result which is iter in the currentBatch or in the `firstBatch` we will flush.
	reqList = reqList[1:]

	// If we still have results to process, then we need to check if the last result has enough data to flush, or we add it to the currentBatch.
	if len(reqList) > 0 {
		lastReq := reqList[len(reqList)-1]
		if qb.sizer.Sizeof(lastReq) < qb.cfg.MinSize {
			// Do not flush the last item and add it to the current batch.
			reqList = reqList[:len(reqList)-1]
			qb.currentBatch = &batch{
				ctx:  ctx,
				req:  lastReq,
				done: multiDone{done},
			}
			qb.resetTimer()
		}
	}

	qb.currentBatchMu.Unlock()
	if firstBatch != nil {
		qb.flush(firstBatch.ctx, firstBatch.req, firstBatch.done)
	}
	for i := 0; i < len(reqList); i++ {
		qb.flush(ctx, reqList[i], done)
	}
}

// Start starts the goroutine that reads from the queue and flushes asynchronously.
func (qb *partitionBatcher) Start(context.Context, component.Host) error {
	if qb.cfg.FlushTimeout <= 0 {
		return nil
	}
	qb.timer = time.NewTimer(qb.cfg.FlushTimeout)
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
	return nil
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *partitionBatcher) Shutdown(context.Context) error {
	close(qb.shutdownCh)
	// Make sure execute one last flush if necessary.
	qb.flushCurrentBatchIfNecessary()
	qb.stopWG.Wait()
	return nil
}

// flushCurrentBatchIfNecessary sends out the current request batch if it is not nil
func (qb *partitionBatcher) flushCurrentBatchIfNecessary() {
	qb.currentBatchMu.Lock()
	if qb.currentBatch == nil {
		qb.currentBatchMu.Unlock()
		return
	}
	batchToFlush := qb.currentBatch
	qb.currentBatch = nil
	qb.currentBatchMu.Unlock()

	// flush() blocks until successfully started a goroutine for flushing.
	qb.flush(batchToFlush.ctx, batchToFlush.req, batchToFlush.done)
	qb.resetTimer()
}

// flush starts a goroutine that calls consumeFunc. It blocks until a worker is available if necessary.
func (qb *partitionBatcher) flush(ctx context.Context, req request.Request, done queue.Done) {
	fmt.Printf("[DEBUG] Worker starting flush: done_type=%T, done_ptr=%p\n", done, done)
	qb.stopWG.Add(1)
	qb.wp.execute(func() {
		defer qb.stopWG.Done()
		fmt.Printf("[DEBUG] Worker executing consumeFunc: done_type=%T, done_ptr=%p\n", done, done)
		consumeErr := qb.consumeFunc(ctx, req)
		fmt.Printf("[DEBUG] Worker consumeFunc completed, calling OnDone: err=%v, done_type=%T, done_ptr=%p\n",
			consumeErr, done, done)
		done.OnDone(consumeErr)
		fmt.Printf("[DEBUG] Worker OnDone completed: done_type=%T, done_ptr=%p\n", done, done)
	})
}

type workerPool struct {
	workers chan struct{}
}

func newWorkerPool(maxWorkers int) *workerPool {
	workers := make(chan struct{}, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workers <- struct{}{}
	}
	return &workerPool{workers: workers}
}

func (wp *workerPool) execute(f func()) {
	<-wp.workers
	go f()
	wp.workers <- struct{}{}
}

type multiDone []queue.Done

func (mdc multiDone) OnDone(err error) {
	for _, d := range mdc {
		d.OnDone(err)
	}
}

type refCountDone struct {
	done     queue.Done
	mu       sync.Mutex
	refCount int64
	err      error
}

func newRefCountDone(done queue.Done, refCount int64) queue.Done {
	rcd := &refCountDone{
		done:     done,
		refCount: refCount,
	}
	// Temporary debug logging
	fmt.Printf("[DEBUG] Created refCountDone: refCount=%d, rcd_ptr=%p, done_type=%T, done_ptr=%p\n",
		refCount, rcd, done, done)
	return rcd
}

func (rcd *refCountDone) OnDone(err error) {
	rcd.mu.Lock()
	defer rcd.mu.Unlock()

	// Temporary debug logging
	fmt.Printf("[DEBUG] refCountDone.OnDone called: err=%v, current_refCount=%d, rcd_ptr=%p, done_type=%T, done_ptr=%p\n",
		err, rcd.refCount, rcd, rcd.done, rcd.done)

	rcd.err = multierr.Append(rcd.err, err)
	rcd.refCount--

	fmt.Printf("[DEBUG] refCountDone decremented: new_refCount=%d, rcd_ptr=%p\n",
		rcd.refCount, rcd)

	if rcd.refCount == 0 {
		// No more references, call done.
		fmt.Printf("[DEBUG] refCountDone calling wrapped done.OnDone: err=%v, done_type=%T, done_ptr=%p\n",
			rcd.err, rcd.done, rcd.done)
		rcd.done.OnDone(rcd.err)
		fmt.Printf("[DEBUG] refCountDone completed wrapped done.OnDone\n")
	}
}
