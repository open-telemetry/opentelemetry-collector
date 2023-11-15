// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension/experimental/storage"
)

var (
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

// persistentQueue holds the queue backed by file storage
type persistentQueue[T any] struct {
	*persistentContiguousStorage[T]
	set       exporter.CreateSettings
	storageID component.ID
	dataType  component.DataType
}

// NewPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func NewPersistentQueue[T any](capacity int, dataType component.DataType, storageID component.ID, marshaler func(req T) ([]byte, error), unmarshaler func([]byte) (T, error), set exporter.CreateSettings) Queue[T] {
	return &persistentQueue[T]{
		persistentContiguousStorage: newPersistentContiguousStorage(set.Logger, uint64(capacity), marshaler, unmarshaler),
		set:                         set,
		storageID:                   storageID,
		dataType:                    dataType,
	}
}

// Start starts the persistentQueue with the given number of consumers.
func (pq *persistentQueue[T]) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pq.storageID, host, pq.set.ID, pq.dataType)
	if err != nil {
		return err
	}
	pq.persistentContiguousStorage.start(ctx, storageClient)
	return nil
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
