// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

type Consumers[T any] struct {
	queue        Queue[T]
	numConsumers int
	consumeFunc  func(context.Context, T) error
	stopWG       sync.WaitGroup
}

func NewQueueConsumers[T any](q Queue[T], numConsumers int, consumeFunc func(context.Context, T) error) *Consumers[T] {
	return &Consumers[T]{
		queue:        q,
		numConsumers: numConsumers,
		consumeFunc:  consumeFunc,
		stopWG:       sync.WaitGroup{},
	}
}

// Start ensures that queue and all consumers are started.
func (qc *Consumers[T]) Start(_ context.Context, _ component.Host) error {
	var startWG sync.WaitGroup
	for i := 0; i < qc.numConsumers; i++ {
		qc.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer qc.stopWG.Done()
			for {
				index, ctx, req, ok := qc.queue.Read(context.Background())
				if !ok {
					return
				}
				consumeErr := qc.consumeFunc(ctx, req)
				qc.queue.OnProcessingFinished(index, consumeErr)
			}
		}()
	}
	startWG.Wait()

	return nil
}

// Shutdown ensures that queue and all consumers are stopped.
func (qc *Consumers[T]) Shutdown(_ context.Context) error {
	qc.stopWG.Wait()
	return nil
}
