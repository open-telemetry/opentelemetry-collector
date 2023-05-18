// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import "context"

// Request defines capabilities required for persistent storage of a request
type Request interface {
	// Context returns the context.Context of the requests.
	Context() context.Context

	// SetContext updates the context.Context of the requests.
	SetContext(context.Context)

	Export(ctx context.Context) error

	// OnError returns a new Request may contain the items left to be sent if some items failed to process and can be retried.
	// Otherwise, it should return the original Request.
	OnError(error) Request

	// Count returns the count of spans/metric points or log records.
	Count() int

	// Marshal serializes the current request into a byte stream
	Marshal() ([]byte, error)

	// OnProcessingFinished calls the optional callback function to handle cleanup after all processing is finished
	OnProcessingFinished()

	// SetOnProcessingFinished allows to set an optional callback function to do the cleanup (e.g. remove the item from persistent queue)
	SetOnProcessingFinished(callback func())
}

// RequestUnmarshaler defines a function which takes a byte slice and unmarshals it into a relevant request
type RequestUnmarshaler func([]byte) (Request, error)
