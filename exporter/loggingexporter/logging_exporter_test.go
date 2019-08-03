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
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"go.uber.org/zap"
)

func TestLoggingTraceExporterNoErrors(t *testing.T) {
	const exporterName = "test_logging_exporter"
	lte, err := NewTraceExporter(exporterName, zap.NewNop())
	if err != nil {
		t.Fatalf("Wanted nil got %v", err)
	}
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	if err := lte.ConsumeTraceData(context.Background(), td); err != nil {
		t.Fatalf("Wanted nil got %v", err)
	}
	if lte.Name() != exporterName {
		t.Errorf("Wanted %q got %q", exporterName, lte.Name())
	}
}

func TestLoggingMetricsExporterNoErrors(t *testing.T) {
	const exporterName = "test_metrics_exporter"
	lme, err := NewMetricsExporter(exporterName, zap.NewNop())
	if err != nil {
		t.Fatalf("Wanted nil got %v", err)
	}
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	if err := lme.ConsumeMetricsData(context.Background(), md); err != nil {
		t.Fatalf("Wanted nil got %v", err)
	}
	if lme.Name() != exporterName {
		t.Errorf("Wanted %q got %q", exporterName, lme.Name())
	}
}
