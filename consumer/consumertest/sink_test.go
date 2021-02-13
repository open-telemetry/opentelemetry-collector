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

package consumertest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTracesSink(t *testing.T) {
	sink := new(TracesSink)
	want := make([]pdata.Traces, 0, 7)
	for i := 0; i < 7; i++ {
		td := testdata.GenerateTraceDataOneSpan()
		require.NoError(t, sink.ConsumeTraces(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpansCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllTraces()))
	assert.Equal(t, 0, sink.SpansCount())
}

func TestTracesSink_CopiesData(t *testing.T) {
	sink := new(TracesSink)
	require.NoError(t, sink.ConsumeTraces(context.Background(), testdata.GenerateTraceDataOneSpan()))
	allTraces := sink.AllTraces()
	require.Len(t, allTraces, 1)
	allTraces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetName("testing")
	assert.NotEqual(t, allTraces, sink.AllTraces())
}

func TestTracesSink_Error(t *testing.T) {
	sink := new(TracesSink)
	sink.SetConsumeError(errors.New("my error"))
	td := testdata.GenerateTraceDataOneSpan()
	require.Error(t, sink.ConsumeTraces(context.Background(), td))
	assert.Len(t, sink.AllTraces(), 0)
	assert.Equal(t, 0, sink.SpansCount())
}

func TestMetricsSink(t *testing.T) {
	sink := new(MetricsSink)
	want := make([]pdata.Metrics, 0, 7)
	for i := 0; i < 7; i++ {
		md := testdata.GenerateMetricsOneMetric()
		require.NoError(t, sink.ConsumeMetrics(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, len(want), sink.MetricsCount())
	sink.Reset()
	assert.Len(t, sink.AllMetrics(), 0)
	assert.Equal(t, 0, sink.MetricsCount())
}

func TestMetricsSink_CopiesData(t *testing.T) {
	sink := new(MetricsSink)
	require.NoError(t, sink.ConsumeMetrics(context.Background(), testdata.GenerateMetricsOneMetric()))
	allMetrics := sink.AllMetrics()
	require.Len(t, allMetrics, 1)
	allMetrics[0].ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).SetName("testing")
	assert.NotEqual(t, allMetrics, sink.AllMetrics())
}

func TestMetricsSink_Error(t *testing.T) {
	sink := new(MetricsSink)
	sink.SetConsumeError(errors.New("my error"))
	md := testdata.GenerateMetricsOneMetric()
	require.Error(t, sink.ConsumeMetrics(context.Background(), md))
	assert.Len(t, sink.AllMetrics(), 0)
	assert.Equal(t, 0, sink.MetricsCount())
}

func TestLogsSink(t *testing.T) {
	sink := new(LogsSink)
	want := make([]pdata.Logs, 0, 7)
	for i := 0; i < 7; i++ {
		md := testdata.GenerateLogDataOneLogNoResource()
		require.NoError(t, sink.ConsumeLogs(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordsCount())
	sink.Reset()
	assert.Len(t, sink.AllLogs(), 0)
	assert.Equal(t, 0, sink.LogRecordsCount())
}

func TestLogsSink_CopiesData(t *testing.T) {
	sink := new(LogsSink)
	require.NoError(t, sink.ConsumeLogs(context.Background(), testdata.GenerateLogDataOneLogNoResource()))
	allLogs := sink.AllLogs()
	require.Len(t, allLogs, 1)
	allLogs[0].ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).SetName("testing")
	assert.NotEqual(t, allLogs, sink.AllLogs())
}

func TestLogsSink_Error(t *testing.T) {
	sink := new(LogsSink)
	sink.SetConsumeError(errors.New("my error"))
	ld := testdata.GenerateLogDataOneLogNoResource()
	require.Error(t, sink.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sink.AllLogs(), 0)
	assert.Equal(t, 0, sink.LogRecordsCount())
}
