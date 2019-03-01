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
	"go.uber.org/zap"
)

const exportFormat = "LoggingExporter"

// A logging exporter that does not sends the data to any destination but logs debugging messages.
type loggingExporter struct{ logger *zap.Logger }

var _ exporter.TraceExporter = (*loggingExporter)(nil)
var _ exporter.MetricsExporter = (*loggingExporter)(nil)

func (sp *loggingExporter) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	// TODO: Record metrics
	// TODO: Add ability to record the received data
	sp.logger.Debug("loggingTraceExporter", zap.Int("#spans", len(td.Spans)))
	return nil
}

func (sp *loggingExporter) ProcessMetricsData(ctx context.Context, md data.MetricsData) error {
	sp.logger.Debug("loggingMetricsExporter", zap.Int("#metrics", len(md.Metrics)))
	// TODO: Record metrics
	// TODO: Add ability to record the received data
	return nil
}

func (sp *loggingExporter) ExportFormat() string {
	return exportFormat
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
