// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/request"
)

// Request defines capabilities required for persistent storage of a request
type Request struct {
	request.Request
	ctx                        context.Context
	processingFinishedCallback func()
}

func NewRequest(ctx context.Context, req request.Request) *Request {
	return &Request{Request: req, ctx: ctx}
}

func (req *Request) OnError(err error) *Request {
	if r, ok := req.Request.(request.ErrorHandler); ok {
		return &Request{
			Request: r.OnError(err),
			ctx:     req.ctx,
		}
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

func (req *Request) Context() context.Context {
	return req.ctx
}

func (req *Request) SetContext(ctx context.Context) {
	req.ctx = ctx
}

func (req *Request) OnProcessingFinished() {
	if req.processingFinishedCallback != nil {
		req.processingFinishedCallback()
	}
}

// RequestUnmarshaler defines a function which takes a byte slice and unmarshals it into a relevant request
type RequestUnmarshaler func([]byte) (*Request, error)

// RequestMarshaler defines a function which takes a request and marshals it into a byte slice
type RequestMarshaler func(*Request) ([]byte, error)
