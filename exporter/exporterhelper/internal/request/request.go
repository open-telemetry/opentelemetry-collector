// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// Request represents a single request that can be sent to an external endpoint.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Request interface {
	// ItemsCount returns a number of basic items in the request where item is the smallest piece of data that can be
	// sent. For example, for OTLP exporter, this value represents the number of spans,
	// metric data points or log records.
	ItemsCount() int
	// MergeSplit is a function that merge and/or splits this request with another one into multiple requests based on the
	// configured limit provided in maxSize.
	// MergeSplit does not split if maxSize is zero.
	// All the returned requests MUST have a number of items that does not exceed the maximum number of items.
	// Size of the last returned request MUST be less or equal than the size of any other returned request.
	// The original request MUST not be mutated if error is returned after mutation or if the exporter is
	// marked as not mutable. The length of the returned slice MUST not be 0.
	// Experimental: This API is at the early stage of development and may change without backward compatibility
	// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
	MergeSplit(ctx context.Context, maxSize int, sizerType SizerType, req Request) ([]Request, error)
	// BytesSize returns the size of the request in bytes.
	BytesSize() int
}

// ErrorHandler is an optional interface that can be implemented by Request to provide a way handle partial
// temporary failures. For example, if some items failed to process and can be retried, this interface allows to
// return a new Request that contains the items left to be sent. Otherwise, the original Request should be returned.
// If not implemented, the original Request will be returned assuming the error is applied to the whole Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type ErrorHandler interface {
	Request
	// OnError returns a new Request may contain the items left to be sent if some items failed to process and can be retried.
	// Otherwise, it should return the original Request.
	OnError(error) Request
}

// DeferredQueueCompletion is implemented by requests whose downstream sender
// has a completion event that occurs after Send returns. The queue keeps the
// request owned until the supplied callback is invoked.
type DeferredQueueCompletion interface {
	SetQueueCompletion(func(error)) bool
}

// QueueCheckpointStore exposes small exporter-owned checkpoints through the
// configured persistent queue's storage namespace. Memory queues do not
// provide this interface.
type QueueCheckpointStore interface {
	LoadCheckpoint(context.Context, string) ([]byte, bool, error)
	SaveCheckpoint(context.Context, string, []byte) error
}

// QueueCheckpointStoreSetter is implemented by requests that need a durable
// checkpoint alongside their persistent queue entries.
type QueueCheckpointStoreSetter interface {
	SetQueueCheckpointStore(QueueCheckpointStore)
}

type RequestConverterFunc[T any] func(context.Context, T) (Request, error)

// RequestConsumeFunc processes the request. After the function returns, the request is no longer accessible,
// and accessing it is considered undefined behavior.
type RequestConsumeFunc = sender.SendFunc[Request]

// QueueRequestCount lets an atomic envelope charge admission for every child.
type QueueRequestCount interface{ QueueRequestsCount() int64 }

// QueueItemTokenSetter associates a request with its durable queue entry.
type QueueItemTokenSetter interface{ SetQueueItemToken(uint64) }

// QueueItemUpdate replaces a queue envelope without changing its admission charge.
type QueueItemUpdate struct {
	Token uint64
	Value []byte
}

// QueueCheckpointTransaction persists child retirement and recovery state together.
// The queue must atomically publish all updates or leave all previous values intact.
type QueueCheckpointTransaction interface {
	SaveCheckpointAndItems(context.Context, string, []byte, []QueueItemUpdate) error
}

// QueueItemReader refreshes a request decoded before checkpoint recovery ran.
type QueueItemReader interface {
	LoadQueueItem(context.Context, uint64) ([]byte, error)
}
