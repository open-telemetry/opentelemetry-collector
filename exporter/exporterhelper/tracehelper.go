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

	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/observability"
)

// Suffix to use for span names emitted by exporter for observability purposes.
const spanNameSuffix = ".ExportTraceData"

// traceDataPusher is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type traceDataPusher func(ctx context.Context, td consumerdata.TraceData) (droppedSpans int, err error)

// otlpTraceDataPusher is a helper function that is similar to ConsumeTraceData but also
// returns the number of dropped spans.
type otlpTraceDataPusher func(ctx context.Context, td consumerdata.OTLPTraceData) (droppedSpans int, err error)

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
	exporterCtx := observability.ContextWithExporterName(ctx, te.exporterFullName)
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
	if opts.recordMetrics {
		dataPusher = dataPusher.withMetrics()
	}

	if opts.recordTrace {
		spanName := config.Name() + spanNameSuffix
		dataPusher = dataPusher.withSpan(spanName)
	}

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

// withMetrics wraps the current pusher into a function that records the metrics of the
// pusher execution.
func (p traceDataPusher) withMetrics() traceDataPusher {
	return func(ctx context.Context, td consumerdata.TraceData) (int, error) {
		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(ctx, td)
		// TODO: How to record the reason of dropping?
		observability.RecordMetricsForTraceExporter(ctx, len(td.Spans), droppedSpans)
		return droppedSpans, err
	}
}

// withSpan wraps the current pusher into a function that records a span during
// pusher execution.
func (p traceDataPusher) withSpan(spanName string) traceDataPusher {
	return func(ctx context.Context, td consumerdata.TraceData) (int, error) {
		ctx, span := trace.StartSpan(ctx, spanName)
		defer span.End()
		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(ctx, td)
		if span.IsRecordingEvents() {
			span.AddAttributes(
				trace.Int64Attribute(numReceivedSpansAttribute, int64(len(td.Spans))),
				trace.Int64Attribute(numDroppedSpansAttribute, int64(droppedSpans)),
			)
			if err != nil {
				span.SetStatus(errToStatus(err))
			}
		}
		return droppedSpans, err
	}
}

type otlpTraceExporter struct {
	exporterFullName string
	dataPusher       otlpTraceDataPusher
	shutdown         Shutdown
}

var _ exporter.OTLPTraceExporter = (*otlpTraceExporter)(nil)

func (te *otlpTraceExporter) Start(host component.Host) error {
	return nil
}

func (te *otlpTraceExporter) ConsumeOTLPTrace(
	ctx context.Context,
	td consumerdata.OTLPTraceData,
) error {
	exporterCtx := observability.ContextWithExporterName(ctx, te.exporterFullName)
	_, err := te.dataPusher(exporterCtx, td)
	return err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (te *otlpTraceExporter) Shutdown() error {
	return te.shutdown()
}

// NewOTLPTraceExporter creates an OTLPTraceExporter that can record metrics and can wrap
// every request with a Span.
func NewOTLPTraceExporter(
	config configmodels.Exporter,
	dataPusher otlpTraceDataPusher,
	options ...ExporterOption,
) (exporter.OTLPTraceExporter, error) {

	if config == nil {
		return nil, errNilConfig
	}

	if dataPusher == nil {
		return nil, errNilPushTraceData
	}

	opts := newExporterOptions(options...)
	if opts.recordMetrics {
		dataPusher = dataPusher.withMetrics()
	}

	if opts.recordTrace {
		spanName := config.Name() + spanNameSuffix
		dataPusher = dataPusher.withSpan(spanName)
	}

	// The default shutdown function does nothing.
	if opts.shutdown == nil {
		opts.shutdown = func() error {
			return nil
		}
	}

	return &otlpTraceExporter{
		exporterFullName: config.Name(),
		dataPusher:       dataPusher,
		shutdown:         opts.shutdown,
	}, nil
}

// withMetrics wraps the current pusher into a function that records the metrics of the
// pusher execution.
func (p otlpTraceDataPusher) withMetrics() otlpTraceDataPusher {
	return func(ctx context.Context, td consumerdata.OTLPTraceData) (int, error) {
		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(ctx, td)

		// Record the results as metrics.
		observability.RecordMetricsForTraceExporter(ctx, td.SpanCount(), droppedSpans)

		return droppedSpans, err
	}
}

// withSpan wraps the current pusher into a function that records a span during
// pusher execution.
func (p otlpTraceDataPusher) withSpan(spanName string) otlpTraceDataPusher {
	return func(ctx context.Context, td consumerdata.OTLPTraceData) (int, error) {
		// Start a span.
		ctx, span := trace.StartSpan(ctx, spanName)

		// End the span after this function is done.
		defer span.End()

		// Forward the data to the next consumer (this pusher is the next).
		droppedSpans, err := p(ctx, td)
		if span.IsRecordingEvents() {
			span.AddAttributes(
				trace.Int64Attribute(numReceivedSpansAttribute, int64(td.SpanCount())),
				trace.Int64Attribute(numDroppedSpansAttribute, int64(droppedSpans)),
			)
			if err != nil {
				span.SetStatus(errToStatus(err))
			}
		}
		return droppedSpans, err
	}
}
