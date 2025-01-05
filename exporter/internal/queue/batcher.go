// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal"
)

type batch[K any] struct {
	ctx     context.Context
	req     K
	idxList []uint64
}

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher interface {
	component.Component
}

type BaseBatcher[K any] struct {
	batchCfg exporterbatcher.Config
	queue    exporterqueue.Queue[K]
	// TODO: Remove when the -1 hack for testing is removed.
	merger     Merger[K]
	maxWorkers int
	workerPool chan bool
	exportFunc func(ctx context.Context, req K) error
	stopWG     sync.WaitGroup
}

func NewBatcher(batchCfg exporterbatcher.Config,
	queue exporterqueue.Queue[internal.Request],
	exportFunc func(ctx context.Context, req internal.Request) error,
	maxWorkers int,
) (Batcher, error) {
	if !batchCfg.Enabled {
		return &DisabledBatcher[internal.Request]{BaseBatcher: newBaseBatcher(batchCfg, queue, exportFunc, maxWorkers)}, nil
	}
	return &DefaultBatcher[internal.Request]{BaseBatcher: newBaseBatcher(batchCfg, queue, exportFunc, maxWorkers)}, nil
}

func newBaseBatcher(batchCfg exporterbatcher.Config,
	queue exporterqueue.Queue[internal.Request],
	exportFunc func(ctx context.Context, req internal.Request) error,
	maxWorkers int,
) BaseBatcher[internal.Request] {
	var workerPool chan bool
	if maxWorkers > 0 {
		workerPool = make(chan bool, maxWorkers)
		for i := 0; i < maxWorkers; i++ {
			workerPool <- true
		}
	}
	return BaseBatcher[internal.Request]{
		batchCfg:   batchCfg,
		queue:      queue,
		merger:     requestMerger{},
		maxWorkers: maxWorkers,
		workerPool: workerPool,
		exportFunc: exportFunc,
		stopWG:     sync.WaitGroup{},
	}
}

// flush starts a goroutine that calls exportFunc. It blocks until a worker is available if necessary.
func (qb *BaseBatcher[K]) flush(batchToFlush batch[K]) {
	err := qb.exportFunc(batchToFlush.ctx, batchToFlush.req)
	for _, idx := range batchToFlush.idxList {
		qb.queue.OnProcessingFinished(idx, err)
	}
}

type Merger[K any] interface {
	MergeSplit(src, dst K, cfg exporterbatcher.MaxSizeConfig) ([]K, error)
}

type requestMerger struct{}

func (requestMerger) MergeSplit(src, dst internal.Request, cfg exporterbatcher.MaxSizeConfig) ([]internal.Request, error) {
	return src.MergeSplit(context.Background(), cfg, dst)
}
