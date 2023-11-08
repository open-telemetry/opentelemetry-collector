// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/collector/exporter"
	"go.uber.org/zap"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
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
	logger      *zap.Logger
	stopped     *atomic.Bool
	compID      component.ID
	storageID   component.ID
	storage     *persistentContiguousStorage
	capacity    uint64
	marshaler   RequestMarshaler
	unmarshaler RequestUnmarshaler
	dataType    component.DataType
}

// buildPersistentStorageName returns a name that is constructed out of queue name and signal type. This is done
// to avoid conflicts between different signals, which require unique persistent storage name
func buildPersistentStorageName(name string, signal component.DataType) string {
	return fmt.Sprintf("%s-%s", name, signal)
}

// NewPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func NewPersistentQueue(set exporter.CreateSettings, capacity int, storageID component.ID, marshaler RequestMarshaler,
	unmarshaler RequestUnmarshaler, dataType component.DataType) Queue {
	return &persistentQueue{
		logger:      set.Logger,
		capacity:    uint64(capacity),
		stopped:     &atomic.Bool{},
		compID:      set.ID,
		storageID:   storageID,
		marshaler:   marshaler,
		unmarshaler: unmarshaler,
		dataType:    dataType,
	}
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pq.storageID, host, pq.compID, pq.dataType)
	if err != nil {
		return err
	}
	storageName := buildPersistentStorageName(pq.compID.Name(), pq.dataType)
	pq.storage = newPersistentContiguousStorage(ctx, storageName, storageClient, pq.logger, pq.capacity, pq.marshaler, pq.unmarshaler)
	return nil
}

// Offer inserts the specified element into this queue if it is possible to do so immediately without violating
// capacity restrictions, returning true upon success and false if no space is currently available.
func (pq *persistentQueue) Offer(item Request) bool {
	if pq.stopped.Load() {
		return false
	}

	return pq.storage.put(item) == nil
}

// Poll retrieves and removes the head of this queue.
func (pq *persistentQueue) Poll() <-chan Request {
	return pq.storage.get()
}

// Stop stops accepting items, shuts down the queue and closes the persistent queue
func (pq *persistentQueue) Shutdown(context.Context) error {
	// stop accepting requests before the storage or the successful processing result will fail to write to persistent storage
	pq.stopped.Store(true)
	stopStorage(pq)
	return nil
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
