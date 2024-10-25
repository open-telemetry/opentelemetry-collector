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
	stopWG     sync.WaitGroup
}

func NewBatcher(batchCfg exporterbatcher.Config, queue Queue[internal.Request], maxWorkers int) Batcher {
	if maxWorkers != 0 {
		panic("not implemented")
	}

	if batchCfg.Enabled {
		panic("not implemented")
	}

	return &DisabledBatcher{
		BaseBatcher{
			batchCfg:   batchCfg,
			queue:      queue,
			maxWorkers: maxWorkers,
			stopWG:     sync.WaitGroup{},
		},
	}
}

// flush exports the incoming batch synchronously.
func (qb *BaseBatcher) flush(batchToFlush batch) {
	err := batchToFlush.req.Export(batchToFlush.ctx)
	for _, idx := range batchToFlush.idxList {
		qb.queue.OnProcessingFinished(idx, err)
	}
}

// flushAsync starts a goroutine that calls flushIfNecessary. It blocks until a worker is available.
func (qb *BaseBatcher) flushAsync(batchToFlush batch) {
	// maxWorker = 0 means we don't limit the number of flushers.
	if qb.maxWorkers == 0 {
		qb.stopWG.Add(1)
		go func() {
			defer qb.stopWG.Done()
			qb.flush(batchToFlush)
		}()
		return
	}
	panic("not implemented")
}

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *BaseBatcher) Shutdown(ctx context.Context) error {
	qb.stopWG.Wait()
	return nil
}
