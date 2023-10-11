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
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var tracesMarshaler = &ptrace.ProtoMarshaler{}
var tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}

type tracesRequest struct {
	baseRequest
	td     ptrace.Traces
	pusher consumer.ConsumeTracesFunc
}

func newTracesRequest(ctx context.Context, td ptrace.Traces, pusher consumer.ConsumeTracesFunc) internal.Request {
	return &tracesRequest{
		baseRequest: baseRequest{ctx: ctx},
		td:          td,
		pusher:      pusher,
	}
}

func newTraceRequestUnmarshalerFunc(pusher consumer.ConsumeTracesFunc) internal.RequestUnmarshaler {
	return func(bytes []byte) (internal.Request, error) {
		traces, err := tracesUnmarshaler.UnmarshalTraces(bytes)
		if err != nil {
			return nil, err
		}
		return newTracesRequest(context.Background(), traces, pusher), nil
	}
}

func tracesRequestMarshaler(req internal.Request) ([]byte, error) {
	return tracesMarshaler.MarshalTraces(req.(*tracesRequest).td)
}

func (req *tracesRequest) OnError(err error) internal.Request {
	var traceError consumererror.Traces
	if errors.As(err, &traceError) {
		return newTracesRequest(req.ctx, traceError.Data(), req.pusher)
	}
	return req
}

func (req *tracesRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.td)
}

func (req *tracesRequest) Count() int {
	return req.td.SpanCount()
}

type traceExporter struct {
	*baseExporter
	consumer.Traces
}

// NewTracesExporter creates an exporter.Traces that records observability metrics and wraps every request with a Span.
func NewTracesExporter(
	_ context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
	pusher consumer.ConsumeTracesFunc,
	options ...Option,
) (exporter.Traces, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if set.Logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushTraceData
	}

	be, err := newBaseExporter(set, component.DataTypeTraces, false, tracesRequestMarshaler,
		newTraceRequestUnmarshalerFunc(pusher), newTracesExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	tc, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		req := newTracesRequest(ctx, td, pusher)
		serr := be.send(req)
		if errors.Is(serr, errSendingQueueIsFull) {
			be.obsrep.recordEnqueueFailure(req.Context(), component.DataTypeTraces, int64(req.Count()))
		}
		return serr
	}, be.consumerOptions...)

	return &traceExporter{
		baseExporter: be,
		Traces:       tc,
	}, err
}

// TracesConverter provides an interface for converting ptrace.Traces into a request.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type TracesConverter interface {
	// RequestFromTraces converts ptrace.Traces into a Request.
	RequestFromTraces(context.Context, ptrace.Traces) (Request, error)
}

// NewTracesRequestExporter creates a new traces exporter based on a custom TracesConverter and RequestSender.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesRequestExporter(
	_ context.Context,
	set exporter.CreateSettings,
	converter TracesConverter,
	options ...Option,
) (exporter.Traces, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilTracesConverter
	}

	be, err := newBaseExporter(set, component.DataTypeTraces, true, nil, nil, newTracesExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	tc, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		req, cErr := converter.RequestFromTraces(ctx, td)
		if cErr != nil {
			set.Logger.Error("Failed to convert traces. Dropping data.",
				zap.Int("dropped_spans", td.SpanCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		r := newRequest(ctx, req)
		sErr := be.send(r)
		if errors.Is(sErr, errSendingQueueIsFull) {
			be.obsrep.recordEnqueueFailure(r.Context(), component.DataTypeTraces, int64(r.Count()))
		}
		return sErr
	}, be.consumerOptions...)

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

func (tewo *tracesExporterWithObservability) send(req internal.Request) error {
	req.SetContext(tewo.obsrep.StartTracesOp(req.Context()))
	// Forward the data to the next consumer (this pusher is the next).
	err := tewo.nextSender.send(req)
	tewo.obsrep.EndTracesOp(req.Context(), req.Count(), err)
	return err
}
