// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

var (
	// Monkey patching for unit test
	stopStorage = func(storage *persistentContiguousStorage, ctx context.Context) error {
		if storage == nil {
			return nil
		}
		return storage.stop(ctx)
	}
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

// persistentQueue holds the queue backed by file storage
type persistentQueue struct {
	stopWG       sync.WaitGroup
	set          exporter.CreateSettings
	storageID    component.ID
	storage      *persistentContiguousStorage
	capacity     uint64
	numConsumers int
	marshaler    QueueRequestMarshaler
	unmarshaler  QueueRequestUnmarshaler
}

// NewPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func NewPersistentQueue(capacity int, numConsumers int, storageID component.ID, marshaler QueueRequestMarshaler,
	unmarshaler QueueRequestUnmarshaler, set exporter.CreateSettings) Queue {
	return &persistentQueue{
		capacity:     uint64(capacity),
		numConsumers: numConsumers,
		set:          set,
		storageID:    storageID,
		marshaler:    marshaler,
		unmarshaler:  unmarshaler,
	}
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue) Start(ctx context.Context, host component.Host, set QueueSettings) error {
	storageClient, err := toStorageClient(ctx, pq.storageID, host, pq.set.ID, set.DataType)
	if err != nil {
		return err
	}
	pq.storage = newPersistentContiguousStorage(ctx, storageClient, pq.set.Logger, pq.capacity, pq.marshaler, pq.unmarshaler)
	for i := 0; i < pq.numConsumers; i++ {
		pq.stopWG.Add(1)
		go func() {
			defer pq.stopWG.Done()
			for {
				req, found := pq.storage.get()
				if !found {
					return
				}
				set.Callback(req)
			}
		}()
	}
	return nil
}

// Offer inserts the specified element into this queue if it is possible to do so immediately
// without violating capacity restrictions. If success returns no error.
// It returns ErrQueueIsFull if no space is currently available.
func (pq *persistentQueue) Offer(_ context.Context, item any) error {
	return pq.storage.put(item)
}

// Shutdown stops accepting items, shuts down the queue and closes the persistent queue
func (pq *persistentQueue) Shutdown(ctx context.Context) error {
	if pq.storage != nil {
		close(pq.storage.stopChan)
	}
	pq.stopWG.Wait()
	return stopStorage(pq.storage, ctx)
}

// Size returns the current depth of the queue, excluding the item already in the storage channel (if any)
func (pq *persistentQueue) Size() int {
	return int(pq.storage.size())
}

func (pq *persistentQueue) Capacity() int {
	return int(pq.capacity)
}

func toStorageClient(ctx context.Context, storageID component.ID, host component.Host, ownerID component.ID, signal component.DataType) (storage.Client, error) {
	ext, found := host.GetExtensions()[storageID]
	if !found {
		return nil, errNoStorageClient
	}

	storageExt, ok := ext.(storage.Extension)
	if !ok {
		return nil, errWrongExtensionType
	}

	return storageExt.GetClient(ctx, component.KindExporter, ownerID, string(signal))
}
