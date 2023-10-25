// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/request"
)

// Request is a wrapper around request.Request which adds context and a callback to be called when the request
// is finished processing.
type Request struct {
	request.Request
	ctx                          context.Context
	onProcessingFinishedCallback func()
}

func New(ctx context.Context, req request.Request) *Request {
	return &Request{
		Request: req,
		ctx:     ctx,
	}
}

// Context returns the context.Context of the requests.
func (req *Request) Context() context.Context {
	return req.ctx
}

// SetContext updates the context.Context of the request.
func (req *Request) SetContext(ctx context.Context) {
	req.ctx = ctx
}

func (req *Request) OnError(err error) *Request {
	if r, ok := req.Request.(request.ErrorHandler); ok {
		return New(req.ctx, r.OnError(err))
	}
	return req
}

// Count returns a number of items in the request. If the request does not implement RequestItemsCounter
// then 0 is returned.
func (req *Request) Count() int {
	if counter, ok := req.Request.(request.ItemsCounter); ok {
		return counter.ItemsCount()
	}
	return 0
}

// OnProcessingFinished calls the optional callback function to handle cleanup after all processing is finished
func (req *Request) OnProcessingFinished() {
	if req.onProcessingFinishedCallback != nil {
		req.onProcessingFinishedCallback()
	}
}

// SetOnProcessingFinished allows to set an optional callback function to do the cleanup (e.g. remove the item from persistent queue)
func (req *Request) SetOnProcessingFinished(callback func()) {
	req.onProcessingFinishedCallback = callback
}

// Unmarshaler defines a function which takes a byte slice and unmarshals it into a relevant request
type Unmarshaler func([]byte) (*Request, error)

// Marshaler defines a function which takes a request and marshals it into a byte slice
type Marshaler func(*Request) ([]byte, error)
