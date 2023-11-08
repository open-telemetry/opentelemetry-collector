// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"go.opentelemetry.io/collector/component"
)

// Queue defines a queue which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
type Queue interface {
	component.Component
	// Offer inserts the specified element into this queue if it is possible to do so immediately without violating
	// capacity restrictions, returning true upon success and false if no space is currently available.
	Offer(item Request) bool
	// Poll retrieves and removes the head of this queue.
	Poll() <-chan Request
	// Size returns the current Size of the queue
	Size() int
	// Capacity returns the capacity of the queue.
	Capacity() int
	// IsPersistent returns true if the queue is persistent.
	// TODO: Do not expose this method if the interface moves to a public package.
	IsPersistent() bool
}
