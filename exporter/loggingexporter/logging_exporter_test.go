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
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/observability/observabilitytest"
	"go.uber.org/zap"
)

func TestLoggingTraceExporterNoErrors(t *testing.T) {
	lte := NewTraceExporter(zap.NewNop())
	td := data.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	if err := lte.ConsumeTraceData(context.Background(), td); err != nil {
		t.Errorf("Wanted nil got error")
		return
	}
	if "logging_trace" != lte.TraceExportFormat() {
		t.Errorf("Wanted logging_trace got %v", lte.TraceExportFormat())
	}
}

func TestLoggingTraceExporterRecordMetrics(t *testing.T) {
	lte := NewTraceExporter(zap.NewNop())
	observabilitytest.CheckRecordedMetricsForTraceExporter(t, lte)
}

func TestLoggingMetricsExporterNoErrors(t *testing.T) {
	lme := NewMetricsExporter(zap.NewNop())
	md := data.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	if err := lme.ConsumeMetricsData(context.Background(), md); err != nil {
		t.Errorf("Wanted nil got error")
		return
	}
	if "logging_metrics" != lme.MetricsExportFormat() {
		t.Errorf("Wanted logging_metrics got %v", lme.MetricsExportFormat())
	}
}
