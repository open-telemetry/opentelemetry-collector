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

// ErrQueueIsFull is the error returned when an item is offered to the Queue and the queue is full and setup to
// not block.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
var ErrQueueIsFull = errors.New("sending queue is full")

// DoneCallback represents the callback that will be called when the read request is completely processed by the
// downstream components.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type DoneCallback func(processErr error)

type ConsumeFunc[T any] func(context.Context, T, DoneCallback)

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
	Size() int64
	// Capacity returns the capacity of the queue.
	Capacity() int64
}

// TODO: Investigate why linter "unused" fails if add a private "read" func on the Queue.
type readableQueue[T any] interface {
	Queue[T]
	// Read pulls the next available item from the queue along with its done callback. Once processing is
	// finished, the done callback must be called to clean up the storage.
	// The function blocks until an item is available or if the queue is stopped.
	// If the queue is stopped returns false, otherwise true.
	Read(context.Context) (context.Context, T, DoneCallback, bool)
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
type Factory[T any] func(context.Context, Settings, Config, ConsumeFunc[T]) Queue[T]

// NewMemoryQueueFactory returns a factory to create a new memory queue.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMemoryQueueFactory[T any]() Factory[T] {
	return func(_ context.Context, _ Settings, cfg Config, consume ConsumeFunc[T]) Queue[T] {
		q := newBoundedMemoryQueue[T](memoryQueueSettings[T]{
			sizer:    &requestSizer[T]{},
			capacity: int64(cfg.QueueSize),
			blocking: cfg.Blocking,
		})
		return newConsumerQueue(q, cfg.NumConsumers, consume)
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
	return func(_ context.Context, set Settings, cfg Config, consume ConsumeFunc[T]) Queue[T] {
		q := newPersistentQueue[T](persistentQueueSettings[T]{
			sizer:       &requestSizer[T]{},
			capacity:    int64(cfg.QueueSize),
			blocking:    cfg.Blocking,
			signal:      set.Signal,
			storageID:   *storageID,
			marshaler:   factorySettings.Marshaler,
			unmarshaler: factorySettings.Unmarshaler,
			set:         set.ExporterSettings,
		})
		return newConsumerQueue(q, cfg.NumConsumers, consume)
	}
}
