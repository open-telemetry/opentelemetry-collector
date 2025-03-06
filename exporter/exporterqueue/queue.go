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

// Done represents the callback that will be called when the read request is completely processed by the
// downstream components.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Done interface {
	// OnDone needs to be called when processing of the queue item is done.
	OnDone(error)
}

type ConsumeFunc[T any] func(context.Context, T, Done)

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
	Read(context.Context) (context.Context, T, Done, bool)
}

// Settings defines settings for creating a queue.
type Settings[T any] struct {
	Signal           pipeline.Signal
	ExporterSettings exporter.Settings
	Encoding         Encoding[T]
}

type Encoding[T any] interface {
	// Marshal is a function that can marshal a request into bytes.
	Marshal(T) ([]byte, error)

	// Unmarshal is a function that can unmarshal bytes into a request.
	Unmarshal([]byte) (T, error)
}

// NewQueue returns a queue
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewQueue[T any](_ context.Context, set Settings[T], cfg Config, consume ConsumeFunc[T]) Queue[T] {
	if !cfg.Enabled {
		return newDisabledQueue(consume)
	}
	if cfg.StorageID != nil {
		q := newPersistentQueue[T](persistentQueueSettings[T]{
			sizer:     &requestSizer[T]{},
			capacity:  int64(cfg.QueueSize),
			blocking:  cfg.Blocking,
			signal:    set.Signal,
			storageID: *cfg.StorageID,
			encoding:  set.Encoding,
			set:       set.ExporterSettings,
		})
		return newAsyncQueue(q, cfg.NumConsumers, consume)
	}

	q := newMemoryQueue[T](memoryQueueSettings[T]{
		sizer:    &requestSizer[T]{},
		capacity: int64(cfg.QueueSize),
		blocking: cfg.Blocking,
	})

	return newAsyncQueue(q, cfg.NumConsumers, consume)
}
