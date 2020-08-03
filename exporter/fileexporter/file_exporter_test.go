// Copyright The OpenTelemetry Authors
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
package fileexporter

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	logspb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
	otresourcepb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
	"go.opentelemetry.io/collector/testutil"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func TestFileTraceExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	lte := &Exporter{file: mf}
	require.NotNil(t, lte)

	td := consumerdata.TraceData{
		Resource: &resourcepb.Resource{
			Type:   "ServiceA",
			Labels: map[string]string{"attr1": "value1"},
		},
		Spans: []*tracepb.Span{
			{
				StartTime: internal.TimeToTimestamp(time.Unix(123, 0)),
				EndTime:   internal.TimeToTimestamp(time.Unix(124, 0)),
				TraceId:   []byte("123"),
				SpanId:    []byte("456"),
				Name:      &tracepb.TruncatableString{Value: "Checkout"},
				Kind:      tracepb.Span_CLIENT,
			},
			{
				StartTime: internal.TimeToTimestamp(time.Unix(123, 0)),
				EndTime:   internal.TimeToTimestamp(time.Unix(124, 0)),
				Name:      &tracepb.TruncatableString{Value: "Frontend"},
				Kind:      tracepb.Span_SERVER,
			},
		},
	}
	assert.NoError(t, lte.ConsumeTraces(context.Background(), internaldata.OCToTraceData(td)))
	assert.NoError(t, lte.Shutdown(context.Background()))

	var j map[string]interface{}
	assert.NoError(t, json.Unmarshal(mf.Bytes(), &j))

	assert.EqualValues(t,
		map[string]interface{}{
			"node": map[string]interface{}{},
			"resource": map[string]interface{}{
				"type":   "ServiceA",
				"labels": map[string]interface{}{"attr1": "value1"},
			},
			"spans": []interface{}{
				map[string]interface{}{
					"startTime": map[string]interface{}{"seconds": "123"},
					"endTime":   map[string]interface{}{"seconds": "124"},
					"traceId":   "MTIz", // base64 encoding of "123"
					"spanId":    "NDU2", // base64 encoding of "456"
					"kind":      "CLIENT",
					"name":      map[string]interface{}{"value": "Checkout"},
				},
				map[string]interface{}{
					"startTime": map[string]interface{}{"seconds": "123"},
					"endTime":   map[string]interface{}{"seconds": "124"},
					"kind":      "SERVER",
					"name":      map[string]interface{}{"value": "Frontend"},
				},
			},
		}, j)
}

func TestFileMetricsExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	lme := &Exporter{file: mf}
	require.NotNil(t, lme)

	md := pdatautil.MetricsFromMetricsData([]consumerdata.MetricsData{
		{
			Resource: &resourcepb.Resource{
				Type:   "ServiceA",
				Labels: map[string]string{"attr1": "value1"},
			},
			Metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:        "my-metric",
						Description: "My metric",
						Type:        metricspb.MetricDescriptor_GAUGE_INT64,
					},
					Timeseries: []*metricspb.TimeSeries{
						{
							Points: []*metricspb.Point{
								{Value: &metricspb.Point_Int64Value{Int64Value: 123}},
							},
						},
					},
				},
			},
		},
	})
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, lme.Shutdown(context.Background()))

	var j map[string]interface{}
	assert.NoError(t, json.Unmarshal(mf.Bytes(), &j))

	assert.EqualValues(t, j,
		map[string]interface{}{
			"resource": map[string]interface{}{
				"type":   "ServiceA",
				"labels": map[string]interface{}{"attr1": "value1"},
			},
			"metrics": []interface{}{
				map[string]interface{}{
					"metricDescriptor": map[string]interface{}{
						"name":        "my-metric",
						"description": "My metric",
						"type":        "GAUGE_INT64",
					},
					"timeseries": []interface{}{
						map[string]interface{}{
							"points": []interface{}{
								map[string]interface{}{
									"int64Value": "123",
								},
							},
						},
					},
				},
			},
		})
}

func TestFileLogsExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	exporter := &Exporter{file: mf}
	require.NotNil(t, exporter)

	now := time.Now()
	ld := []*logspb.ResourceLogs{
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr1",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logA",
						},
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logB",
						},
					},
				},
			},
		},
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr2",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logC",
						},
					},
				},
			},
		},
	}
	assert.NoError(t, exporter.ConsumeLogs(context.Background(), pdata.LogsFromOtlp(ld)))
	assert.NoError(t, exporter.Shutdown(context.Background()))

	decoder := json.NewDecoder(mf)
	var j map[string]interface{}
	assert.NoError(t, decoder.Decode(&j))

	assert.EqualValues(t,
		map[string]interface{}{
			"resource": map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key": "attr1",
						"value": map[string]interface{}{
							"stringValue": "value1",
						},
					},
				},
			},
			"logs": []interface{}{
				map[string]interface{}{
					"timeUnixNano": strconv.Itoa(int(now.UnixNano())),
					"name":         "logA",
				},
				map[string]interface{}{
					"timeUnixNano": strconv.Itoa(int(now.UnixNano())),
					"name":         "logB",
				},
			},
		}, j)

	require.NoError(t, decoder.Decode(&j))

	assert.EqualValues(t,
		map[string]interface{}{
			"resource": map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key": "attr2",
						"value": map[string]interface{}{
							"stringValue": "value2",
						},
					},
				},
			},
			"logs": []interface{}{
				map[string]interface{}{
					"timeUnixNano": strconv.Itoa(int(now.UnixNano())),
					"name":         "logC",
				},
			},
		}, j)
}

func TestFileLogsExporterErrors(t *testing.T) {

	now := time.Now()
	ld := []*logspb.ResourceLogs{
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr1",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logA",
						},
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logB",
						},
					},
				},
			},
		},
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr2",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logC",
						},
					},
				},
			},
		},
	}

	cases := []struct {
		Name   string
		MaxLen int
	}{
		{
			Name:   "opening",
			MaxLen: 1,
		},
		{
			Name:   "resource",
			MaxLen: 16,
		},
		{
			Name:   "log_start",
			MaxLen: 78,
		},
		{
			Name:   "logs",
			MaxLen: 128,
		},
	}

	for i := range cases {
		maxLen := cases[i].MaxLen
		t.Run(cases[i].Name, func(t *testing.T) {
			mf := &testutil.LimitedWriter{
				MaxLen: maxLen,
			}
			exporter := &Exporter{file: mf}
			require.NotNil(t, exporter)

			assert.Error(t, exporter.ConsumeLogs(context.Background(), pdata.LogsFromOtlp(ld)))
			assert.NoError(t, exporter.Shutdown(context.Background()))
		})
	}
}
