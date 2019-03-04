// Copyright 2019, OpenCensus Authors
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

package loggingexporter

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/observability"
	"go.uber.org/zap"
)

const (
	traceExportFormat   = "logging_trace"
	metricsExportFormat = "logging_metrics"
)

// A logging exporter that does not sends the data to any destination but logs debugging messages.
type loggingExporter struct{ logger *zap.Logger }

var _ exporter.TraceExporter = (*loggingExporter)(nil)
var _ exporter.MetricsExporter = (*loggingExporter)(nil)

func (le *loggingExporter) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	le.logger.Debug("loggingTraceExporter", zap.Int("#spans", len(td.Spans)))
	// TODO: Add ability to record the received data

	// Even though we just log all the spans, we record 0 dropped spans.
	observability.RecordTraceExporterMetrics(observability.ContextWithExporterName(ctx, traceExportFormat), len(td.Spans), 0)
	return nil
}

func (le *loggingExporter) ProcessMetricsData(ctx context.Context, md data.MetricsData) error {
	le.logger.Debug("loggingMetricsExporter", zap.Int("#metrics", len(md.Metrics)))
	// TODO: Add ability to record the received data
	// TODO: Record metrics
	return nil
}

func (le *loggingExporter) TraceExportFormat() string {
	return traceExportFormat
}

func (le *loggingExporter) MetricsExportFormat() string {
	return metricsExportFormat
}

// NewTraceExporter creates an exporter.TraceExporter that just drops the
// received data and logs debugging messages.
func NewTraceExporter(logger *zap.Logger) exporter.TraceExporter {
	return &loggingExporter{logger: logger}
}

// NewMetricsExporter creates an exporter.MetricsExporter that just drops the
// received data and logs debugging messages.
func NewMetricsExporter(logger *zap.Logger) exporter.MetricsExporter {
	return &loggingExporter{logger: logger}
}
