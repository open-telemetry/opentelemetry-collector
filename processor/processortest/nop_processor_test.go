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
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
)

func TestNopTraceProcessorNoErrors(t *testing.T) {
	sink := new(exportertest.SinkTraceExporter)
	ntp := NewNopTraceProcessor(sink)
	want := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}

	assert.NoError(t, ntp.ConsumeTraceData(context.Background(), want))

	got := sink.AllTraces()[0]
	assert.Equal(t, want, got)
}

func TestNopMetricsProcessorNoErrors(t *testing.T) {
	sink := new(exportertest.SinkMetricsExporter)
	nmp := NewNopMetricsProcessor(sink)
	want := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}

	assert.NoError(t, nmp.ConsumeMetricsData(context.Background(), want))

	got := sink.AllMetrics()[0]
	assert.Equal(t, want, got)
}

func TestNopProcessorFactory(t *testing.T) {
	f := &NopProcessorFactory{}
	cfg := f.CreateDefaultConfig()

	tp, err := f.CreateTraceProcessor(
		zap.NewNop(),
		new(exportertest.SinkTraceExporter),
		cfg)
	assert.NoError(t, err)
	assert.NotNil(t, tp)

	mp, err := f.CreateMetricsProcessor(
		zap.NewNop(),
		new(exportertest.SinkMetricsExporter),
		cfg)
	assert.NoError(t, err)
	assert.NotNil(t, mp)
}
