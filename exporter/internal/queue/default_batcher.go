// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal"
)

type batch struct {
	ctx   context.Context
	req   internal.Request
	dones []exporterqueue.DoneCallback
}

// defaultBatcher continuously batch incoming requests and flushes asynchronously if minimum size limit is met or on timeout.
type defaultBatcher struct {
	batchCfg       exporterbatcher.Config
	workerPool     chan struct{}
	exportFunc     func(ctx context.Context, req internal.Request) error
	stopWG         sync.WaitGroup
	currentBatchMu sync.Mutex
	currentBatch   *batch
	timer          *time.Timer
	shutdownCh     chan struct{}
}

func newDefaultBatcher(batchCfg exporterbatcher.Config,
	exportFunc func(ctx context.Context, req internal.Request) error,
	maxWorkers int,
) *defaultBatcher {
	workerPool := make(chan struct{}, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workerPool <- struct{}{}
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

func (qb *defaultBatcher) Consume(ctx context.Context, req internal.Request, done exporterqueue.DoneCallback) {
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
			done(mergeSplitErr)
			qb.currentBatchMu.Unlock()
			return
		}

		// If there was a split, we flush everything immediately.
		if reqList[0].ItemsCount() >= qb.batchCfg.MinSizeItems || len(reqList) > 1 {
			qb.currentBatch = nil
			qb.currentBatchMu.Unlock()
			for i := 0; i < len(reqList); i++ {
				qb.flush(ctx, reqList[i], []exporterqueue.DoneCallback{done})
				// TODO: handle partial failure
			}
			qb.resetTimer()
		} else {
			qb.currentBatch = &batch{
				req:   reqList[0],
				ctx:   ctx,
				dones: []exporterqueue.DoneCallback{done},
			}
			qb.currentBatchMu.Unlock()
		}
	} else {
		if qb.currentBatch == nil || qb.currentBatch.req == nil {
			qb.resetTimer()
			qb.currentBatch = &batch{
				req:   req,
				ctx:   ctx,
				dones: []exporterqueue.DoneCallback{done},
			}
		} else {
			// TODO: consolidate implementation for the cases where MaxSizeConfig is specified and the case where it is not specified
			mergedReq, mergeErr := qb.currentBatch.req.MergeSplit(qb.currentBatch.ctx, qb.batchCfg.MaxSizeConfig, req)
			if mergeErr != nil {
				done(mergeErr)
				qb.currentBatchMu.Unlock()
				return
			}
			qb.currentBatch = &batch{
				req:   mergedReq[0],
				ctx:   qb.currentBatch.ctx,
				dones: append(qb.currentBatch.dones, done),
			}
		}

		if qb.currentBatch.req.ItemsCount() >= qb.batchCfg.MinSizeItems {
			batchToFlush := *qb.currentBatch
			qb.currentBatch = nil
			qb.currentBatchMu.Unlock()

			// flush() blocks until successfully started a goroutine for flushing.
			qb.flush(batchToFlush.ctx, batchToFlush.req, batchToFlush.dones)
			qb.resetTimer()
		} else {
			qb.currentBatchMu.Unlock()
		}
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
	if qb.currentBatch == nil || qb.currentBatch.req == nil {
		qb.currentBatchMu.Unlock()
		return
	}
	batchToFlush := *qb.currentBatch
	qb.currentBatch = nil
	qb.currentBatchMu.Unlock()

	// flush() blocks until successfully started a goroutine for flushing.
	qb.flush(batchToFlush.ctx, batchToFlush.req, batchToFlush.dones)
	qb.resetTimer()
}

// flush starts a goroutine that calls exportFunc. It blocks until a worker is available if necessary.
func (qb *defaultBatcher) flush(ctx context.Context, req internal.Request, dones []exporterqueue.DoneCallback) {
	qb.stopWG.Add(1)
	<-qb.workerPool
	go func() {
		defer qb.stopWG.Done()
		err := qb.exportFunc(ctx, req)
		for _, done := range dones {
			done(err)
		}
		qb.workerPool <- struct{}{}
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
