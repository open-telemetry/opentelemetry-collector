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

type batch struct {
	ctx  context.Context
	req  request.Request
	done multiDone
}

type batcherSettings[T any] struct {
	sizerType   request.SizerType
	sizer       request.Sizer[T]
	partitioner Partitioner[T]
	next        sender.SendFunc[T]
	maxWorkers  int
}

type shard struct {
	cfg            BatchConfig
	workerPool     *chan struct{}
	sizerType      request.SizerType
	sizer          request.Sizer[request.Request]
	consumeFunc    sender.SendFunc[request.Request]
	stopWG         sync.WaitGroup
	currentBatchMu sync.Mutex
	currentBatch   *batch
	timer          *time.Timer
	shutdownCh     chan struct{}
}

func newShard(bCfg BatchConfig, bSet batcherSettings[request.Request], workerPool *chan struct{}) *shard {
	return &shard{
		cfg:         bCfg,
		workerPool:  workerPool,
		sizerType:   bSet.sizerType,
		sizer:       bSet.sizer,
		consumeFunc: bSet.next,
		stopWG:      sync.WaitGroup{},
		shutdownCh:  make(chan struct{}, 1),
	}
}

func (s *shard) resetTimer() {
	if s.cfg.FlushTimeout > 0 {
		s.timer.Reset(s.cfg.FlushTimeout)
	}
}

func (s *shard) Consume(ctx context.Context, req request.Request, done Done) {
	s.currentBatchMu.Lock()

	if s.currentBatch == nil {
		reqList, mergeSplitErr := req.MergeSplit(ctx, int(s.cfg.MaxSize), s.sizerType, nil)
		if mergeSplitErr != nil || len(reqList) == 0 {
			done.OnDone(mergeSplitErr)
			s.currentBatchMu.Unlock()
			return
		}

		// If more than one flush is required for this request, call done only when all flushes are done.
		if len(reqList) > 1 {
			done = newRefCountDone(done, int64(len(reqList)))
		}

		// We have at least one result in the reqList. Last in the list may not have enough data to be flushed.
		// Find if it has at least MinSize, and if it does then move that as the current batch.
		lastReq := reqList[len(reqList)-1]
		if s.sizer.Sizeof(lastReq) < s.cfg.MinSize {
			// Do not flush the last item and add it to the current batch.
			reqList = reqList[:len(reqList)-1]
			s.currentBatch = &batch{
				ctx:  ctx,
				req:  lastReq,
				done: multiDone{done},
			}
			s.resetTimer()
		}

		s.currentBatchMu.Unlock()
		for i := 0; i < len(reqList); i++ {
			s.flush(ctx, reqList[i], done)
		}

		return
	}

	reqList, mergeSplitErr := s.currentBatch.req.MergeSplit(ctx, int(s.cfg.MaxSize), s.sizerType, req)
	// If failed to merge signal all Done callbacks from current batch as well as the current request and reset the current batch.
	if mergeSplitErr != nil || len(reqList) == 0 {
		done.OnDone(mergeSplitErr)
		s.currentBatchMu.Unlock()
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
	s.currentBatch.req = reqList[0]
	s.currentBatch.done = append(s.currentBatch.done, done)
	s.currentBatch.ctx = contextWithMergedLinks(s.currentBatch.ctx, ctx)

	// Save the "currentBatch" if we need to flush it, because we want to execute flush without holding the lock, and
	// cannot unlock and re-lock because we are not done processing all the responses.
	var firstBatch *batch
	// Need to check the currentBatch if more than 1 result returned or if 1 result return but larger than MinSize.
	if len(reqList) > 1 || s.sizer.Sizeof(s.currentBatch.req) >= s.cfg.MinSize {
		firstBatch = s.currentBatch
		s.currentBatch = nil
	}
	// At this moment we dealt with the first result which is iter in the currentBatch or in the `firstBatch` we will flush.
	reqList = reqList[1:]

	// If we still have results to process, then we need to check if the last result has enough data to flush, or we add it to the currentBatch.
	if len(reqList) > 0 {
		lastReq := reqList[len(reqList)-1]
		if s.sizer.Sizeof(lastReq) < s.cfg.MinSize {
			// Do not flush the last item and add it to the current batch.
			reqList = reqList[:len(reqList)-1]
			s.currentBatch = &batch{
				ctx:  ctx,
				req:  lastReq,
				done: multiDone{done},
			}
			s.resetTimer()
		}
	}

	s.currentBatchMu.Unlock()
	if firstBatch != nil {
		s.flush(firstBatch.ctx, firstBatch.req, firstBatch.done)
	}
	for i := 0; i < len(reqList); i++ {
		s.flush(ctx, reqList[i], done)
	}
}

// startTimeBasedFlushingGoroutine starts a goroutine that flushes on timeout.
func (s *shard) startTimeBasedFlushingGoroutine() {
	s.stopWG.Add(1)
	go func() {
		defer s.stopWG.Done()
		for {
			select {
			case <-s.shutdownCh:
				return
			case <-s.timer.C:
				s.flushCurrentBatchIfNecessary()
			}
		}
	}()
}

// Start starts the goroutine that reads from the queue and flushes asynchronously.
func (s *shard) Start(_ context.Context, _ component.Host) error {
	if s.cfg.FlushTimeout > 0 {
		s.timer = time.NewTimer(s.cfg.FlushTimeout)
		s.startTimeBasedFlushingGoroutine()
	}

	return nil
}

// flushCurrentBatchIfNecessary sends out the current request batch if it is not nil
func (s *shard) flushCurrentBatchIfNecessary() {
	s.currentBatchMu.Lock()
	if s.currentBatch == nil {
		s.currentBatchMu.Unlock()
		return
	}
	batchToFlush := s.currentBatch
	s.currentBatch = nil
	s.currentBatchMu.Unlock()

	// flush() blocks until successfully started a goroutine for flushing.
	s.flush(batchToFlush.ctx, batchToFlush.req, batchToFlush.done)
	s.resetTimer()
}

// flush starts a goroutine that calls consumeFunc. It blocks until a worker is available if necessary.
func (s *shard) flush(ctx context.Context, req request.Request, done Done) {
	s.stopWG.Add(1)
	if s.workerPool != nil {
		<-*s.workerPool
	}
	go func() {
		defer s.stopWG.Done()
		done.OnDone(s.consumeFunc(ctx, req))
		if s.workerPool != nil {
			*s.workerPool <- struct{}{}
		}
	}()
}

// Shutdown ensures that queue and all Batcher are stopped.
func (s *shard) Shutdown(_ context.Context) error {
	close(s.shutdownCh)
	// Make sure execute one last flush if necessary.
	s.flushCurrentBatchIfNecessary()
	s.stopWG.Wait()
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
