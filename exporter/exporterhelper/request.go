// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// Request represents a single request that can be sent to an external endpoint.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Request interface {
	// Export exports the request to an external endpoint.
	Export(ctx context.Context) error
}

// RequestItemsCounter is an optional interface that can be implemented by Request to provide a number of items
// in the request. This is a recommended interface to implement for exporters. It is required for batching and queueing
// based on number of items. Also, it's used for reporting number of items in collector's logs, metrics and traces.
// If not implemented, collector's logs, metrics and traces will report 0 items.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestItemsCounter interface {
	// ItemsCount returns a number of basic items in the request where item is the smallest piece of data that can be
	// sent. For example, for OTLP exporter, this value represents the number of spans,
	// metric data points or log records.
	ItemsCount() int
}

type request struct {
	Request
	baseRequest
}

var _ internal.Request = (*request)(nil)

func newRequest(ctx context.Context, req Request) *request {
	return &request{
		Request:     req,
		baseRequest: baseRequest{ctx: ctx},
	}
}

func (req *request) OnError(_ error) internal.Request {
	// Potentially we could introduce a new RequestError type that would represent partially succeeded request.
	// In that case we should consider returning them back to the pipeline converted back to pdata in case if
	// sending queue is disabled. We leave it as a future improvement if decided that it's needed.
	return req
}

// Count returns a number of items in the request. If the request does not implement RequestItemsCounter
// then 0 is returned.
func (req *request) Count() int {
	if counter, ok := req.Request.(RequestItemsCounter); ok {
		return counter.ItemsCount()
	}
	return 0
}
