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
	ctx     context.Context
	req     internal.Request
	idxList []uint64
}

type Batcher struct {
	cfg exporterbatcher.Config

	queue      Queue[internal.Request]
	maxWorkers int
	workerPool chan bool

	exportFunc func(context.Context, internal.Request) error

	batchListMu sync.Mutex
	batchList   []batch
	lastFlushed time.Time
	timer       *time.Timer
	shutdownCh  chan bool

	stopWG sync.WaitGroup
}

func NewBatcher(cfg exporterbatcher.Config, queue Queue[internal.Request],
	maxWorkers int, exportFunc func(context.Context, internal.Request) error) *Batcher {
	return &Batcher{
		cfg:        cfg,
		queue:      queue,
		maxWorkers: maxWorkers,
		exportFunc: exportFunc,
		stopWG:     sync.WaitGroup{},
		batchList:  make([]batch, 0),
		shutdownCh: make(chan bool, 1),
	}
}

// If precondition.s pass, flushIfNecessary() take an item from the head of batch list and exports it.
func (qb *Batcher) flushIfNecessary(timeout bool, force bool) {
	qb.batchListMu.Lock()

	if len(qb.batchList) == 0 || qb.batchList[0].req == nil {
		qb.resetTimer()
		qb.batchListMu.Unlock()
		return
	}

	if !force && timeout && time.Since(qb.lastFlushed) < qb.cfg.FlushTimeout {
		qb.batchListMu.Unlock()
		return
	}

	// If item size is over the threshold, there is another flusher that is already triggered.
	if !force && timeout && qb.batchList[0].req.ItemsCount() >= qb.cfg.MinSizeItems {
		qb.batchListMu.Unlock()
		return
	}

	if !force && !timeout && qb.batchList[0].req.ItemsCount() < qb.cfg.MinSizeItems {
		qb.batchListMu.Unlock()
		return
	}

	flushedBatch := qb.batchList[0]
	qb.batchList = qb.batchList[1:]
	qb.lastFlushed = time.Now()
	qb.resetTimer()

	qb.batchListMu.Unlock()
	err := qb.exportFunc(flushedBatch.ctx, flushedBatch.req)
	for _, idx := range flushedBatch.idxList {
		qb.queue.OnProcessingFinished(idx, err)
	}
}

// push() adds a new item to the batchList.
func (qb *Batcher) push(req internal.Request, idx uint64) (int, error) {
	qb.batchListMu.Lock()
	defer qb.batchListMu.Unlock()

	// If batching is not enabled.
	if !qb.cfg.Enabled {
		qb.batchList = append(
			qb.batchList,
			batch{
				req:     req,
				ctx:     context.Background(),
				idxList: []uint64{idx}})
		return 1, nil
	}

	if qb.cfg.MaxSizeItems > 0 {
		if len(qb.batchList) == 0 {
			qb.batchList = append(qb.batchList, batch{
				req:     nil,
				ctx:     context.Background(),
				idxList: []uint64{idx}})
		}

		var (
			reqs []internal.Request
			err  error
		)
		if qb.batchList[len(qb.batchList)-1].req == nil {
			reqs, err = req.MergeSplit(context.Background(), qb.cfg.MaxSizeConfig, nil)
		} else {
			reqs, err = qb.batchList[len(qb.batchList)-1].req.MergeSplit(context.Background(),
				qb.cfg.MaxSizeConfig, req)
		}

		if err != nil || len(reqs) == 0 {
			return 0, err
		}

		qb.batchList[len(qb.batchList)-1] = batch{
			req:     reqs[0],
			ctx:     context.Background(),
			idxList: []uint64{idx},
		}
		for i := 1; i < len(reqs); i++ {
			qb.batchList = append(qb.batchList, batch{
				req:     reqs[i],
				ctx:     context.Background(),
				idxList: []uint64{idx}})
		}

		// Force flush the last batch if there was a split.
		if len(reqs) > 1 && qb.batchList[len(qb.batchList)-1].req.ItemsCount() < qb.cfg.MinSizeItems {
			lastBatch := qb.batchList[len(qb.batchList)-1]
			qb.batchList = qb.batchList[0 : len(qb.batchList)-1]

			err := qb.exportFunc(lastBatch.ctx, lastBatch.req)
			for _, idx := range lastBatch.idxList {
				qb.queue.OnProcessingFinished(idx, err)
			}

			return len(reqs) - 1, nil
		}
		return len(reqs), nil
	}

	if len(qb.batchList) == 0 {
		qb.batchList = append(
			qb.batchList,
			batch{
				req:     req,
				ctx:     context.Background(),
				idxList: []uint64{idx}})
	} else {
		lastItem := len(qb.batchList) - 1
		mergedReq, err := qb.batchList[lastItem].req.Merge(context.Background(), req)
		if err != nil {
			return 0, err
		}
		qb.batchList[lastItem].req = mergedReq
		qb.batchList[lastItem].idxList = append(qb.batchList[lastItem].idxList, idx)
	}
	if qb.batchList[len(qb.batchList)-1].req.ItemsCount() >= qb.cfg.MinSizeItems {
		return 1, nil
	}
	return 0, nil

}

func (qb *Batcher) resetTimer() {
	if qb.cfg.Enabled {
		qb.timer.Reset(qb.cfg.FlushTimeout)
	}
}

// allocateFlusher() starts a goroutine that calls flushIfNecessary(). It blocks until a worker is available.
func (qb *Batcher) allocateFlusher(timeout bool) {
	// maxWorker = 0 means we don't limit the number of flushers.
	if qb.maxWorkers == 0 {
		qb.stopWG.Add(1)
		go func() {
			qb.flushIfNecessary(timeout /*timeout*/, false /*force*/)
			qb.stopWG.Done()
		}()
		return
	}

	qb.stopWG.Add(1)
	<-qb.workerPool
	go func() {
		qb.flushIfNecessary(timeout /*timeout*/, false /*force*/)
		qb.workerPool <- true
		qb.stopWG.Done()
	}()
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

	qb.workerPool = make(chan bool, qb.maxWorkers)
	for i := 0; i < qb.maxWorkers; i++ {
		qb.workerPool <- true
	}

	// This goroutine keeps reading until flush is triggered because of request size.
	qb.stopWG.Add(1)
	go func() {
		defer qb.stopWG.Done()
		for {
			idx, _, req, ok := qb.queue.Read(context.Background())

			if !ok {
				qb.shutdownCh <- true
				qb.flushIfNecessary(false /*timeout*/, true /*force*/)
				return
			}

			flushCount, err := qb.push(req, idx)
			if err != nil {
				qb.queue.OnProcessingFinished(idx, err)
			}

			if flushCount > 0 {
				for i := 1; i < flushCount; i++ {
					qb.allocateFlusher(false /*timeout*/)
				}
				// allocateFlusher() blocks until the number of flushers are under the threshold.
				// This ensures that reading slows down when we are too busy sending.
				qb.allocateFlusher(false /*timeout*/)
			}
		}
	}()

	qb.stopWG.Add(1)
	go func() {
		defer qb.stopWG.Done()
		for {
			select {
			case <-qb.shutdownCh:
				return
			case <-qb.timer.C:
				qb.allocateFlusher(true /*timeout*/)
			}
		}
	}()
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
