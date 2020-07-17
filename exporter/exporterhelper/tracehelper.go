// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// traceDataPusherOld is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusherOld func(ctx context.Context, td consumerdata.TraceData) (droppedSpans int, err error)

// traceDataPusher is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusher func(ctx context.Context, td pdata.Traces) (droppedSpans int, err error)

// traceExporterOld implements the exporter with additional helper internalOptions.
type traceExporterOld struct {
	baseExporter
	dataPusher traceDataPusherOld
}

func (te *traceExporterOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	exporterCtx := obsreport.ExporterContext(ctx, te.exporterFullName)
	_, err := te.dataPusher(exporterCtx, td)
	return err
}

// NewTraceExporterOld creates an TraceExporterOld that can record metrics and can wrap every
// request with a Span. If no internalOptions are passed it just adds the exporter format as a
// tag in the Context.
func NewTraceExporterOld(
	config configmodels.Exporter,
	dataPusher traceDataPusherOld,
	options ...ExporterOption,
) (component.TraceExporterOld, error) {

	if config == nil {
		return nil, errNilConfig
	}

	if dataPusher == nil {
		return nil, errNilPushTraceData
	}

	dataPusher = dataPusher.withObservability(config.Name())

	return &traceExporterOld{
		baseExporter: newBaseExporter(config.Name(), options...),
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

type traceExporter struct {
	baseExporter
	dataPusher traceDataPusher
}

func (te *traceExporter) ConsumeTraces(
	ctx context.Context,
	td pdata.Traces,
) error {
	exporterCtx := obsreport.ExporterContext(ctx, te.exporterFullName)
	_, err := te.dataPusher(exporterCtx, td)
	return err
}

// NewTraceExporter creates a TraceExporter that can record metrics and can wrap
// every request with a Span.
func NewTraceExporter(
	config configmodels.Exporter,
	dataPusher traceDataPusher,
	options ...ExporterOption,
) (component.TraceExporter, error) {

	if config == nil {
		return nil, errNilConfig
	}

	if dataPusher == nil {
		return nil, errNilPushTraceData
	}

	dataPusher = dataPusher.withObservability(config.Name())

	return &traceExporter{
		baseExporter: newBaseExporter(config.Name(), options...),
		dataPusher:   dataPusher,
	}, nil
}

// withObservability wraps the current pusher into a function that records
// the observability signals during the pusher execution.
func (p traceDataPusher) withObservability(exporterName string) traceDataPusher {
	return func(ctx context.Context, td pdata.Traces) (int, error) {
		ctx = obsreport.StartTraceDataExportOp(ctx, exporterName)
		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(ctx, td)

		// TODO: this is not ideal: it should come from the next function itself.
		// 	temporarily loading it from internal format. Once full switch is done
		// 	to new metrics will remove this.
		numSpans := td.SpanCount()
		obsreport.EndTraceDataExportOp(ctx, numSpans, droppedSpans, err)
		return droppedSpans, err
	}
}
