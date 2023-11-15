// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

type QueueConsumers[T any] struct {
	queue        Queue[T]
	numConsumers int
	callback     func(context.Context, T)
	stopWG       sync.WaitGroup
}

func NewQueueConsumers[T any](q Queue[T], numConsumers int, callback func(context.Context, T)) *QueueConsumers[T] {
	return &QueueConsumers[T]{
		queue:        q,
		numConsumers: numConsumers,
		callback:     callback,
		stopWG:       sync.WaitGroup{},
	}
}

// Start ensures that queue and all consumers are started.
func (qc *QueueConsumers[T]) Start(ctx context.Context, host component.Host) error {
	if err := qc.queue.Start(ctx, host); err != nil {
		return err
	}

	var startWG sync.WaitGroup
	for i := 0; i < qc.numConsumers; i++ {
		qc.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer qc.stopWG.Done()
			for {
				item, success := qc.queue.Poll()
				if !success {
					return
				}
				qc.callback(item.Context, item.Request)
				item.OnProcessingFinished()
			}
		}()
	}
	startWG.Wait()

	return nil
}

// Shutdown ensures that queue and all consumers are stopped.
func (qc *QueueConsumers[T]) Shutdown(ctx context.Context) error {
	if err := qc.queue.Shutdown(ctx); err != nil {
		return err
	}
	qc.stopWG.Wait()
	return nil
}
