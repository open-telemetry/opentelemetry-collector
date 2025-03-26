// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
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
		Encoding: tracesEncoding{},
		Sizers: map[RequestSizerType]request.Sizer[Request]{
			RequestSizerTypeRequests: NewRequestsSizer(),
			RequestSizerTypeItems:    request.NewItemsSizer(),
			RequestSizerTypeBytes: request.BaseSizer{
				SizeofFunc: func(req request.Request) int64 {
					return int64(tracesMarshaler.TracesSize(req.(*tracesRequest).td))
				},
			},
		},
	}
}

type tracesRequest struct {
	td          ptrace.Traces
	cachedItems int
	cachedBytes int
}

func newTracesRequest(td ptrace.Traces) Request {
	return &tracesRequest{
		td:          td,
		cachedItems: td.SpanCount(),
		cachedBytes: -1,
	}
}

type tracesEncoding struct{}

func (tracesEncoding) Unmarshal(bytes []byte) (Request, error) {
	traces, err := tracesUnmarshaler.UnmarshalTraces(bytes)
	if err != nil {
		return nil, err
	}
	return newTracesRequest(traces), nil
}

func (tracesEncoding) Marshal(req Request) ([]byte, error) {
	return tracesMarshaler.MarshalTraces(req.(*tracesRequest).td)
}

func (req *tracesRequest) OnError(err error) Request {
	var traceError consumererror.Traces
	if errors.As(err, &traceError) {
		return newTracesRequest(traceError.Data())
	}
	return req
}

func (req *tracesRequest) ItemsCount() int {
	return req.cachedItems
}

func (req *tracesRequest) ByteSize() int {
	if req.cachedBytes == -1 {
		sz := sizer.TracesBytesSizer{}
		req.cachedBytes = sz.TracesSize(req.td)
	}
	return req.cachedBytes
}

// sizeFromSizer returns the size of the request based on the sizer. If the size is unknown, it returns 0.
func (req *tracesRequest) sizeFromSizer(sz sizer.TracesSizer) int {
	switch sz.(type) {
	case *sizer.TracesCountSizer:
		return req.ItemsCount()
	case *sizer.TracesBytesSizer:
		return req.ByteSize()
	default:
		return 0
	}
}

func (req *tracesRequest) setCachedSize(sz sizer.TracesSizer, size int) {
	switch sz.(type) {
	case *sizer.TracesCountSizer:
		req.cachedItems = size
	case *sizer.TracesBytesSizer:
		req.cachedBytes = size
	default:
	}
}

type tracesExporter struct {
	*internal.BaseExporter
	consumer.Traces
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
	return NewTracesRequest(ctx, set, requestFromTraces(), requestConsumeFromTraces(pusher),
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

// NewTracesRequest creates a new traces exporter based on a custom TracesConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesRequest(
	_ context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[ptrace.Traces],
	pusher RequestConsumeFunc,
	options ...Option,
) (exporter.Traces, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilTracesConverter
	}

	if pusher == nil {
		return nil, errNilConsumeRequest
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalTraces, pusher, options...)
	if err != nil {
		return nil, err
	}

	tc, err := consumer.NewTraces(newConsumeTraces(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &tracesExporter{BaseExporter: be, Traces: tc}, nil
}

func newConsumeTraces(converter RequestConverterFunc[ptrace.Traces], be *internal.BaseExporter, logger *zap.Logger) consumer.ConsumeTracesFunc {
	return func(ctx context.Context, td ptrace.Traces) error {
		req, err := converter(ctx, td)
		if err != nil {
			logger.Error("Failed to convert traces. Dropping data.",
				zap.Int("dropped_spans", td.SpanCount()),
				zap.Error(err))
			return consumererror.NewPermanent(err)
		}
		return be.Send(ctx, req)
	}
}
