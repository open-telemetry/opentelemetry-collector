// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// Request represents a single request that can be sent to the endpoint.
type Request interface {
	// ItemsCount returns the count basic item in the request, the smallest pieces of data that can be sent to the endpoint.
	// For example, for OTLP exporter, this value represents the number of spans, metric data points or log records.
	ItemsCount() int
}

// RequestSender is a helper function that sends a request.
type RequestSender func(ctx context.Context, req Request) error

type request struct {
	Request
	baseRequest
	sender RequestSender
}

var _ internal.Request = (*request)(nil)

func (req *request) Export(ctx context.Context) error {
	return req.sender(ctx, req.Request)
}

func (req *request) OnError(_ error) internal.Request {
	// Potentially we could introduce a new RequestError type that would represent partially succeeded request.
	// In that case we should consider returning them back to the pipeline converted back to pdata in case if
	// sending queue is disabled. We leave it as a future improvement if decided that it's needed.
	return req
}
