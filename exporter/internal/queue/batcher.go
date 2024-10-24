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

	currentBatch *batch // the batch that is being built
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

// flushAsync() starts a goroutine that calls flushIfNecessary(). It blocks until a worker is available.
func (qb *Batcher) flushAsync(batchToFlush batch) {
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

	// This goroutine reads and then flushes if request reaches size limit. There are two operations in this
	// goroutine that could be blocking:
	// 1. Reading from the queue is blocked until the queue is non-empty or until the queue is stopped.
	// 2. flushAsync() blocks until there are idle workers in the worker pool.
	qb.stopWG.Add(1)
	go func() {
		defer qb.stopWG.Done()
		for {
			idx, _, req, ok := qb.queue.Read(context.Background())

			if !ok {
				qb.shutdownCh <- true
				if qb.currentBatch != nil {
					panic("batching is not supported yet so reading batch should always be nil")
				}
				return
			}
			if !qb.batchCfg.Enabled {
				qb.currentBatch = &batch{
					req:     req,
					ctx:     context.Background(),
					idxList: []uint64{idx}}
				qb.flushAsync(*qb.currentBatch)
				qb.currentBatch = nil
			} else {
				panic("not implemented")
			}
		}
	}()

	// The following goroutine is in charge of listening to timer and shutdown signal. This is a seperate goroutine
	// from the reading-flushing, because we want to keep timer seperate blocking operations.
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
