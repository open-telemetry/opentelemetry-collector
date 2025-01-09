// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pipeline"
)

// ErrQueueIsFull is the error returned when an item is offered to the Queue and the queue is full.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
var ErrQueueIsFull = errors.New("sending queue is full")

// Queue defines a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Queue[T any] interface {
	component.Component
	// Offer inserts the specified element into this queue if it is possible to do so immediately
	// without violating capacity restrictions. If success returns no error.
	// It returns ErrQueueIsFull if no space is currently available.
	Offer(ctx context.Context, item T) error
	// Size returns the current Size of the queue
	Size() int
	// Capacity returns the capacity of the queue.
	Capacity() int
	// Read pulls the next available item from the queue along with its index. Once processing is
	// finished, the index should be called with OnProcessingFinished to clean up the storage.
	// The function blocks until an item is available or if the queue is stopped.
	// Returns false if reading failed or if the queue is stopped.
	Read(context.Context) (uint64, context.Context, T, bool)
	// OnProcessingFinished should be called to remove the item of the given index from the queue once processing is finished.
	OnProcessingFinished(index uint64, consumeErr error)
}

// Settings defines settings for creating a queue.
type Settings struct {
	Signal           pipeline.Signal
	ExporterSettings exporter.Settings
}

// Marshaler is a function that can marshal a request into bytes.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Marshaler[T any] func(T) ([]byte, error)

// Unmarshaler is a function that can unmarshal bytes into a request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Unmarshaler[T any] func([]byte) (T, error)

// Factory is a function that creates a new queue.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Factory[T any] func(context.Context, Settings, Config) Queue[T]

// NewMemoryQueueFactory returns a factory to create a new memory queue.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMemoryQueueFactory[T any]() Factory[T] {
	return func(_ context.Context, _ Settings, cfg Config) Queue[T] {
		return newBoundedMemoryQueue[T](memoryQueueSettings[T]{
			sizer:    &requestSizer[T]{},
			capacity: int64(cfg.QueueSize),
		})
	}
}

// PersistentQueueSettings defines developer settings for the persistent queue factory.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type PersistentQueueSettings[T any] struct {
	// Marshaler is used to serialize queue elements before storing them in the persistent storage.
	Marshaler Marshaler[T]
	// Unmarshaler is used to deserialize requests after reading them from the persistent storage.
	Unmarshaler Unmarshaler[T]
}

// NewPersistentQueueFactory returns a factory to create a new persistent queue.
// If cfg.storageID is nil then it falls back to memory queue.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewPersistentQueueFactory[T any](storageID *component.ID, factorySettings PersistentQueueSettings[T]) Factory[T] {
	if storageID == nil {
		return NewMemoryQueueFactory[T]()
	}
	return func(_ context.Context, set Settings, cfg Config) Queue[T] {
		return newPersistentQueue[T](persistentQueueSettings[T]{
			sizer:       &requestSizer[T]{},
			capacity:    int64(cfg.QueueSize),
			signal:      set.Signal,
			storageID:   *storageID,
			marshaler:   factorySettings.Marshaler,
			unmarshaler: factorySettings.Unmarshaler,
			set:         set.ExporterSettings,
		})
	}
}
