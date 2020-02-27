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
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
)

// traceDataPusher is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusher func(ctx context.Context, td consumerdata.TraceData) (droppedSpans int, err error)

// traceV2DataPusher is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceV2DataPusher func(ctx context.Context, td data.TraceData) (droppedSpans int, err error)

// traceExporter implements the exporter with additional helper options.
type traceExporter struct {
	exporterFullName string
	dataPusher       traceDataPusher
	shutdown         Shutdown
}

var _ exporter.TraceExporter = (*traceExporter)(nil)

func (te *traceExporter) Start(host component.Host) error {
	return nil
}

func (te *traceExporter) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	exporterCtx := obsreport.ExporterContext(ctx, te.exporterFullName)
	_, err := te.dataPusher(exporterCtx, td)
	return err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (te *traceExporter) Shutdown() error {
	return te.shutdown()
}

// NewTraceExporter creates an TraceExporter that can record metrics and can wrap every
// request with a Span. If no options are passed it just adds the exporter format as a
// tag in the Context.
func NewTraceExporter(
	config configmodels.Exporter,
	dataPusher traceDataPusher,
	options ...ExporterOption,
) (exporter.TraceExporter, error) {

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
		opts.shutdown = func() error {
			return nil
		}
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
	return func(ctx context.Context, td consumerdata.TraceData) (int, error) {
		exporterCtx, span := obsreport.StartTraceDataExportOp(ctx, exporterName)
		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(exporterCtx, td)

		// TODO: this is not ideal: it should come from the next function itself.
		// 	temporarily loading it from internal format. Once full switch is done
		// 	to new metrics will remove this.
		numSpans := len(td.Spans)
		obsreport.EndTraceDataExportOp(exporterCtx, span, numSpans, droppedSpans, err)
		return droppedSpans, err
	}
}

type traceExporterV2 struct {
	exporterFullName string
	dataPusher       traceV2DataPusher
	shutdown         Shutdown
}

var _ exporter.TraceExporterV2 = (*traceExporterV2)(nil)

func (te *traceExporterV2) Start(host component.Host) error {
	return nil
}

func (te *traceExporterV2) ConsumeTraceV2(
	ctx context.Context,
	td data.TraceData,
) error {
	exporterCtx := obsreport.ExporterContext(ctx, te.exporterFullName)
	_, err := te.dataPusher(exporterCtx, td)
	return err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (te *traceExporterV2) Shutdown() error {
	return te.shutdown()
}

// NewTraceExporterV2 creates a TraceExporterV2 that can record metrics and can wrap
// every request with a Span.
func NewTraceExporterV2(
	config configmodels.Exporter,
	dataPusher traceV2DataPusher,
	options ...ExporterOption,
) (exporter.TraceExporterV2, error) {

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
		opts.shutdown = func() error {
			return nil
		}
	}

	return &traceExporterV2{
		exporterFullName: config.Name(),
		dataPusher:       dataPusher,
		shutdown:         opts.shutdown,
	}, nil
}

// withObservability wraps the current pusher into a function that records
// the observability signals during the pusher execution.
func (p traceV2DataPusher) withObservability(exporterName string) traceV2DataPusher {
	return func(ctx context.Context, td data.TraceData) (int, error) {
		exporterCtx, span := obsreport.StartTraceDataExportOp(ctx, exporterName)
		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(exporterCtx, td)

		// TODO: this is not ideal: it should come from the next function itself.
		// 	temporarily loading it from internal format. Once full switch is done
		// 	to new metrics will remove this.
		numSpans := td.SpanCount()
		obsreport.EndTraceDataExportOp(exporterCtx, span, numSpans, droppedSpans, err)
		return droppedSpans, err
	}
}
