// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/internal/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/queue"
)

// ErrQueueIsFull is the error that Queue returns when full.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
var ErrQueueIsFull = exporterqueue.ErrQueueIsFull

// Queue defines a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type Queue[T any] queue.Queue[T]

// Settings defines settings for creating a queue.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type Settings = exporterqueue.Settings

// Marshaler is a function that can marshal a request into bytes.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type Marshaler[T any] func(T) ([]byte, error)

// Unmarshaler is a function that can unmarshal bytes into a request.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type Unmarshaler[T any] func([]byte) (T, error)

// Factory is a function that creates a new queue.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type Factory[T any] func(context.Context, Settings, Config) Queue[T]

// NewMemoryQueueFactory returns a factory to create a new memory queue.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
func NewMemoryQueueFactory[T exporterqueue.ItemsCounter]() Factory[T] {
	// Ugly code to convert between external and internal defined types
	internalFactory := exporterqueue.NewMemoryQueueFactory[T]()
	return func(ctx context.Context, settings Settings, cfg Config) Queue[T] {
		return internalFactory(ctx, settings, cfg)
	}
}

// PersistentQueueSettings defines developer settings for the persistent queue factory.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type PersistentQueueSettings[T any] struct {
	// Marshaler is used to serialize queue elements before storing them in the persistent storage.
	Marshaler Marshaler[T]
	// Unmarshaler is used to deserialize requests after reading them from the persistent storage.
	Unmarshaler Unmarshaler[T]
}

// NewPersistentQueueFactory returns a factory to create a new persistent queue.
// If cfg.StorageID is nil then it falls back to memory queue.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
func NewPersistentQueueFactory[T exporterqueue.ItemsCounter](storageID *component.ID, factorySettings PersistentQueueSettings[T]) Factory[T] {
	// Ugly code to convert between external and internal defined types
	internalFactory := exporterqueue.NewPersistentQueueFactory(storageID, exporterqueue.PersistentQueueSettings[T]{
		Marshaler:   exporterqueue.Marshaler[T](factorySettings.Marshaler),
		Unmarshaler: exporterqueue.Unmarshaler[T](factorySettings.Unmarshaler),
	})
	return func(ctx context.Context, s exporterqueue.Settings, c exporterqueue.Config) Queue[T] {
		return internalFactory(ctx, s, c)
	}
}
