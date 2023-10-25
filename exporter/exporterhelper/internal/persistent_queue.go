// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	intrequest "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/request"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

var (
	// Monkey patching for unit test
	stopStorage = func(queue *persistentQueue) {
		if queue.storage != nil {
			queue.storage.stop()
		}
	}
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

// persistentQueue holds the queue backed by file storage
type persistentQueue struct {
	stopWG       sync.WaitGroup
	stopOnce     sync.Once
	stopChan     chan struct{}
	storageID    component.ID
	storage      *persistentContiguousStorage
	capacity     uint64
	numConsumers int
	marshaler    request.Marshaler
	unmarshaler  request.Unmarshaler
	set          exporter.CreateSettings
	dataType     component.DataType
	callback     func(*intrequest.Request)
}

// buildPersistentStorageName returns a name that is constructed out of queue name and signal type. This is done
// to avoid conflicts between different signals, which require unique persistent storage name
func buildPersistentStorageName(name string, signal component.DataType) string {
	return fmt.Sprintf("%s-%s", name, signal)
}

// NewPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func NewPersistentQueue(capacity int, numConsumers int, storageID component.ID, set queue.Settings, marshaler request.Marshaler,
	unmarshaler request.Unmarshaler) queue.Queue {
	return &persistentQueue{
		capacity:     uint64(capacity),
		numConsumers: numConsumers,
		storageID:    storageID,
		marshaler:    marshaler,
		unmarshaler:  unmarshaler,
		stopChan:     make(chan struct{}),
		set:          set.CreateSettings,
		dataType:     set.DataType,
		callback:     set.Callback,
	}
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pq.storageID, host, pq.set.ID, pq.dataType)
	if err != nil {
		return err
	}
	storageName := buildPersistentStorageName(pq.set.ID.Name(), pq.dataType)
	pq.storage = newPersistentContiguousStorage(ctx, storageName, storageClient, pq.set.Logger, pq.capacity, pq.marshaler, pq.unmarshaler)
	for i := 0; i < pq.numConsumers; i++ {
		pq.stopWG.Add(1)
		go func() {
			defer pq.stopWG.Done()
			for {
				select {
				case req := <-pq.storage.get():
					pq.callback(req)
				case <-pq.stopChan:
					return
				}
			}
		}()
	}
	return nil
}

// Produce adds an item to the queue and returns true if it was accepted
func (pq *persistentQueue) Produce(req *intrequest.Request) bool {
	err := pq.storage.put(req)
	return err == nil
}

// Stop stops accepting items, shuts down the queue and closes the persistent queue
func (pq *persistentQueue) Stop() {
	pq.stopOnce.Do(func() {
		// stop the consumers before the storage or the successful processing result will fail to write to persistent storage
		close(pq.stopChan)
		pq.stopWG.Wait()
		stopStorage(pq)
	})
}

// Size returns the current depth of the queue, excluding the item already in the storage channel (if any)
func (pq *persistentQueue) Size() int {
	return int(pq.storage.size())
}

func (pq *persistentQueue) Capacity() int {
	return int(pq.capacity)
}

func (pq *persistentQueue) IsPersistent() bool {
	return true
}

func toStorageClient(ctx context.Context, storageID component.ID, host component.Host, ownerID component.ID, signal component.DataType) (storage.Client, error) {
	extension, err := getStorageExtension(host.GetExtensions(), storageID)
	if err != nil {
		return nil, err
	}

	client, err := extension.GetClient(ctx, component.KindExporter, ownerID, string(signal))
	if err != nil {
		return nil, err
	}

	return client, err
}

func getStorageExtension(extensions map[component.ID]component.Component, storageID component.ID) (storage.Extension, error) {
	if ext, found := extensions[storageID]; found {
		if storageExt, ok := ext.(storage.Extension); ok {
			return storageExt, nil
		}
		return nil, errWrongExtensionType
	}
	return nil, errNoStorageClient
}
