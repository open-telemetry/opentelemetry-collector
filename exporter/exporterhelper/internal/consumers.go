// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync"
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

// Start ensures that all consumers are started.
func (c *QueueConsumers[T]) Start() {
	var startWG sync.WaitGroup
	for i := 0; i < c.numConsumers; i++ {
		c.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer c.stopWG.Done()
			for {
				item, success := c.queue.Poll()
				if !success {
					return
				}
				c.callback(item.Context, item.Request)
				item.OnProcessingFinished()
			}
		}()
	}
	startWG.Wait()
}

// Shutdown ensures that all consumers are stopped.
func (c *QueueConsumers[T]) Shutdown() {
	c.stopWG.Wait()
}
