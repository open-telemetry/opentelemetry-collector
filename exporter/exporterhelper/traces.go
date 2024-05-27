// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/ctrace"
	"go.opentelemetry.io/collector/consumer/ctrace/ctraceerror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var tracesMarshaler = &ptrace.ProtoMarshaler{}
var tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}

type tracesRequest struct {
	td     ptrace.Traces
	pusher ctrace.ConsumeTracesFunc
}

func newTracesRequest(td ptrace.Traces, pusher ctrace.ConsumeTracesFunc) Request {
	return &tracesRequest{
		td:     td,
		pusher: pusher,
	}
}

func newTraceRequestUnmarshalerFunc(pusher ctrace.ConsumeTracesFunc) exporterqueue.Unmarshaler[Request] {
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
	var traceError ctraceerror.Traces
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
	*baseExporter
	ctrace.Traces
}

// NewTracesExporter creates an exporter.Traces that records observability metrics and wraps every request with a Span.
func NewTracesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
	pusher ctrace.ConsumeTracesFunc,
	options ...Option,
) (exporter.Traces, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushTraceData
	}
	tracesOpts := []Option{
		withMarshaler(tracesRequestMarshaler), withUnmarshaler(newTraceRequestUnmarshalerFunc(pusher)),
		withBatchFuncs(mergeTraces, mergeSplitTraces),
	}
	return NewTracesRequestExporter(ctx, set, requestFromTraces(pusher), append(tracesOpts, options...)...)
}

// RequestFromTracesFunc converts ptrace.Traces into a user-defined Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromTracesFunc func(context.Context, ptrace.Traces) (Request, error)

// requestFromTraces returns a RequestFromTracesFunc that converts ptrace.Traces into a Request.
func requestFromTraces(pusher ctrace.ConsumeTracesFunc) RequestFromTracesFunc {
	return func(_ context.Context, traces ptrace.Traces) (Request, error) {
		return newTracesRequest(traces, pusher), nil
	}
}

// NewTracesRequestExporter creates a new traces exporter based on a custom TracesConverter and RequestSender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesRequestExporter(
	_ context.Context,
	set exporter.CreateSettings,
	converter RequestFromTracesFunc,
	options ...Option,
) (exporter.Traces, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilTracesConverter
	}

	be, err := newBaseExporter(set, component.DataTypeTraces, newTracesExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	tc, err := ctrace.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		req, cErr := converter(ctx, td)
		if cErr != nil {
			set.Logger.Error("Failed to convert traces. Dropping data.",
				zap.Int("dropped_spans", td.SpanCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		sErr := be.send(ctx, req)
		if errors.Is(sErr, queue.ErrQueueIsFull) {
			be.obsrep.recordEnqueueFailure(ctx, component.DataTypeTraces, int64(req.ItemsCount()))
		}
		return sErr
	}, ctrace.WithCapabilities(be.capabilities))

	return &traceExporter{
		baseExporter: be,
		Traces:       tc,
	}, err
}

type tracesExporterWithObservability struct {
	baseRequestSender
	obsrep *ObsReport
}

func newTracesExporterWithObservability(obsrep *ObsReport) requestSender {
	return &tracesExporterWithObservability{obsrep: obsrep}
}

func (tewo *tracesExporterWithObservability) send(ctx context.Context, req Request) error {
	c := tewo.obsrep.StartTracesOp(ctx)
	// Forward the data to the next consumer (this pusher is the next).
	err := tewo.nextSender.send(c, req)
	tewo.obsrep.EndTracesOp(c, req.ItemsCount(), err)
	return err
}
