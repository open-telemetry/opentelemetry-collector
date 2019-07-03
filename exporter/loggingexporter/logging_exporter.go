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

package loggingexporter

import (
	"context"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterhelper"
	"go.uber.org/zap"
)

const (
	traceExportFormat   = "logging_trace"
	metricsExportFormat = "logging_metrics"
)

// NewTraceExporter creates an exporter.TraceExporter that just drops the
// received data and logs debugging messages.
func NewTraceExporter(logger *zap.Logger) (exporter.TraceExporter, error) {
	return exporterhelper.NewTraceExporter(
		traceExportFormat,
		func(ctx context.Context, td consumerdata.TraceData) (int, error) {
			logger.Debug("loggingTraceExporter", zap.Int("#spans", len(td.Spans)))
			// TODO: Add ability to record the received data
			return 0, nil
		},
		exporterhelper.WithSpanName("LoggingExporter.ConsumeTraceData"), exporterhelper.WithRecordMetrics(true),
	)
}

// NewMetricsExporter creates an exporter.MetricsExporter that just drops the
// received data and logs debugging messages.
func NewMetricsExporter(logger *zap.Logger) (exporter.MetricsExporter, error) {
	return exporterhelper.NewMetricsExporter(
		metricsExportFormat,
		func(ctx context.Context, md consumerdata.MetricsData) (int, error) {
			logger.Debug("loggingMetricsExporter", zap.Int("#metrics", len(md.Metrics)))
			// TODO: Add ability to record the received data
			return 0, nil
		},
		exporterhelper.WithSpanName("LoggingExporter.ConsumeMetricsData"), exporterhelper.WithRecordMetrics(true),
	)
}
