// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/queue"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	intrequest "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// Queue defines a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Queue interface {
	// Start starts the queue with a given number of goroutines consuming items from the queue
	// and passing them into the consumer callback.
	Start(ctx context.Context, host component.Host) error
	// Produce is used by the producer to submit new item to the queue. Returns false if the item wasn't added
	// to the queue due to queue overflow.
	Produce(item *intrequest.Request) bool
	// Size returns the current Size of the queue
	Size() int
	// Stop stops all consumers, as well as the length reporter if started,
	// and releases the items channel. It blocks until all consumers have stopped.
	Stop()
	// Capacity returns the capacity of the queue.
	Capacity() int
	// IsPersistent returns true if the queue is persistent.
	// TODO: Remove this method once we move it to the config.
	IsPersistent() bool
}
