package exporterhelper

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
)

type batcher struct {
	mu             sync.Mutex
	activeBatch    *batch
	cfg            exporterbatcher.Config
	mergeFunc      exporterbatcher.BatchMergeFunc[Request]
	mergeSplitFunc exporterbatcher.BatchMergeSplitFunc[Request]
	activeRequests atomic.Int64
	exportCtx      context.Context
}

func newBatcher(cfg exporterbatcher.Config, mf exporterbatcher.BatchMergeFunc[Request], msf exporterbatcher.BatchMergeSplitFunc[Request]) *batcher {
	return &batcher{
		activeBatch:    newEmptyBatch(),
		cfg:            cfg,
		mergeFunc:      mf,
		mergeSplitFunc: msf,
		exportCtx:      context.Background(),
	}
}

func (b *batcher) updateActiveBatch(ctx context.Context, req Request) {
	if b.activeBatch.request == nil {
		b.activeBatch.ctx = ctx
	}
	b.activeBatch.request = req
}

func (b *batcher) merge(ctx context.Context, req Request) ([]Request, error) {
	var reqs []Request
	var err error
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cfg.MaxSizeItems > 0 {
		reqs, err = b.mergeSplitFunc(ctx, b.cfg.MaxSizeConfig, b.activeBatch.request, req)
	} else {
		var mergedReq Request
		mergedReq, err = b.mergeFunc(ctx, b.activeBatch.request, req)
		reqs = []Request{mergedReq}
	}
	return reqs, err
}

func (bs *batchSender) startQueueBatchers() error {
	bs.batchers = make([]*batcher, bs.cfg.NumBatchers)
	for i := 0; i < bs.cfg.NumBatchers; i++ {
		bs.batchers[i] = newBatcher(bs.cfg, bs.mergeFunc, bs.mergeSplitFunc)

		go func(index int) {
			batcher := bs.batchers[index]
			for {
				ok := bs.queue.Consume(func(ctx context.Context, req Request) error {
					var err error
					batcher.activeRequests.Add(1)
					// take the request split it up if necessary and send all requests
					reqs, err := bs.batchers[index].merge(ctx, req)
					if err != nil || len(reqs) == 0 || reqs[0] == nil {
						batcher.activeRequests.Add(-1)
						return err
					}

					batcher.mu.Lock()
					batcher.updateActiveBatch(ctx, reqs[0])

					if len(reqs) > 1 || reqs[0].ItemsCount() >= bs.cfg.MinSizeItems {
						err = bs.nextSender.send(ctx, reqs...)
						batcher.activeRequests.Store(0)
						batcher.activeBatch = newEmptyBatch()
						bs.lastFlushed = time.Now()
					} else {
						// If using persistant queue, make sure items are not marked for deletion until the batch is sent.
						ctx = exporterbatcher.SetBatchingKeyInContext(ctx)
					}
					batcher.mu.Unlock()
					return err
				})

				if !ok {
					// queue is stopped
					return
				}
			}
		}(i)
	}

	go bs.startMultiBatchFlushLoop()

	return nil
}

func (bs *batchSender) shutdownAllBatchers(timer *time.Timer) {
	for _, b := range bs.batchers {
		for b.activeRequests.Load() > 0 {
			b.mu.Lock()
			if b.activeBatch.request != nil {
				bs.nextSender.send(b.activeBatch.ctx, b.activeBatch.request)
			}
			b.activeRequests.Add(-1)
			b.mu.Unlock()
		}
		if !timer.Stop() {
			<-timer.C
		}
	}
	close(bs.shutdownCompleteCh)
}

func (bs *batchSender) flushAllBatchers(timer *time.Timer) {
	for i := range bs.batchers {
		bs.batchers[i].mu.Lock()
		if bs.batchers[i].activeBatch.request != nil {
			bs.nextSender.send(bs.batchers[i].activeBatch.ctx, bs.batchers[i].activeBatch.request)
			bs.batchers[i].activeRequests.Store(0)
			bs.batchers[i].activeBatch = newEmptyBatch()
			bs.lastFlushed = time.Now()
		}
		bs.batchers[i].mu.Unlock()
	}
}

func (bs *batchSender) startMultiBatchFlushLoop() {
	bs.shutdownCh = make(chan struct{})
	timer := time.NewTimer(bs.cfg.FlushTimeout)
	// bs.flushAllBatchers(timer)
	for {
		select {
		case <-bs.shutdownCh:
			// There is a minimal chance that another request is added after the shutdown signal.
			// This loop will handle that case.
			bs.shutdownAllBatchers(timer)
			return
		case <-timer.C:
			bs.mu.Lock()
			nextFlush := bs.cfg.FlushTimeout
			// if bs.activeBatch.request != nil {
			sinceLastFlush := time.Since(bs.lastFlushed)
			if sinceLastFlush >= bs.cfg.FlushTimeout {
				bs.flushAllBatchers(timer)
			} else {
				nextFlush = bs.cfg.FlushTimeout - sinceLastFlush
			}
			// }
			bs.mu.Unlock()
			timer.Reset(nextFlush)
		}
	}
}
