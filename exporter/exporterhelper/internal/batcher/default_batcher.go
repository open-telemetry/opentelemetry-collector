// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

type batch struct {
	ctx  context.Context
	req  request.Request
	done multiDone
}

// defaultBatcher continuously batch incoming requests and flushes asynchronously if minimum size limit is met or on timeout.
type defaultBatcher struct {
	batchCfg       exporterbatcher.Config
	workerPool     chan struct{}
	exportFunc     func(ctx context.Context, req request.Request) error
	stopWG         sync.WaitGroup
	currentBatchMu sync.Mutex
	currentBatch   *batch
	timer          *time.Timer
	shutdownCh     chan struct{}
}

func newDefaultBatcher(batchCfg exporterbatcher.Config,
	exportFunc func(ctx context.Context, req request.Request) error,
	maxWorkers int,
) *defaultBatcher {
	// TODO: Determine what is the right behavior for this in combination with async queue.
	var workerPool chan struct{}
	if maxWorkers != 0 {
		workerPool = make(chan struct{}, maxWorkers)
		for i := 0; i < maxWorkers; i++ {
			workerPool <- struct{}{}
		}
	}
	return &defaultBatcher{
		batchCfg:   batchCfg,
		workerPool: workerPool,
		exportFunc: exportFunc,
		stopWG:     sync.WaitGroup{},
		shutdownCh: make(chan struct{}, 1),
	}
}

func (qb *defaultBatcher) resetTimer() {
	if qb.batchCfg.FlushTimeout != 0 {
		qb.timer.Reset(qb.batchCfg.FlushTimeout)
	}
}

func (qb *defaultBatcher) Consume(ctx context.Context, req request.Request, done exporterqueue.Done) {
	qb.currentBatchMu.Lock()

	if qb.currentBatch == nil {
		reqList, mergeSplitErr := req.MergeSplit(ctx, qb.batchCfg.SizeConfig, nil)
		if mergeSplitErr != nil || len(reqList) == 0 {
			done.OnDone(mergeSplitErr)
			qb.currentBatchMu.Unlock()
			return
		}

		// If more than one flush is required for this request, call done only when all flushes are done.
		if len(reqList) > 1 {
			done = newRefCountDone(done, int64(len(reqList)))
		}

		// We have at least one result in the reqList. Last in the list may not have enough data to be flushed.
		// Find if it has at least MinSize, and if it does then move that as the current batch.
		lastReq := reqList[len(reqList)-1]
		if lastReq.ItemsCount() < qb.batchCfg.MinSize {
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

	reqList, mergeSplitErr := qb.currentBatch.req.MergeSplit(ctx, qb.batchCfg.SizeConfig, req)
	// If failed to merge signal all Done callbacks from current batch as well as the current request and reset the current batch.
	if mergeSplitErr != nil || len(reqList) == 0 {
		done.OnDone(mergeSplitErr)
		qb.currentBatchMu.Unlock()
		return
	}

	// If more than one flush is required for this request, call done only when all flushes are done.
	if len(reqList) > 1 {
		done = newRefCountDone(done, int64(len(reqList)))
	}

	// We have at least one result in the reqList, if more results here is what that means:
	// - First result will contain items from the current batch + some results from the current request.
	// - All other results except first will contain items only from current request.
	// - Last result may not have enough data to be flushed.

	// Logic on how to deal with the current batch:
	// TODO: Deal with merging Context.
	qb.currentBatch.req = reqList[0]
	qb.currentBatch.done = append(qb.currentBatch.done, done)
	// Save the "currentBatch" if we need to flush it, because we want to execute flush without holding the lock, and
	// cannot unlock and re-lock because we are not done processing all the responses.
	var firstBatch *batch
	// Need to check the currentBatch if more than 1 result returned or if 1 result return but larger than MinSize.
	if len(reqList) > 1 || qb.currentBatch.req.ItemsCount() >= qb.batchCfg.MinSize {
		firstBatch = qb.currentBatch
		qb.currentBatch = nil
	}
	// At this moment we dealt with the first result which is iter in the currentBatch or in the `firstBatch` we will flush.
	reqList = reqList[1:]

	// If we still have results to process, then we need to check if the last result has enough data to flush, or we add it to the currentBatch.
	if len(reqList) > 0 {
		lastReq := reqList[len(reqList)-1]
		if lastReq.ItemsCount() < qb.batchCfg.MinSize {
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

// startTimeBasedFlushingGoroutine starts a goroutine that flushes on timeout.
func (qb *defaultBatcher) startTimeBasedFlushingGoroutine() {
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
func (qb *defaultBatcher) Start(_ context.Context, _ component.Host) error {
	if qb.batchCfg.FlushTimeout != 0 {
		qb.timer = time.NewTimer(qb.batchCfg.FlushTimeout)
		qb.startTimeBasedFlushingGoroutine()
	}

	return nil
}

// flushCurrentBatchIfNecessary sends out the current request batch if it is not nil
func (qb *defaultBatcher) flushCurrentBatchIfNecessary() {
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

// flush starts a goroutine that calls exportFunc. It blocks until a worker is available if necessary.
func (qb *defaultBatcher) flush(ctx context.Context, req request.Request, done exporterqueue.Done) {
	qb.stopWG.Add(1)
	if qb.workerPool != nil {
		<-qb.workerPool
	}
	go func() {
		defer qb.stopWG.Done()
		done.OnDone(qb.exportFunc(ctx, req))
		if qb.workerPool != nil {
			qb.workerPool <- struct{}{}
		}
	}()
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *defaultBatcher) Shutdown(_ context.Context) error {
	close(qb.shutdownCh)
	// Make sure execute one last flush if necessary.
	qb.flushCurrentBatchIfNecessary()
	qb.stopWG.Wait()
	return nil
}

type multiDone []exporterqueue.Done

func (mdc multiDone) OnDone(err error) {
	for _, d := range mdc {
		d.OnDone(err)
	}
}

type refCountDone struct {
	done     exporterqueue.Done
	mu       sync.Mutex
	refCount int64
	err      error
}

func newRefCountDone(done exporterqueue.Done, refCount int64) exporterqueue.Done {
	return &refCountDone{
		done:     done,
		refCount: refCount,
	}
}

func (rcd *refCountDone) OnDone(err error) {
	rcd.mu.Lock()
	defer rcd.mu.Unlock()
	rcd.err = multierr.Append(rcd.err, err)
	rcd.refCount--
	if rcd.refCount == 0 {
		// No more references, call done.
		rcd.done.OnDone(rcd.err)
	}
}
