// Copyright 2019, OpenTelemetry Authors
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

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
)

// traceDataPusherOld is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusherOld func(ctx context.Context, td consumerdata.TraceData) (droppedSpans int, err error)

// traceDataPusher is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusher func(ctx context.Context, td pdata.TraceData) (droppedSpans int, err error)

// traceExporterOld implements the exporter with additional helper options.
type traceExporterOld struct {
	exporterFullName string
	dataPusher       traceDataPusherOld
	shutdown         Shutdown
}

func (te *traceExporterOld) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (te *traceExporterOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	exporterCtx := obsreport.ExporterContext(ctx, te.exporterFullName)
	_, err := te.dataPusher(exporterCtx, td)
	return err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (te *traceExporterOld) Shutdown(ctx context.Context) error {
	return te.shutdown(ctx)
}

// NewTraceExporterOld creates an TraceExporterOld that can record metrics and can wrap every
// request with a Span. If no options are passed it just adds the exporter format as a
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

	opts := newExporterOptions(options...)

	dataPusher = dataPusher.withObservability(config.Name())

	// The default shutdown function does nothing.
	if opts.shutdown == nil {
		opts.shutdown = func(context.Context) error { return nil }
	}

	return &traceExporterOld{
		exporterFullName: config.Name(),
		dataPusher:       dataPusher,
		shutdown:         opts.shutdown,
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
	exporterFullName string
	dataPusher       traceDataPusher
	shutdown         Shutdown
}

func (te *traceExporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (te *traceExporter) ConsumeTrace(
	ctx context.Context,
	td pdata.TraceData,
) error {
	exporterCtx := obsreport.ExporterContext(ctx, te.exporterFullName)
	_, err := te.dataPusher(exporterCtx, td)
	return err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (te *traceExporter) Shutdown(ctx context.Context) error {
	return te.shutdown(ctx)
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

	opts := newExporterOptions(options...)

	dataPusher = dataPusher.withObservability(config.Name())

	// The default shutdown function does nothing.
	if opts.shutdown == nil {
		opts.shutdown = func(context.Context) error { return nil }
	}

	return &traceExporter{
		exporterFullName: config.Name(),
		dataPusher:       dataPusher,
		shutdown:         opts.shutdown,
	}, nil
}

// withObservability wraps the current pusher into a function that records
// the observability signals during the pusher execution.
func (p traceDataPusher) withObservability(exporterName string) traceDataPusher {
	return func(ctx context.Context, td pdata.TraceData) (int, error) {
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
