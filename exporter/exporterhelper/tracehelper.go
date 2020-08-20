// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporterhelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// traceDataPusherOld is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusherOld func(ctx context.Context, td consumerdata.TraceData) (droppedSpans int, err error)

type traceExporterOld struct {
	*baseExporter
	dataPusher traceDataPusherOld
}

func (texp *traceExporterOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	exporterCtx := obsreport.ExporterContext(ctx, texp.cfg.Name())
	_, err := texp.dataPusher(exporterCtx, td)
	return err
}

// NewTraceExporterOld creates an TraceExporterOld that records observability metrics and wraps every request with a Span.
func NewTraceExporterOld(
	cfg configmodels.Exporter,
	dataPusher traceDataPusherOld,
	options ...ExporterOption,
) (component.TraceExporterOld, error) {

	if cfg == nil {
		return nil, errNilConfig
	}

	if dataPusher == nil {
		return nil, errNilPushTraceData
	}

	dataPusher = dataPusher.withObservability(cfg.Name())

	return &traceExporterOld{
		baseExporter: newBaseExporter(cfg, options...),
		dataPusher:   dataPusher,
	}, nil
}

// withObservability wraps the current pusher into a function that records
// the observability signals during the pusher execution.
func (p traceDataPusherOld) withObservability(exporterName string) traceDataPusherOld {
	return func(ctx context.Context, td consumerdata.TraceData) (int, error) {
		ctx = obsreport.StartTraceDataExportOp(ctx, exporterName)
		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(ctx, td)

		// TODO: this is not ideal: it should come from the next function itself.
		// 	temporarily loading it from internal format. Once full switch is done
		// 	to new metrics will remove this.
		numSpans := len(td.Spans)
		obsreport.EndTraceDataExportOp(ctx, numSpans, droppedSpans, err)
		return droppedSpans, err
	}
}

// traceDataPusher is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusher func(ctx context.Context, td pdata.Traces) (droppedSpans int, err error)

type tracesRequest struct {
	baseRequest
	td     pdata.Traces
	pusher traceDataPusher
}

func newTracesRequest(ctx context.Context, td pdata.Traces, pusher traceDataPusher) request {
	return &tracesRequest{
		baseRequest: baseRequest{ctx: ctx},
		td:          td,
		pusher:      pusher,
	}
}

func (req *tracesRequest) onPartialError(partialErr consumererror.PartialError) request {
	return newTracesRequest(req.ctx, partialErr.GetTraces(), req.pusher)
}

func (req *tracesRequest) export(ctx context.Context) (int, error) {
	return req.pusher(ctx, req.td)
}

func (req *tracesRequest) count() int {
	return req.td.SpanCount()
}

type traceExporter struct {
	*baseExporter
	pusher traceDataPusher
}

func (texp *traceExporter) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	exporterCtx := obsreport.ExporterContext(ctx, texp.cfg.Name())
	req := newTracesRequest(exporterCtx, td, texp.pusher)
	_, err := texp.sender.send(req)
	return err
}

// NewTraceExporter creates a TraceExporter that records observability metrics and wraps every request with a Span.
func NewTraceExporter(
	cfg configmodels.Exporter,
	dataPusher traceDataPusher,
	options ...ExporterOption,
) (component.TraceExporter, error) {

	if cfg == nil {
		return nil, errNilConfig
	}

	if dataPusher == nil {
		return nil, errNilPushTraceData
	}

	be := newBaseExporter(cfg, options...)
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &tracesExporterWithObservability{
			exporterName: cfg.Name(),
			nextSender:   nextSender,
		}
	})

	return &traceExporter{
		baseExporter: be,
		pusher:       dataPusher,
	}, nil
}

type tracesExporterWithObservability struct {
	exporterName string
	nextSender   requestSender
}

func (tewo *tracesExporterWithObservability) send(req request) (int, error) {
	req.setContext(obsreport.StartTraceDataExportOp(req.context(), tewo.exporterName))
	// Forward the data to the next consumer (this pusher is the next).
	droppedSpans, err := tewo.nextSender.send(req)

	// TODO: this is not ideal: it should come from the next function itself.
	// 	temporarily loading it from internal format. Once full switch is done
	// 	to new metrics will remove this.
	obsreport.EndTraceDataExportOp(req.context(), req.count(), droppedSpans, err)
	return droppedSpans, err
}
