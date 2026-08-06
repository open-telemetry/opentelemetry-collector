// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
	pdatareq "go.opentelemetry.io/collector/pdata/xpdata/request"
)

var (
	tracesMarshaler   = &ptrace.ProtoMarshaler{}
	tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}
)

// NewTracesQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using ptrace.Traces.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesQueueBatchSettings() Settings[request.Request] {
	return Settings[request.Request]{
		ReferenceCounter: tracesReferenceCounter{},
		Encoding:         tracesEncoding{},
	}
}

var (
	_ request.Request      = (*tracesRequest)(nil)
	_ request.ErrorHandler = (*tracesRequest)(nil)
)

type tracesRequest struct {
	td ptrace.Traces
	// Sizes are cached per sizer type: the batcher and the queue may ask for
	// bytes and items on the same request, and a single cache would return a
	// value computed for the wrong sizer. -1 means "not yet computed".
	cachedItemsSize int
	cachedBytesSize int
}

func newTracesRequest(td ptrace.Traces) request.Request {
	return &tracesRequest{
		td:              td,
		cachedItemsSize: -1,
		cachedBytesSize: -1,
	}
}

type tracesEncoding struct{}

var _ encoding[request.Request] = tracesEncoding{}

func (tracesEncoding) Unmarshal(bytes []byte) (context.Context, request.Request, error) {
	ctx, traces, err := pdatareq.UnmarshalTraces(bytes)
	if errors.Is(err, pdatareq.ErrInvalidFormat) {
		// fall back to unmarshaling without context
		traces, err = tracesUnmarshaler.UnmarshalTraces(bytes)
	}
	return ctx, newTracesRequest(traces), err
}

func (tracesEncoding) Marshal(ctx context.Context, req request.Request) ([]byte, error) {
	return pdatareq.MarshalTraces(ctx, req.(*tracesRequest).td)
}

var _ queue.ReferenceCounter[request.Request] = tracesReferenceCounter{}

type tracesReferenceCounter struct{}

func (tracesReferenceCounter) Ref(req request.Request) {
	pref.RefTraces(req.(*tracesRequest).td)
}

func (tracesReferenceCounter) Unref(req request.Request) {
	pref.UnrefTraces(req.(*tracesRequest).td)
}

func (req *tracesRequest) OnError(err error) request.Request {
	var traceError consumererror.Traces
	if errors.As(err, &traceError) {
		// TODO: Add logic to unref the new request created here.
		return newTracesRequest(traceError.Data())
	}
	return req
}

func (req *tracesRequest) ItemsCount() int {
	if req.cachedItemsSize < 0 {
		req.cachedItemsSize = req.td.SpanCount()
	}
	return req.cachedItemsSize
}

func (req *tracesRequest) size(sz sizer.TracesSizer) int {
	switch sz.(type) {
	case *sizer.TracesCountSizer:
		if req.cachedItemsSize < 0 {
			req.cachedItemsSize = sz.TracesSize(req.td)
		}
		return req.cachedItemsSize
	case *sizer.TracesBytesSizer:
		if req.cachedBytesSize < 0 {
			req.cachedBytesSize = sz.TracesSize(req.td)
		}
		return req.cachedBytesSize
	default:
		return sz.TracesSize(req.td)
	}
}

// setCachedSize records the size for sz's dimension and invalidates the other,
// which the caller (mergeTo/split) did not maintain. This keeps the dimension
// used by the batcher (a single configured sizer) O(1) across merges while
// never returning a stale value for the other dimension.
func (req *tracesRequest) setCachedSize(sz sizer.TracesSizer, size int) {
	switch sz.(type) {
	case *sizer.TracesCountSizer:
		req.cachedItemsSize = size
		req.cachedBytesSize = -1
	case *sizer.TracesBytesSizer:
		req.cachedBytesSize = size
		req.cachedItemsSize = -1
	}
}

func (req *tracesRequest) BytesSize() int {
	if req.cachedBytesSize < 0 {
		req.cachedBytesSize = tracesMarshaler.TracesSize(req.td)
	}
	return req.cachedBytesSize
}

// RequestConsumeFromTraces returns a RequestConsumeFunc that consumes ptrace.Traces.
func RequestConsumeFromTraces(pusher consumer.ConsumeTracesFunc) request.RequestConsumeFunc {
	return func(ctx context.Context, request request.Request) error {
		return pusher.ConsumeTraces(ctx, request.(*tracesRequest).td)
	}
}

// RequestFromTraces returns a RequestConverterFunc that converts ptrace.Traces into a Request.
func RequestFromTraces() request.RequestConverterFunc[ptrace.Traces] {
	return func(_ context.Context, traces ptrace.Traces) (request.Request, error) {
		return newTracesRequest(traces), nil
	}
}
