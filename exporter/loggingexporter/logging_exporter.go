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

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter"
	"github.com/open-telemetry/opentelemetry-service/exporter/exporterhelper"
)

// NewTraceExporter creates an exporter.TraceExporter that just drops the
// received data and logs debugging messages.
func NewTraceExporter(exporterName string, logger *zap.Logger) (exporter.TraceExporter, error) {
	tl := traceLogger{
		name:   exporterName,
		logger: logger,
	}
	return exporterhelper.NewTraceExporter(
		exporterName,
		tl.pushTraceData,
		tl.stop,
		exporterhelper.WithSpanName(exporterName+".ConsumeTraceData"), exporterhelper.WithRecordMetrics(true),
	)
}

// NewMetricsExporter creates an exporter.MetricsExporter that just drops the
// received data and logs debugging messages.
func NewMetricsExporter(exporterName string, logger *zap.Logger) (exporter.MetricsExporter, error) {
	ml := metricsLogger{
		name:   exporterName,
		logger: logger,
	}
	return exporterhelper.NewMetricsExporter(
		exporterName,
		ml.pushMetricsData,
		ml.stop,
		exporterhelper.WithSpanName(exporterName+".ConsumeMetricsData"), exporterhelper.WithRecordMetrics(true),
	)
}

type traceLogger struct {
	name   string
	logger *zap.Logger
}

func (tl *traceLogger) pushTraceData(
	ctx context.Context,
	td consumerdata.TraceData) (int, error) {
	tl.logger.Debug(tl.name, zap.Int("#spans", len(td.Spans)))
	// TODO: Add ability to record the received data
	return 0, nil
}

func (tl *traceLogger) stop() error {
	return nil
}

type metricsLogger struct {
	name   string
	logger *zap.Logger
}

func (ml *metricsLogger) pushMetricsData(ctx context.Context, md consumerdata.MetricsData) (int, error) {
	ml.logger.Debug(ml.name, zap.Int("#metrics", len(md.Metrics)))
	// TODO: Add ability to record the received data
	return 0, nil
}

func (ml *metricsLogger) stop() error {
	return nil
}
