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

package otlp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestProtobufLogsUnmarshaler_error(t *testing.T) {
	p := NewProtobufLogsUnmarshaler()
	_, err := p.UnmarshalLogs([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtobufMetricsUnmarshaler_error(t *testing.T) {
	p := NewProtobufMetricsUnmarshaler()
	_, err := p.UnmarshalMetrics([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtobufTracesUnmarshaler_error(t *testing.T) {
	p := NewProtobufTracesUnmarshaler()
	_, err := p.UnmarshalTraces([]byte("+$%"))
	assert.Error(t, err)
}

func BenchmarkLogsToProtobuf(b *testing.B) {
	marshaler := NewProtobufLogsMarshaler()
	logs := generateBenchmarkLogs(128)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalLogs(logs)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkLogsFromProtobuf(b *testing.B) {
	marshaler := NewProtobufLogsMarshaler()
	unmarshaler := NewProtobufLogsUnmarshaler()
	baseLogs := generateBenchmarkLogs(128)
	buf, err := marshaler.MarshalLogs(baseLogs)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logs, err := unmarshaler.UnmarshalLogs(buf)
		require.NoError(b, err)
		assert.Equal(b, baseLogs.ResourceLogs().Len(), logs.ResourceLogs().Len())
	}
}

func BenchmarkMetricsToProtobuf(b *testing.B) {
	marshaler := NewProtobufMetricsMarshaler()
	metrics := generateBenchmarkMetrics(128)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalMetrics(metrics)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkMetricsFromProtobuf(b *testing.B) {
	marshaler := NewProtobufMetricsMarshaler()
	unmarshaler := NewProtobufMetricsUnmarshaler()
	baseMetrics := generateBenchmarkMetrics(128)
	buf, err := marshaler.MarshalMetrics(baseMetrics)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		metrics, err := unmarshaler.UnmarshalMetrics(buf)
		require.NoError(b, err)
		assert.Equal(b, baseMetrics.ResourceMetrics().Len(), metrics.ResourceMetrics().Len())
	}
}

func BenchmarkTracesToProtobuf(b *testing.B) {
	marshaler := NewProtobufTracesMarshaler()
	traces := generateBenchmarkTraces(128)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalTraces(traces)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkTracesFromProtobuf(b *testing.B) {
	marshaler := NewProtobufTracesMarshaler()
	unmarshaler := NewProtobufTracesUnmarshaler()
	baseTraces := generateBenchmarkTraces(128)
	buf, err := marshaler.MarshalTraces(baseTraces)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		traces, err := unmarshaler.UnmarshalTraces(buf)
		require.NoError(b, err)
		assert.Equal(b, baseTraces.ResourceSpans().Len(), traces.ResourceSpans().Len())
	}
}

func generateBenchmarkLogs(logsCount int) pdata.Logs {
	endTime := pdata.TimestampFromTime(time.Now())

	md := pdata.NewLogs()
	ilm := md.ResourceLogs().AppendEmpty().InstrumentationLibraryLogs().AppendEmpty()
	ilm.Logs().Resize(logsCount)
	for i := 0; i < logsCount; i++ {
		im := ilm.Logs().At(i)
		im.SetName("test_name")
		im.SetTimestamp(endTime)
	}
	return md
}

func generateBenchmarkMetrics(metricsCount int) pdata.Metrics {
	now := time.Now()
	startTime := pdata.TimestampFromTime(now.Add(-10 * time.Second))
	endTime := pdata.TimestampFromTime(now)

	md := pdata.NewMetrics()
	ilm := md.ResourceMetrics().AppendEmpty().InstrumentationLibraryMetrics().AppendEmpty()
	ilm.Metrics().Resize(metricsCount)
	for i := 0; i < metricsCount; i++ {
		im := ilm.Metrics().At(i)
		im.SetName("test_name")
		im.SetDataType(pdata.MetricDataTypeIntSum)
		idp := im.IntSum().DataPoints().AppendEmpty()
		idp.SetStartTimestamp(startTime)
		idp.SetTimestamp(endTime)
		idp.SetValue(123)
	}
	return md
}

func generateBenchmarkTraces(metricsCount int) pdata.Traces {
	now := time.Now()
	startTime := pdata.TimestampFromTime(now.Add(-10 * time.Second))
	endTime := pdata.TimestampFromTime(now)

	md := pdata.NewTraces()
	ilm := md.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty()
	ilm.Spans().Resize(metricsCount)
	for i := 0; i < metricsCount; i++ {
		im := ilm.Spans().At(i)
		im.SetName("test_name")
		im.SetStartTimestamp(startTime)
		im.SetEndTimestamp(endTime)
	}
	return md
}
