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
	"go.uber.org/zap"
)

func TestLoggingTraceExporterNoErrors(t *testing.T) {
	dtdp := NewTraceExporter(zap.NewNop())
	td := data.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	if err := dtdp.ProcessTraceData(context.Background(), td); err != nil {
		t.Errorf("Wanted nil got error")
		return
	}
	if "LoggingExporter" != dtdp.ExportFormat() {
		t.Errorf("Wanted LoggingExporter got %v", dtdp.ExportFormat())
	}
}

func TestLoggingMetricsExporterNoErrors(t *testing.T) {
	dmdp := NewMetricsExporter(zap.NewNop())
	md := data.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	if err := dmdp.ProcessMetricsData(context.Background(), md); err != nil {
		t.Errorf("Wanted nil got error")
		return
	}
	if "LoggingExporter" != dmdp.ExportFormat() {
		t.Errorf("Wanted LoggingExporter got %v", dmdp.ExportFormat())
	}
}
