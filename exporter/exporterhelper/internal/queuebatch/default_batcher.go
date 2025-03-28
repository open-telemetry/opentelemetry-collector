// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

type batcherSettings[T any] struct {
	sizerType   request.SizerType
	sizer       request.Sizer[T]
	partitioner Partitioner[T]
	next        sender.SendFunc[T]
	maxWorkers  int
}

// defaultBatcher continuously batch incoming requests and flushes asynchronously if minimum size limit is met or on timeout.
type defaultBatcher struct {
	cfg         BatchConfig
	workerPool  chan struct{}
	sizerType   request.SizerType
	sizer       request.Sizer[request.Request]
	consumeFunc sender.SendFunc[request.Request]
	stopWG      sync.WaitGroup
	ticker      *time.Ticker
	shutdownCh  chan struct{}

	partitionManager partitionManager
}

func newDefaultBatcher(bCfg BatchConfig, bSet batcherSettings[request.Request]) *defaultBatcher {
	// TODO: Determine what is the right behavior for this in combination with async queue.
	var workerPool chan struct{}
	if bSet.maxWorkers != 0 {
		workerPool = make(chan struct{}, bSet.maxWorkers)
		for i := 0; i < bSet.maxWorkers; i++ {
			workerPool <- struct{}{}
		}
	}
	return &defaultBatcher{
		cfg:              bCfg,
		workerPool:       workerPool,
		sizerType:        bSet.sizerType,
		sizer:            bSet.sizer,
		consumeFunc:      bSet.next,
		stopWG:           sync.WaitGroup{},
		shutdownCh:       make(chan struct{}, 1),
		partitionManager: newPartitionManager(bSet.partitioner),
	}
}

func (qb *defaultBatcher) Consume(ctx context.Context, req request.Request, done Done) {
	shard := qb.partitionManager.getShard(ctx, req)
	shard.Lock()

	if shard.batch == nil {
		reqList, mergeSplitErr := req.MergeSplit(ctx, int(qb.cfg.MaxSize), qb.sizerType, nil)
		if mergeSplitErr != nil || len(reqList) == 0 {
			done.OnDone(mergeSplitErr)
			shard.Unlock()
			return
		}

		// If more than one flush is required for this request, call done only when all flushes are done.
		if len(reqList) > 1 {
			done = newRefCountDone(done, int64(len(reqList)))
		}

		// We have at least one result in the reqList. Last in the list may not have enough data to be flushed.
		// Find if it has at least MinSize, and if it does then move that as the current batch.
		lastReq := reqList[len(reqList)-1]
		if qb.sizer.Sizeof(lastReq) < qb.cfg.MinSize {
			// Do not flush the last item and add it to the current batch.
			reqList = reqList[:len(reqList)-1]
			shard.batch = &batch{
				ctx:     ctx,
				req:     lastReq,
				done:    multiDone{done},
				created: time.Now(),
			}
		}

		shard.Unlock()
		for i := 0; i < len(reqList); i++ {
			qb.flush(ctx, reqList[i], done)
		}

		return
	}

	reqList, mergeSplitErr := shard.batch.req.MergeSplit(ctx, int(qb.cfg.MaxSize), qb.sizerType, req)
	// If failed to merge signal all Done callbacks from current batch as well as the current request and reset the current batch.
	if mergeSplitErr != nil || len(reqList) == 0 {
		done.OnDone(mergeSplitErr)
		shard.Unlock()
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
	shard.req = reqList[0]
	shard.done = append(shard.done, done)
	// Save the "currentBatch" if we need to flush it, because we want to execute flush without holding the lock, and
	// cannot unlock and re-lock because we are not done processing all the responses.
	var firstBatch *batch
	// Need to check the currentBatch if more than 1 result returned or if 1 result return but larger than MinSize.
	if len(reqList) > 1 || qb.sizer.Sizeof(shard.batch.req) >= qb.cfg.MinSize {
		firstBatch = shard.batch
		shard.batch = nil
	}
	// At this moment we dealt with the first result which is iter in the currentBatch or in the `firstBatch` we will flush.
	reqList = reqList[1:]

	// If we still have results to process, then we need to check if the last result has enough data to flush, or we add it to the currentBatch.
	if len(reqList) > 0 {
		lastReq := reqList[len(reqList)-1]
		if qb.sizer.Sizeof(lastReq) < qb.cfg.MinSize {
			// Do not flush the last item and add it to the current batch.
			reqList = reqList[:len(reqList)-1]
			shard.batch = &batch{
				ctx:     ctx,
				req:     lastReq,
				done:    multiDone{done},
				created: time.Now(),
			}
		}
	}

	shard.Unlock()
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
			case <-qb.ticker.C:
				qb.flushCurrentBatchIfNecessary(false)
			}
		}
	}()
}

// Start starts the goroutine that reads from the queue and flushes asynchronously.
func (qb *defaultBatcher) Start(_ context.Context, _ component.Host) error {
	if qb.cfg.FlushTimeout > 0 {
		qb.ticker = time.NewTicker(qb.cfg.FlushTimeout)
		qb.startTimeBasedFlushingGoroutine()
	}

	return nil
}

// flushCurrentBatchIfNecessary sends out the current request batch if it is not nil
func (qb *defaultBatcher) flushCurrentBatchIfNecessary(forceFlush bool) {
	qb.partitionManager.forEachShard(func(shard *shard) {
		shard.Lock()
		if shard.batch == nil {
			shard.Unlock()
			return
		}
		if !forceFlush && time.Since(shard.created) < qb.cfg.FlushTimeout {
			shard.Unlock()
			return
		}
		batchToFlush := shard.batch
		shard.batch = nil
		shard.Unlock()

		// flush() blocks until successfully started a goroutine for flushing.
		qb.flush(batchToFlush.ctx, batchToFlush.req, batchToFlush.done)
	})
}

// flush starts a goroutine that calls consumeFunc. It blocks until a worker is available if necessary.
func (qb *defaultBatcher) flush(ctx context.Context, req request.Request, done Done) {
	qb.stopWG.Add(1)
	if qb.workerPool != nil {
		<-qb.workerPool
	}
	go func() {
		defer qb.stopWG.Done()
		done.OnDone(qb.consumeFunc(ctx, req))
		if qb.workerPool != nil {
			qb.workerPool <- struct{}{}
		}
	}()
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *defaultBatcher) Shutdown(_ context.Context) error {
	close(qb.shutdownCh)
	// Make sure execute one last flush if necessary.
	qb.flushCurrentBatchIfNecessary(true)
	qb.stopWG.Wait()
	return nil
}

type multiDone []Done

func (mdc multiDone) OnDone(err error) {
	for _, d := range mdc {
		d.OnDone(err)
	}
}

type refCountDone struct {
	done     Done
	mu       sync.Mutex
	refCount int64
	err      error
}

func newRefCountDone(done Done, refCount int64) Done {
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
