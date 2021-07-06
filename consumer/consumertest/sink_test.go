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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestTracesSink(t *testing.T) {
	sink := new(TracesSink)
	td := testdata.GenerateTracesOneSpan()
	want := make([]pdata.Traces, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeTraces(context.Background(), td))
		want = append(want, td)
	}
	assert.Equal(t, want, sink.AllTraces())
	assert.Equal(t, len(want), sink.SpanCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllTraces()))
	assert.Equal(t, 0, sink.SpanCount())
}

func TestMetricsSink(t *testing.T) {
	sink := new(MetricsSink)
	md := testdata.GenerateMetricsOneMetric()
	want := make([]pdata.Metrics, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeMetrics(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllMetrics())
	assert.Equal(t, 2*len(want), sink.DataPointCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllMetrics()))
	assert.Equal(t, 0, sink.DataPointCount())
}

func TestLogsSink(t *testing.T) {
	sink := new(LogsSink)
	md := testdata.GenerateLogsOneLogRecord()
	want := make([]pdata.Logs, 0, 7)
	for i := 0; i < 7; i++ {
		require.NoError(t, sink.ConsumeLogs(context.Background(), md))
		want = append(want, md)
	}
	assert.Equal(t, want, sink.AllLogs())
	assert.Equal(t, len(want), sink.LogRecordCount())
	sink.Reset()
	assert.Equal(t, 0, len(sink.AllLogs()))
	assert.Equal(t, 0, sink.LogRecordCount())
}
