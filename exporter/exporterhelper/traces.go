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
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

var tracesMarshaler = &ptrace.ProtoMarshaler{}
var tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}

type tracesRequest struct {
	td     ptrace.Traces
	pusher consumer.ConsumeTracesFunc
}

func newTracesRequest(td ptrace.Traces, pusher consumer.ConsumeTracesFunc) Request {
	return &tracesRequest{
		td:     td,
		pusher: pusher,
	}
}

func newTraceRequestUnmarshalerFunc(pusher consumer.ConsumeTracesFunc) exporterqueue.Unmarshaler[Request] {
	return func(bytes []byte) (Request, error) {
		traces, err := tracesUnmarshaler.UnmarshalTraces(bytes)
		if err != nil {
			return nil, err
		}
		return newTracesRequest(traces, pusher), nil
	}
}

func tracesRequestMarshaler(req Request) ([]byte, error) {
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

type traceExporter struct {
	*internal.BaseExporter
	consumer.Traces
}

// NewTracesExporter creates an exporter.Traces that records observability metrics and wraps every request with a Span.
func NewTracesExporter(
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
	tracesOpts := []Option{
		internal.WithMarshaler(tracesRequestMarshaler), internal.WithUnmarshaler(newTraceRequestUnmarshalerFunc(pusher)),
		internal.WithBatchFuncs(mergeTraces, mergeSplitTraces),
	}
	return NewTracesRequestExporter(ctx, set, requestFromTraces(pusher), append(tracesOpts, options...)...)
}

// RequestFromTracesFunc converts ptrace.Traces into a user-defined Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromTracesFunc func(context.Context, ptrace.Traces) (Request, error)

// requestFromTraces returns a RequestFromTracesFunc that converts ptrace.Traces into a Request.
func requestFromTraces(pusher consumer.ConsumeTracesFunc) RequestFromTracesFunc {
	return func(_ context.Context, traces ptrace.Traces) (Request, error) {
		return newTracesRequest(traces, pusher), nil
	}
}

// NewTracesRequestExporter creates a new traces exporter based on a custom TracesConverter and RequestSender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesRequestExporter(
	_ context.Context,
	set exporter.Settings,
	converter RequestFromTracesFunc,
	options ...Option,
) (exporter.Traces, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilTracesConverter
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalTraces, newTracesExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	tc, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		req, cErr := converter(ctx, td)
		if cErr != nil {
			set.Logger.Error("Failed to convert traces. Dropping data.",
				zap.Int("dropped_spans", td.SpanCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		sErr := be.Send(ctx, req)
		if errors.Is(sErr, queue.ErrQueueIsFull) {
			be.Obsrep.RecordEnqueueFailure(ctx, pipeline.SignalTraces, int64(req.ItemsCount()))
		}
		return sErr
	}, be.ConsumerOptions...)

	return &traceExporter{
		BaseExporter: be,
		Traces:       tc,
	}, err
}

type tracesExporterWithObservability struct {
	internal.BaseRequestSender
	obsrep *internal.ObsReport
}

func newTracesExporterWithObservability(obsrep *internal.ObsReport) internal.RequestSender {
	return &tracesExporterWithObservability{obsrep: obsrep}
}

func (tewo *tracesExporterWithObservability) Send(ctx context.Context, req Request) error {
	c := tewo.obsrep.StartTracesOp(ctx)
	numTraceSpans := req.ItemsCount()
	// Forward the data to the next consumer (this pusher is the next).
	err := tewo.NextSender.Send(c, req)
	tewo.obsrep.EndTracesOp(c, numTraceSpans, err)
	return err
}
