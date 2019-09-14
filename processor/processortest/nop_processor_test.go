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
package processortest

import (
	"context"
	"reflect"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
)

func TestNopTraceProcessorNoErrors(t *testing.T) {
	sink := new(exportertest.SinkTraceExporter)
	ntp := NewNopTraceProcessor(sink)
	want := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	if err := ntp.ConsumeTraceData(context.Background(), want); err != nil {
		t.Errorf("Wanted nil got error")
		return
	}
	got := sink.AllTraces()[0]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Mismatches responses\nGot:\n\t%v\nWant:\n\t%v\n", got, want)
	}
}

func TestNopMetricsProcessorNoErrors(t *testing.T) {
	sink := new(exportertest.SinkMetricsExporter)
	nmp := NewNopMetricsProcessor(sink)
	want := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	if err := nmp.ConsumeMetricsData(context.Background(), want); err != nil {
		t.Errorf("Wanted nil got error")
		return
	}
	got := sink.AllMetrics()[0]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Mismatches responses\nGot:\n\t%v\nWant:\n\t%v\n", got, want)
	}
}
