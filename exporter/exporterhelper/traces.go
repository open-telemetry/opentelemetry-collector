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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	tracesMarshaler   = &ptrace.ProtoMarshaler{}
	tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}
)

type tracesRequest struct {
	td         ptrace.Traces
	pusher     consumer.ConsumeTracesFunc
	cachedSize int
}

func newTracesRequest(td ptrace.Traces, pusher consumer.ConsumeTracesFunc) Request {
	return &tracesRequest{
		td:         td,
		pusher:     pusher,
		cachedSize: -1,
	}
}

type tracesEncoding struct {
	pusher consumer.ConsumeTracesFunc
}

func (te *tracesEncoding) Unmarshal(bytes []byte) (Request, error) {
	traces, err := tracesUnmarshaler.UnmarshalTraces(bytes)
	if err != nil {
		return nil, err
	}
	return newTracesRequest(traces, te.pusher), nil
}

func (te *tracesEncoding) Marshal(req Request) ([]byte, error) {
	return tracesMarshaler.MarshalTraces(req.(*tracesRequest).td)
}

func (req *tracesRequest) OnError(err error) Request {
	var traceError consumererror.Traces
	if errors.As(err, &traceError) {
		return newTracesRequest(traceError.Data(), req.pusher)
	}
	return req
}

func (req *tracesRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.td)
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
		return nil, errNilPushTraceData
	}
	return NewTracesRequest(ctx, set, requestFromTraces(pusher), append([]Option{internal.WithEncoding(&tracesEncoding{pusher: pusher})}, options...)...)
}

// Deprecated: [v0.122.0] use RequestConverterFunc[ptrace.Traces].
type RequestFromTracesFunc = RequestConverterFunc[ptrace.Traces]

// requestFromTraces returns a RequestConverterFunc that converts ptrace.Traces into a Request.
func requestFromTraces(pusher consumer.ConsumeTracesFunc) RequestConverterFunc[ptrace.Traces] {
	return func(_ context.Context, traces ptrace.Traces) (Request, error) {
		return newTracesRequest(traces, pusher), nil
	}
}

// NewTracesRequest creates a new traces exporter based on a custom TracesConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesRequest(
	_ context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[ptrace.Traces],
	options ...Option,
) (exporter.Traces, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilTracesConverter
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalTraces, options...)
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
