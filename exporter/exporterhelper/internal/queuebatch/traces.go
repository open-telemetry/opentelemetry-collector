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
	td         ptrace.Traces
	cachedSize int
}

func newTracesRequest(td ptrace.Traces) request.Request {
	return &tracesRequest{
		td:         td,
		cachedSize: -1,
	}
}

type tracesEncoding struct{}

var _ encoding[request.Request] = tracesEncoding{}

func (tracesEncoding) Unmarshal(bytes []byte) (context.Context, request.Request, error) {
	if queue.PersistRequestContextOnRead() {
		ctx, traces, err := pdatareq.UnmarshalTraces(bytes)
		if errors.Is(err, pdatareq.ErrInvalidFormat) {
			// fall back to unmarshaling without context
			traces, err = tracesUnmarshaler.UnmarshalTraces(bytes)
		}
		return ctx, newTracesRequest(traces), err
	}
	traces, err := tracesUnmarshaler.UnmarshalTraces(bytes)
	if err != nil {
		var req request.Request
		return context.Background(), req, err
	}
	return context.Background(), newTracesRequest(traces), nil
}

func (tracesEncoding) Marshal(ctx context.Context, req request.Request) ([]byte, error) {
	traces := req.(*tracesRequest).td
	if queue.PersistRequestContextOnWrite() {
		return pdatareq.MarshalTraces(ctx, traces)
	}
	return tracesMarshaler.MarshalTraces(traces)
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
	return req.td.SpanCount()
}

func (req *tracesRequest) size(sizer sizer.TracesSizer) int {
	if req.cachedSize == -1 {
		req.cachedSize = sizer.TracesSize(req.td)
	}
	return req.cachedSize
}

func (req *tracesRequest) setCachedSize(size int) {
	req.cachedSize = size
}

func (req *tracesRequest) BytesSize() int {
	return tracesMarshaler.TracesSize(req.td)
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
