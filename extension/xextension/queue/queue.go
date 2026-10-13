// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/extension/xextension/queue"
import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Extension is the interface that storage extensions must implement
type Extension interface {
	extension.Extension

	// GetClient will create a client for use by the specified component.
	// Each component can have multiple queues (e.g. one for each signal),
	// which can be identified using queueName parameter.
	// The component can use the client to manage state
	GetClient(ctx context.Context, kind component.Kind, id component.ID, queueName string) (Client, error)
}

// Client is the interface that queue clients must implement
type Client interface {
	// Peek returns a channel to watch for the next queue event.
	// If Peek return nil, the queue is closed. If the channel returns nil, the queue closed while blocking on Peek.
	Peek() chan PeekWithCallback
	// Write writes a new element to the queue, alongside its size.
	// The payload of the element is passed as a byte array, and the size is dictated by the sizer used by the queue (items, bytes, requests).
	Write(op WriteOp) error
	// Shutdown stops the client, freeing any resources. The client can no longer be used after this is done.
	Shutdown(ctx context.Context) error
	// Size returns the size of the queue according to the sizer used by the queue (items, bytes, requests).
	Size() int64
}

// WriteOp is a single write operation to the queue
type WriteOp struct {
	// Payload is the byte array
	Payload []byte
	// Size is the size of the item according to the sizer used by the queue (items, bytes, requests)
	Size int64
}

// PeekWithCallback is the response of a peek request, with a payload and a callback to declare the element consumed.
type PeekWithCallback struct {
	// ConsumeCallback is a callback to declare the element as read. It will be disposed and no longer available.
	ConsumeCallback func(err error)
	// Payload is the serialized representation of the entry as a byte array
	Payload []byte
}
