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
	workerPool chan bool
	exportFunc func(ctx context.Context, req internal.Request) error
	stopWG     sync.WaitGroup
}

func NewBatcher(batchCfg exporterbatcher.Config,
	queue Queue[internal.Request],
	exportFunc func(ctx context.Context, req internal.Request) error,
	maxWorkers int,
) (Batcher, error) {
	if !batchCfg.Enabled {
		return &DisabledBatcher{BaseBatcher: newBaseBatcher(batchCfg, queue, exportFunc, maxWorkers)}, nil
	}

	return &DefaultBatcher{BaseBatcher: newBaseBatcher(batchCfg, queue, exportFunc, maxWorkers)}, nil
}

func newBaseBatcher(batchCfg exporterbatcher.Config,
	queue Queue[internal.Request],
	exportFunc func(ctx context.Context, req internal.Request) error,
	maxWorkers int,
) BaseBatcher {
	workerPool := make(chan bool, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workerPool <- true
	}
	return BaseBatcher{
		batchCfg:   batchCfg,
		queue:      queue,
		workerPool: workerPool,
		exportFunc: exportFunc,
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
	<-qb.workerPool
	go func() {
		defer func() {
			qb.workerPool <- true
			qb.stopWG.Done()
		}()
		qb.flush(batchToFlush)
	}()
}
