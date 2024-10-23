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
	batchCfg exporterbatcher.Config

	queue      Queue[internal.Request]
	maxWorkers int

	exportFunc func(context.Context, internal.Request) error

	readingBatch *batch
	timer        *time.Timer
	shutdownCh   chan bool

	stopWG sync.WaitGroup
}

func NewBatcher(batchCfg exporterbatcher.Config, queue Queue[internal.Request],
	maxWorkers int, exportFunc func(context.Context, internal.Request) error) *Batcher {
	return &Batcher{
		batchCfg:   batchCfg,
		queue:      queue,
		maxWorkers: maxWorkers,
		exportFunc: exportFunc,
		stopWG:     sync.WaitGroup{},
		shutdownCh: make(chan bool, 1),
	}
}

// If preconditions pass, flush() take an item from the head of batch list and exports it.
func (qb *Batcher) flush(batchToFlush batch) {
	err := qb.exportFunc(batchToFlush.ctx, batchToFlush.req)
	for _, idx := range batchToFlush.idxList {
		qb.queue.OnProcessingFinished(idx, err)
	}
}

// allocateFlusher() starts a goroutine that calls flushIfNecessary(). It blocks until a worker is available.
func (qb *Batcher) allocateFlusher(batchToFlush batch) {
	// maxWorker = 0 means we don't limit the number of flushers.
	if qb.maxWorkers == 0 {
		qb.stopWG.Add(1)
		go func() {
			qb.flush(batchToFlush)
			qb.stopWG.Done()
		}()
		return
	}
	panic("not implemented")
}

// Start ensures that queue and all consumers are started.
func (qb *Batcher) Start(ctx context.Context, host component.Host) error {
	if err := qb.queue.Start(ctx, host); err != nil {
		return err
	}

	if qb.batchCfg.Enabled {
		panic("not implemented")
	}

	// Timer doesn't do anything yet, but adding it so compiler won't complain about qb.timer.C
	qb.timer = time.NewTimer(math.MaxInt)
	qb.timer.Stop()

	if qb.maxWorkers != 0 {
		panic("not implemented")
	}

	// This goroutine keeps reading until flush is triggered because of request size.
	qb.stopWG.Add(1)
	go func() {
		defer qb.stopWG.Done()
		for {
			idx, _, req, ok := qb.queue.Read(context.Background())

			if !ok {
				qb.shutdownCh <- true
				if qb.readingBatch != nil {
					panic("batching is supported yet so reading batch should always be nil")
				}

				return
			}
			if !qb.batchCfg.Enabled {
				qb.readingBatch = &batch{
					req:     req,
					ctx:     context.Background(),
					idxList: []uint64{idx}}
				qb.allocateFlusher(*qb.readingBatch)
				qb.readingBatch = nil
			} else {
				panic("not implemented")
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
				panic("batching is not yet implemented. Having timer here just so compiler won't complain")
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
