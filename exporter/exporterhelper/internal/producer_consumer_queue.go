// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

type QueueSettings struct {
	exporter.CreateSettings
	DataType component.DataType
	Callback func(item Request)
}

// ProducerConsumerQueue defines a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
type ProducerConsumerQueue interface {
	// Start starts the queue with a given number of goroutines consuming items from the queue
	// and passing them into the consumer callback.
	Start(ctx context.Context, host component.Host, set QueueSettings) error
	// Produce is used by the producer to submit new item to the queue. Returns false if the item wasn't added
	// to the queue due to queue overflow.
	Produce(item Request) bool
	// Size returns the current Size of the queue
	Size() int
	// Stop stops all consumers, as well as the length reporter if started,
	// and releases the items channel. It blocks until all consumers have stopped.
	Stop()
	// Capacity returns the capacity of the queue.
	Capacity() int
	// IsPersistent returns true if the queue is persistent.
	// TODO: Do not expose this method if the interface moves to a public package.
	IsPersistent() bool
}
