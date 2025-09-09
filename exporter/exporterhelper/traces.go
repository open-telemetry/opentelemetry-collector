// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
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
func NewTracesQueueBatchSettings() QueueBatchSettings {
	return QueueBatchSettings{
		ReferenceCounter: tracesReferenceCounter{},
		Encoding:         tracesEncoding{},
		ItemsSizer:       request.NewItemsSizer(),
		BytesSizer: request.BaseSizer{
			SizeofFunc: func(req request.Request) int64 {
				return int64(tracesMarshaler.TracesSize(req.(*tracesRequest).td))
			},
		},
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

func newTracesRequest(td ptrace.Traces) Request {
	return &tracesRequest{
		td:         td,
		cachedSize: -1,
	}
}

type tracesEncoding struct{}

var _ QueueBatchEncoding[Request] = tracesEncoding{}

func (tracesEncoding) Unmarshal(bytes []byte) (context.Context, Request, error) {
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
		var req Request
		return context.Background(), req, err
	}
	return context.Background(), newTracesRequest(traces), nil
}

func (tracesEncoding) Marshal(ctx context.Context, req Request) ([]byte, error) {
	traces := req.(*tracesRequest).td
	if queue.PersistRequestContextOnWrite() {
		return pdatareq.MarshalTraces(ctx, traces)
	}
	return tracesMarshaler.MarshalTraces(traces)
}

var _ queue.ReferenceCounter[Request] = tracesReferenceCounter{}

type tracesReferenceCounter struct{}

func (tracesReferenceCounter) Ref(req Request) {
	pref.RefTraces(req.(*tracesRequest).td)
}

func (tracesReferenceCounter) Unref(req Request) {
	pref.UnrefTraces(req.(*tracesRequest).td)
}

func (req *tracesRequest) OnError(err error) Request {
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


// NewTraces creates an exporter.Traces that records observability metrics and wraps every request with a Span.
func NewTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeTracesFunc,
	options ...Option,
) (exporter.Traces, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushTraces
	}
	return internal.NewTracesRequest(ctx, set, requestFromTraces(), requestConsumeFromTraces(pusher),
		append([]Option{internal.WithQueueBatchSettings(NewTracesQueueBatchSettings())}, options...)...)
}

// requestConsumeFromTraces returns a RequestConsumeFunc that consumes ptrace.Traces.
func requestConsumeFromTraces(pusher consumer.ConsumeTracesFunc) RequestConsumeFunc {
	return func(ctx context.Context, request Request) error {
		return pusher.ConsumeTraces(ctx, request.(*tracesRequest).td)
	}
}

// requestFromTraces returns a RequestConverterFunc that converts ptrace.Traces into a Request.
func requestFromTraces() RequestConverterFunc[ptrace.Traces] {
	return func(_ context.Context, traces ptrace.Traces) (Request, error) {
		return newTracesRequest(traces), nil
	}
}

