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

// NewPersistentQueue creates a new queue backed by file storage; name and signal must be a unique combination that identifies the queue storage
func NewPersistentQueue[T any](capacity int, dataType component.DataType, storageID component.ID, marshaler func(req T) ([]byte, error), unmarshaler func([]byte) (T, error), set exporter.CreateSettings) Queue[T] {
	return &persistentContiguousStorage[T]{
		set:         set,
		storageID:   storageID,
		dataType:    dataType,
		unmarshaler: unmarshaler,
		marshaler:   marshaler,
		capacity:    uint64(capacity),
		putChan:     make(chan struct{}, capacity),
		stopChan:    make(chan struct{}),
	}
}

// Start starts the persistentQueue with the given number of consumers.
func (pcs *persistentContiguousStorage[T]) Start(ctx context.Context, host component.Host) error {
	storageClient, err := toStorageClient(ctx, pcs.storageID, host, pcs.set.ID, pcs.dataType)
	if err != nil {
		return err
	}
	pcs.initClient(ctx, storageClient)
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
