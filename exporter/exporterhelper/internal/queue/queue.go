// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pipeline"
)

type Encoding[T any] interface {
	// Marshal is a function that can marshal a request into bytes.
	Marshal(context.Context, T) ([]byte, error)

	// Unmarshal is a function that can unmarshal bytes into a request.
	Unmarshal([]byte) (context.Context, T, error)
}

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

// Settings define internal parameters for a new Queue creation.
type Settings[T any] struct {
	ItemsSizer      request.Sizer[T]
	BytesSizer      request.Sizer[T]
	SizerType       request.SizerType
	Capacity        int64
	NumConsumers    int
	WaitForResult   bool
	BlockOnOverflow bool
	Signal          pipeline.Signal
	StorageID       *component.ID
	Encoding        Encoding[T]
	ID              component.ID
	Telemetry       component.TelemetrySettings
}

func (set *Settings[T]) activeSizer() request.Sizer[T] {
	switch set.SizerType {
	case request.SizerTypeBytes:
		return set.BytesSizer
	case request.SizerTypeItems:
		return set.ItemsSizer
	default:
		return request.RequestsSizer[T]{}
	}
}

func NewQueue[T any](set Settings[T], next ConsumeFunc[T]) (Queue[T], error) {
	// Configure memory queue or persistent based on the config.
	if set.StorageID == nil {
		return newAsyncQueue(newMemoryQueue[T](set), set.NumConsumers, next), nil
	}
	if set.ItemsSizer == nil {
		return nil, errors.New("PersistentQueue requires ItemsSizer to be set")
	}
	if set.BytesSizer == nil {
		return nil, errors.New("PersistentQueue requires BytesSizer to be set")
	}
	return newAsyncQueue(newPersistentQueue[T](set), set.NumConsumers, next), nil
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
