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

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcollectorlogs "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpcollectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/collector/internal/data/protogen/common/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/protogen/logs/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/protogen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/protogen/resource/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/translator/conventions"
)

var tracesOTLP = &otlpcollectortrace.ExportTraceServiceRequest{
	ResourceSpans: []*otlptrace.ResourceSpans{
		{
			Resource: otlpresource.Resource{
				Attributes: []otlpcommon.KeyValue{
					{
						Key:   conventions.AttributeHostName,
						Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "testHost"}},
					},
				},
			},
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					InstrumentationLibrary: otlpcommon.InstrumentationLibrary{
						Name:    "name",
						Version: "version",
					},
					Spans: []*otlptrace.Span{
						{
							Name: "testSpan",
						},
					},
				},
			},
		},
	},
}

var tracesJSON = `{"resourceSpans":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"instrumentationLibrarySpans":[{"instrumentationLibrary":{"name":"name","version":"version"},"spans":[{"traceId":"","spanId":"","parentSpanId":"","name":"testSpan","status":{}}]}]}]}`

var metricsOTLP = &otlpcollectormetrics.ExportMetricsServiceRequest{
	ResourceMetrics: []*otlpmetrics.ResourceMetrics{
		{
			Resource: otlpresource.Resource{
				Attributes: []otlpcommon.KeyValue{
					{
						Key:   conventions.AttributeHostName,
						Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "testHost"}},
					},
				},
			},
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: otlpcommon.InstrumentationLibrary{
						Name:    "name",
						Version: "version",
					},
					Metrics: []*otlpmetrics.Metric{
						{
							Name: "testMetric",
						},
					},
				},
			},
		},
	},
}

var metricsJSON = `{"resourceMetrics":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"instrumentationLibraryMetrics":[{"instrumentationLibrary":{"name":"name","version":"version"},"metrics":[{"name":"testMetric"}]}]}]}`

var logsOTLP = &otlpcollectorlogs.ExportLogsServiceRequest{
	ResourceLogs: []*otlplogs.ResourceLogs{
		{
			Resource: otlpresource.Resource{
				Attributes: []otlpcommon.KeyValue{
					{
						Key:   conventions.AttributeHostName,
						Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "testHost"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					InstrumentationLibrary: otlpcommon.InstrumentationLibrary{
						Name:    "name",
						Version: "version",
					},
					Logs: []*otlplogs.LogRecord{
						{
							Name: "testMetric",
						},
					},
				},
			},
		},
	},
}

var logsJSON = `{"resourceLogs":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"instrumentationLibraryLogs":[{"instrumentationLibrary":{"name":"name","version":"version"},"logs":[{"name":"testMetric","body":{},"traceId":"","spanId":""}]}]}]}`

func TestTracesJSON(t *testing.T) {
	encoder := newJSONEncoder()
	jsonBuf, err := encoder.EncodeTraces(tracesOTLP)
	assert.NoError(t, err)

	decoder := newJSONDecoder()
	var got interface{}
	got, err = decoder.DecodeTraces(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, tracesOTLP, got)
}

func TestMetricsJSON(t *testing.T) {
	encoder := newJSONEncoder()
	jsonBuf, err := encoder.EncodeMetrics(metricsOTLP)
	assert.NoError(t, err)

	decoder := newJSONDecoder()
	var got interface{}
	got, err = decoder.DecodeMetrics(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, metricsOTLP, got)
}

func TestLogsJSON(t *testing.T) {
	encoder := newJSONEncoder()
	jsonBuf, err := encoder.EncodeLogs(logsOTLP)
	assert.NoError(t, err)

	decoder := newJSONDecoder()
	var got interface{}
	got, err = decoder.DecodeLogs(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, logsOTLP, got)
}

func TestTracesJSON_InvalidInterface(t *testing.T) {
	encoder := newJSONEncoder()
	_, err := encoder.EncodeTraces(pdata.NewTraces())
	assert.Error(t, err)
}

func TestMetricsJSON_InvalidInterface(t *testing.T) {
	encoder := newJSONEncoder()
	_, err := encoder.EncodeMetrics(pdata.NewMetrics())
	assert.Error(t, err)
}

func TestLogsJSON_InvalidInterface(t *testing.T) {
	encoder := newJSONEncoder()
	_, err := encoder.EncodeLogs(pdata.NewLogs())
	assert.Error(t, err)
}

func TestTracesJSON_Encode(t *testing.T) {
	encoder := newJSONEncoder()
	jsonBuf, err := encoder.EncodeTraces(tracesOTLP)
	assert.NoError(t, err)
	assert.Equal(t, tracesJSON, string(jsonBuf))
}

func TestMetricsJSON_Encode(t *testing.T) {
	encoder := newJSONEncoder()
	jsonBuf, err := encoder.EncodeMetrics(metricsOTLP)
	assert.NoError(t, err)
	assert.Equal(t, metricsJSON, string(jsonBuf))
}

func TestLogsJSON_Encode(t *testing.T) {
	encoder := newJSONEncoder()
	jsonBuf, err := encoder.EncodeLogs(logsOTLP)
	assert.NoError(t, err)
	assert.Equal(t, logsJSON, string(jsonBuf))
}
