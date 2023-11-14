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
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

// persistentQueue holds the queue backed by file storage
type persistentQueue struct {
	*persistentContiguousStorage
	stopWG       sync.WaitGroup
	set          exporter.CreateSettings
	storageID    component.ID
	numConsumers int
}

// NewPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func NewPersistentQueue(capacity int, numConsumers int, storageID component.ID, marshaler QueueRequestMarshaler,
	unmarshaler QueueRequestUnmarshaler, set exporter.CreateSettings) Queue {
	return &persistentQueue{
		persistentContiguousStorage: newPersistentContiguousStorage(set.Logger, uint64(capacity), marshaler, unmarshaler),
		numConsumers:                numConsumers,
		set:                         set,
		storageID:                   storageID,
	}
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue) Start(ctx context.Context, host component.Host, set QueueSettings) error {
	storageClient, err := toStorageClient(ctx, pq.storageID, host, pq.set.ID, set.DataType)
	if err != nil {
		return err
	}
	pq.persistentContiguousStorage.start(ctx, storageClient)
	for i := 0; i < pq.numConsumers; i++ {
		pq.stopWG.Add(1)
		go func() {
			defer pq.stopWG.Done()
			for {
				req, found := pq.persistentContiguousStorage.get()
				if !found {
					return
				}
				set.Callback(req)
			}
		}()
	}
	return nil
}

// Shutdown stops accepting items, shuts down the queue and closes the persistent queue
func (pq *persistentQueue) Shutdown(ctx context.Context) error {
	err := pq.persistentContiguousStorage.Shutdown(ctx)
	pq.stopWG.Wait()
	return err
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
