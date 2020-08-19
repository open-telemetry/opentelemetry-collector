// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package exportertest

import (
	"context"
	"errors"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestSinkTraceExporterOld(t *testing.T) {
	sink := new(SinkTraceExporterOld)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	want := make([]consumerdata.TraceData, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeTraceData(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllTraces())
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkTraceExporterOld_Error(t *testing.T) {
	sink := new(SinkTraceExporterOld)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetConsumeTraceError(errors.New("my error"))
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	require.Error(t, sink.ConsumeTraceData(context.Background(), td))
	assert.Len(t, sink.AllTraces(), 0)
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkTraceExporter(t *testing.T) {
	sink := new(SinkTraceExporter)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	td := testdata.GenerateTraceDataOneSpan()
	want := make([]pdata.Traces, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeTraces(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpansCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllTraces()))
	assert.Equal(t, 0, sink.SpansCount())
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkTraceExporter_Error(t *testing.T) {
	sink := new(SinkTraceExporter)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetConsumeTraceError(errors.New("my error"))
	td := testdata.GenerateTraceDataOneSpan()
	require.Error(t, sink.ConsumeTraces(context.Background(), td))
	assert.Len(t, sink.AllTraces(), 0)
	assert.Equal(t, 0, sink.SpansCount())
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkMetricsExporterOld(t *testing.T) {
	sink := new(SinkMetricsExporterOld)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	want := make([]consumerdata.MetricsData, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeMetricsData(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllMetrics())
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkMetricsExporterOld_Error(t *testing.T) {
	sink := new(SinkMetricsExporterOld)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetConsumeMetricsError(errors.New("my error"))
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	require.Error(t, sink.ConsumeMetricsData(context.Background(), md))
	assert.Len(t, sink.AllMetrics(), 0)
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkMetricsExporter(t *testing.T) {
	sink := new(SinkMetricsExporter)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	md := testdata.GenerateMetricDataOneMetric()
	want := make([]pdata.Metrics, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(md)))
		want = append(want, pdatautil.MetricsFromInternalMetrics(md))
	}
	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, len(want), sink.MetricsCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllMetrics()))
	assert.Equal(t, 0, sink.MetricsCount())
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkMetricsExporter_Error(t *testing.T) {
	sink := new(SinkMetricsExporter)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetConsumeMetricsError(errors.New("my error"))
	md := testdata.GenerateMetricDataOneMetric()
	require.Error(t, sink.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(md)))
	assert.Len(t, sink.AllMetrics(), 0)
	assert.Equal(t, 0, sink.MetricsCount())
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkLogsExporter(t *testing.T) {
	sink := new(SinkLogsExporter)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	md := testdata.GenerateLogDataOneLogNoResource()
	want := make([]pdata.Logs, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeLogs(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordsCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllLogs()))
	assert.Equal(t, 0, sink.LogRecordsCount())
	require.NoError(t, sink.Shutdown(context.Background()))
}

func TestSinkLogsExporter_Error(t *testing.T) {
	sink := new(SinkLogsExporter)
	require.NoError(t, sink.Start(context.Background(), componenttest.NewNopHost()))
	sink.SetConsumeLogError(errors.New("my error"))
	ld := testdata.GenerateLogDataOneLogNoResource()
	require.Error(t, sink.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sink.AllLogs(), 0)
	assert.Equal(t, 0, sink.LogRecordsCount())
	require.NoError(t, sink.Shutdown(context.Background()))
}
