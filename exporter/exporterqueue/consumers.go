// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

type consumerQueue[T any] struct {
	readableQueue[T]
	numConsumers int
	consumeFunc  ConsumeFunc[T]
	stopWG       sync.WaitGroup
}

func newConsumerQueue[T any](q readableQueue[T], numConsumers int, consumeFunc ConsumeFunc[T]) *consumerQueue[T] {
	return &consumerQueue[T]{
		readableQueue: q,
		numConsumers:  numConsumers,
		consumeFunc:   consumeFunc,
	}
}

// Start ensures that queue and all consumers are started.
func (qc *consumerQueue[T]) Start(ctx context.Context, host component.Host) error {
	if err := qc.readableQueue.Start(ctx, host); err != nil {
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
				ctx, req, done, ok := qc.readableQueue.Read(context.Background())
				if !ok {
					return
				}
				qc.consumeFunc(ctx, req, done)
			}
		}()
	}
	startWG.Wait()

	return nil
}

// Shutdown ensures that queue and all consumers are stopped.
func (qc *consumerQueue[T]) Shutdown(ctx context.Context) error {
	err := qc.readableQueue.Shutdown(ctx)
	qc.stopWG.Wait()
	return err
}
