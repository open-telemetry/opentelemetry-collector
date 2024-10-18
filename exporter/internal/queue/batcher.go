// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
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
	cfg            exporterbatcher.Config
	mergeFunc      exporterbatcher.BatchMergeFunc[internal.Request]
	mergeSplitFunc exporterbatcher.BatchMergeSplitFunc[internal.Request]

	queue      Queue[internal.Request]
	maxWorkers int
	numWorkers atomic.Int64
	exportFunc func(context.Context, internal.Request) error

	batchListMu      sync.Mutex
	batchList        []batch
	lastFlushed      time.Time
	timer            *time.Timer
	shutdownCh       chan bool
	allocateWorkerCh chan bool

	stopWG sync.WaitGroup
}

func NewBatcher(cfg exporterbatcher.Config, queue Queue[internal.Request],
	maxWorkers int, exportFunc func(context.Context, internal.Request) error,
	mergeFunc exporterbatcher.BatchMergeFunc[internal.Request],
	mergeSplitFunc exporterbatcher.BatchMergeSplitFunc[internal.Request]) *Batcher {
	return &Batcher{
		cfg:              cfg,
		queue:            queue,
		maxWorkers:       maxWorkers,
		exportFunc:       exportFunc,
		stopWG:           sync.WaitGroup{},
		batchList:        make([]batch, 0),
		mergeFunc:        mergeFunc,
		mergeSplitFunc:   mergeSplitFunc,
		shutdownCh:       make(chan bool, 1),
		allocateWorkerCh: make(chan bool, 1),
	}
}

func (qb *Batcher) flushIfNecessary(forceFlush bool) {
	qb.batchListMu.Lock()

	if len(qb.batchList) == 0 || qb.batchList[0].req == nil {
		qb.batchListMu.Unlock()
		return
	}

	if !forceFlush && time.Since(qb.lastFlushed) < qb.cfg.FlushTimeout && qb.batchList[0].req.ItemsCount() < qb.cfg.MinSizeItems {
		qb.batchListMu.Unlock()
		return
	}

	flushedBatch := qb.batchList[0]
	qb.batchList = qb.batchList[1:]
	qb.lastFlushed = time.Now()

	if qb.cfg.Enabled {
		qb.timer.Reset(qb.cfg.FlushTimeout)
	}

	qb.batchListMu.Unlock()
	err := qb.exportFunc(flushedBatch.ctx, flushedBatch.req)
	for _, idx := range flushedBatch.idxList {
		qb.queue.OnProcessingFinished(idx, err)
	}
}

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

		lastItem := len(qb.batchList) - 1
		reqs, err := qb.mergeSplitFunc(context.Background(),
			qb.cfg.MaxSizeConfig,
			qb.batchList[lastItem].req, req)

		if err != nil || len(reqs) == 0 {
			return 0, err
		}

		for offset, newReq := range reqs {
			if offset != 0 {
				qb.batchList = append(qb.batchList, batch{})
			}
			qb.batchList[lastItem+offset].req = newReq
			qb.batchList[lastItem+offset].ctx = context.Background()
			qb.batchList[lastItem+offset].idxList = []uint64{idx}
		}

		if reqs[len(reqs)-1].ItemsCount() > qb.cfg.MinSizeItems {
			return len(reqs), nil
		}
		return len(reqs) - 1, nil
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
		mergedReq, err := qb.mergeFunc(context.Background(), qb.batchList[lastItem].req, req)
		if err != nil {
			return 0, err
		}
		qb.batchList[lastItem].req = mergedReq
		qb.batchList[lastItem].idxList = append(qb.batchList[lastItem].idxList, idx)
	}
	if qb.batchList[len(qb.batchList)-1].req.ItemsCount() > qb.cfg.MinSizeItems {
		return 1, nil
	}
	return 0, nil

}

// allocateWorker() allocates a worker to run the incoming function. It blocks until a worker is available.
func (qb *Batcher) allocateWorker(workerFunc func()) {
	if qb.maxWorkers == 0 {
		qb.stopWG.Add(1)
		go func() {
			workerFunc()
			qb.stopWG.Done()
		}()
		return
	}

	if qb.numWorkers.Load() >= int64(qb.maxWorkers) {
		<-qb.allocateWorkerCh
	}
	qb.numWorkers.Add(1)
	qb.stopWG.Add(1)
	go func() {
		workerFunc()
		qb.stopWG.Done()

		if qb.numWorkers.Load() >= int64(qb.maxWorkers) {
			qb.allocateWorkerCh <- true
		}
		qb.numWorkers.Add(-1)
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

	readCh := make(chan bool, 1)
	readCh <- true

	// This function reads until flush is triggered because of request size, then send a signal
	// for the next readerFlusherFunc to be scheduled.
	readerFlusherFunc := func() {
		for {
			idx, _, req, ok := qb.queue.Read(context.Background())
			if !ok {
				qb.shutdownCh <- true
				return
			}

			flushCount, err := qb.push(req, idx)
			if err != nil {
				qb.queue.OnProcessingFinished(idx, err)
			}

			if flushCount > 0 {
				readCh <- true // Allocate the next readerFlusherFunc
				for i := 1; i < flushCount; i++ {
					qb.allocateWorker(func() { qb.flushIfNecessary(false) })
				}
				qb.flushIfNecessary(false)
				return
			}
		}
	}

	// Negative max worker is for testing queue metrics
	if qb.maxWorkers == -1 {
		return nil
	}

	qb.stopWG.Add(1)
	go func() {
		for {
			select {
			case <-readCh:
				qb.allocateWorker(readerFlusherFunc)
			case <-qb.timer.C:
				qb.allocateWorker(func() { qb.flushIfNecessary(false) })
			case <-qb.shutdownCh:
				qb.allocateWorker(func() { qb.flushIfNecessary(true) })
				qb.stopWG.Done()
				return
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
