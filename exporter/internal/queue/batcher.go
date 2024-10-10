// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
)

type batch struct {
	ctx              context.Context
	req              internal.Request
	onExportFinished func(error)
}

type Batcher struct {
	cfg            exporterbatcher.Config
	mergeFunc      exporterbatcher.BatchMergeFunc[internal.Request]
	mergeSplitFunc exporterbatcher.BatchMergeSplitFunc[internal.Request]

	queue      Queue[internal.Request]
	numWorkers int
	exportFunc func(context.Context, internal.Request) error
	stopWG     sync.WaitGroup

	mu             sync.Mutex
	lastFlushed    time.Time
	pendingBatches []batch
	timer          *time.Timer
	shutdownCh     chan bool
}

func NewBatcher(cfg exporterbatcher.Config, queue Queue[internal.Request],
	numWorkers int, exportFunc func(context.Context, internal.Request) error,
	mergeFunc exporterbatcher.BatchMergeFunc[internal.Request],
	mergeSplitFunc exporterbatcher.BatchMergeSplitFunc[internal.Request]) *Batcher {
	return &Batcher{
		cfg:            cfg,
		queue:          queue,
		numWorkers:     max(numWorkers, 1),
		exportFunc:     exportFunc,
		stopWG:         sync.WaitGroup{},
		pendingBatches: make([]batch, 0),
		mergeFunc:      mergeFunc,
		mergeSplitFunc: mergeSplitFunc,
	}
}

func (qb *Batcher) flushIfNecessary(forceFlush bool) {
	qb.mu.Lock()

	if len(qb.pendingBatches) == 0 || qb.pendingBatches[0].req == nil {
		qb.mu.Unlock()
		return
	}

	if !forceFlush && time.Since(qb.lastFlushed) < qb.cfg.FlushTimeout && qb.pendingBatches[0].req.ItemsCount() < qb.cfg.MinSizeItems {
		qb.mu.Unlock()
		return
	}

	flushedBatch := qb.pendingBatches[0]
	qb.pendingBatches = qb.pendingBatches[1:]
	qb.lastFlushed = time.Now()

	if qb.cfg.Enabled {
		qb.timer.Reset(qb.cfg.FlushTimeout)
	}

	qb.mu.Unlock()
	err := qb.exportFunc(flushedBatch.ctx, flushedBatch.req)
	if flushedBatch.onExportFinished != nil {
		flushedBatch.onExportFinished(err)
	}
}

func (qb *Batcher) push(req internal.Request, onExportFinished func(error)) (int, error) {
	qb.mu.Lock()
	defer qb.mu.Unlock()

	// If batching is not enabled.
	if !qb.cfg.Enabled {
		qb.pendingBatches = append(
			qb.pendingBatches,
			batch{
				req:              req,
				ctx:              context.Background(),
				onExportFinished: onExportFinished})
		return 1, nil
	}

	if qb.cfg.MaxSizeItems > 0 {
		if len(qb.pendingBatches) == 0 {
			qb.pendingBatches = append(qb.pendingBatches, batch{
				req:              nil,
				ctx:              context.Background(),
				onExportFinished: onExportFinished})
		}

		lastItem := len(qb.pendingBatches) - 1
		reqs, err := qb.mergeSplitFunc(context.Background(),
			qb.cfg.MaxSizeConfig,
			qb.pendingBatches[lastItem].req, req)

		if err != nil || len(reqs) == 0 {
			return 0, err
		}

		for offset, newReq := range reqs {
			if offset != 0 {
				qb.pendingBatches = append(qb.pendingBatches, batch{})
			}
			qb.pendingBatches[lastItem+offset].req = newReq
			qb.pendingBatches[lastItem+offset].ctx = context.Background()
			qb.pendingBatches[lastItem+offset].onExportFinished = onExportFinished
		}

		if reqs[len(reqs)-1].ItemsCount() > qb.cfg.MinSizeItems {
			return len(reqs), nil
		}
		return len(reqs) - 1, nil
	}

	if len(qb.pendingBatches) == 0 {
		qb.pendingBatches = append(
			qb.pendingBatches,
			batch{
				req:              req,
				ctx:              context.Background(),
				onExportFinished: onExportFinished})
	} else {
		lastItem := len(qb.pendingBatches) - 1
		mergedReq, err := qb.mergeFunc(context.Background(), qb.pendingBatches[lastItem].req, req)
		if err != nil {
			return 0, err
		}
		qb.pendingBatches[lastItem].req = mergedReq
	}
	if qb.pendingBatches[len(qb.pendingBatches)-1].req.ItemsCount() > qb.cfg.MinSizeItems {
		return 1, nil
	}
	return 0, nil

}

// Start ensures that queue and all consumers are started.
func (qb *Batcher) Start(ctx context.Context, host component.Host) error {
	if err := qb.queue.Start(ctx, host); err != nil {
		return err
	}

	if qb.cfg.Enabled {
		qb.timer = time.NewTimer(qb.cfg.FlushTimeout)
	} else {
		qb.timer = time.NewTimer(math.MaxInt)
		qb.timer.Stop()
	}

	flushCh := make(chan bool, qb.numWorkers)
	qb.shutdownCh = make(chan bool, qb.numWorkers)

	var startWG sync.WaitGroup

	qb.stopWG.Add(1)
	startWG.Add(1)
	go func() {
		startWG.Done()
		defer qb.stopWG.Done()
		for {
			req, ok, onProcessingFinished := qb.queue.Read()
			if !ok {
				for i := 0; i < qb.numWorkers; i++ {
					qb.shutdownCh <- true
				}
				return
			}
			flushCount, err := qb.push(req, onProcessingFinished) // Handle error
			if err != nil && onProcessingFinished != nil {
				onProcessingFinished(err)
			}
			for i := 0; i < flushCount; i++ {
				flushCh <- true
			}
		}
	}()

	for i := 0; i < qb.numWorkers; i++ {
		qb.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer qb.stopWG.Done()
			for {
				select {
				case <-flushCh:
					qb.flushIfNecessary(false)
				case <-qb.timer.C:
					qb.flushIfNecessary(false)
					if qb.cfg.Enabled {
						qb.timer.Reset(qb.cfg.FlushTimeout)
					}
				case <-qb.shutdownCh:
					qb.flushIfNecessary(true)
					return
				}
			}
		}()
	}
	startWG.Wait()

	return nil
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *Batcher) Shutdown(ctx context.Context) error {
	if err := qb.queue.Shutdown(ctx); err != nil {
		return err
	}
	qb.stopWG.Wait()
	return nil
}
