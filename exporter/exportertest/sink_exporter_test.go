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
package exportertest

import (
	"context"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

func TestSinkTraceExporter(t *testing.T) {
	sink := new(SinkTraceExporter)
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	want := make([]consumerdata.TraceData, 0, 7)
	for i := 0; i < 7; i++ {
		err := sink.ConsumeTraceData(context.Background(), td)
		require.Nil(t, err)
		want = append(want, td)
	}
	got := sink.AllTraces()
	assert.Equal(t, want, got)
}

func TestSinkMetricsExporter(t *testing.T) {
	sink := new(SinkMetricsExporter)
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	want := make([]consumerdata.MetricsData, 0, 7)
	for i := 0; i < 7; i++ {
		err := sink.ConsumeMetricsData(context.Background(), md)
		require.Nil(t, err)
		want = append(want, md)
	}
	got := sink.AllMetrics()
	assert.Equal(t, want, got)
}
