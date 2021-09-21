// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build enable_unstable
// +build enable_unstable

package exporterhelper

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/extension/storage"
)

// persistentQueue holds the queue backed by file storage
type persistentQueue struct {
	logger     *zap.Logger
	stopWG     sync.WaitGroup
	stopOnce   sync.Once
	stopChan   chan struct{}
	numWorkers int
	storage    persistentStorage
}

// newPersistentQueue creates a new queue backed by file storage; name parameter must be a unique value that identifies the queue
func newPersistentQueue(ctx context.Context, name string, capacity int, logger *zap.Logger, client storage.Client, unmarshaler requestUnmarshaler) *persistentQueue {
	return &persistentQueue{
		logger:   logger,
		stopChan: make(chan struct{}),
		storage:  newPersistentContiguousStorage(ctx, name, uint64(capacity), logger, client, unmarshaler),
	}
}

// StartConsumers starts the given number of consumers which will be consuming items
func (pq *persistentQueue) StartConsumers(num int, callback func(item interface{})) {
	pq.numWorkers = num

	factory := func() internal.Consumer {
		return internal.ConsumerFunc(callback)
	}

	for i := 0; i < pq.numWorkers; i++ {
		pq.stopWG.Add(1)
		go func() {
			defer pq.stopWG.Done()
			consumer := factory()

			for {
				select {
				case req := <-pq.storage.get():
					consumer.Consume(req)
				case <-pq.stopChan:
					return
				}
			}
		}()
	}
}

// Produce adds an item to the queue and returns true if it was accepted
func (pq *persistentQueue) Produce(item interface{}) bool {
	err := pq.storage.put(item.(request))
	return err == nil
}

// Stop stops accepting items, shuts down the queue and closes the persistent queue
func (pq *persistentQueue) Stop() {
	pq.stopOnce.Do(func() {
		pq.storage.stop()
		close(pq.stopChan)
		pq.stopWG.Wait()
	})
}

// Size returns the current depth of the queue, excluding the item already in the storage channel (if any)
func (pq *persistentQueue) Size() int {
	return int(pq.storage.size())
}
