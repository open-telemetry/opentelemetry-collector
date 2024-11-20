// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
)

type batch struct {
	ctx     context.Context
	req     internal.Request
	idxList []uint64
}

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher interface {
	component.Component
}

type BaseBatcher struct {
	batchCfg   exporterbatcher.Config
	queue      Queue[internal.Request]
	maxWorkers int
	workerPool chan bool
	exportFunc func(ctx context.Context, req internal.Request) error
	stopWG     sync.WaitGroup
}

func NewBatcher(batchCfg exporterbatcher.Config,
	queue Queue[internal.Request],
	exportFunc func(ctx context.Context, req internal.Request) error,
	maxWorkers int) (Batcher, error) {
	if !batchCfg.Enabled {
		return &DisabledBatcher{
			BaseBatcher{
				batchCfg:   batchCfg,
				queue:      queue,
				maxWorkers: maxWorkers,
				exportFunc: exportFunc,
				stopWG:     sync.WaitGroup{},
			},
		}, nil
	}

	return &DefaultBatcher{
		BaseBatcher: BaseBatcher{
			batchCfg:   batchCfg,
			queue:      queue,
			maxWorkers: maxWorkers,
			exportFunc: exportFunc,
			stopWG:     sync.WaitGroup{},
		},
	}, nil
}

func (qb *BaseBatcher) startWorkerPool() {
	if qb.maxWorkers == 0 {
		return
	}
	qb.workerPool = make(chan bool, qb.maxWorkers)
	for i := 0; i < qb.maxWorkers; i++ {
		qb.workerPool <- true
	}
}

// flush exports the incoming batch synchronously.
func (qb *BaseBatcher) flush(batchToFlush batch) {
	err := qb.exportFunc(batchToFlush.ctx, batchToFlush.req)
	for _, idx := range batchToFlush.idxList {
		qb.queue.OnProcessingFinished(idx, err)
	}
}

// flushAsync starts a goroutine that calls flushIfNecessary. It blocks until a worker is available.
func (qb *BaseBatcher) flushAsync(batchToFlush batch) {
	qb.stopWG.Add(1)
	if qb.maxWorkers == 0 {
		go func() {
			defer qb.stopWG.Done()
			qb.flush(batchToFlush)
		}()
		return
	}
	<-qb.workerPool
	go func() {
		defer qb.stopWG.Done()
		qb.flush(batchToFlush)
		qb.workerPool <- true
	}()
}
