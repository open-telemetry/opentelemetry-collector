// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"errors"
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
	stopWG     sync.WaitGroup
}

func NewBatcher(batchCfg exporterbatcher.Config, queue Queue[internal.Request], maxWorkers int) (Batcher, error) {
	if !batchCfg.Enabled {
		return &DisabledBatcher{
			BaseBatcher{
				batchCfg:   batchCfg,
				queue:      queue,
				maxWorkers: maxWorkers,
				stopWG:     sync.WaitGroup{},
			},
		}, nil
	}

	if batchCfg.MaxSizeConfig.MaxSizeItems != 0 {
		return nil, errors.ErrUnsupported
	}

	return &DefaultBatcher{
		BaseBatcher: BaseBatcher{
			batchCfg:   batchCfg,
			queue:      queue,
			maxWorkers: maxWorkers,
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
	err := batchToFlush.req.Export(batchToFlush.ctx)
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

// Shutdown ensures that queue and all Batcher are stopped.
func (qb *BaseBatcher) Shutdown(_ context.Context) error {
	qb.stopWG.Wait()
	return nil
}
