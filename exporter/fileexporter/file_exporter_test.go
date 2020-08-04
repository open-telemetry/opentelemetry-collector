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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	logspb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
	otresourcepb "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/testutil"
)

func TestFileTraceExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	lte := &Exporter{file: mf}
	require.NotNil(t, lte)

	td := testdata.GenerateTraceDataTwoSpansSameResource()

	assert.NoError(t, lte.ConsumeTraces(context.Background(), td))
	assert.NoError(t, lte.Shutdown(context.Background()))

	var j map[string]interface{}
	assert.NoError(t, json.Unmarshal(mf.Bytes(), &j))

	assert.EqualValues(t,
		map[string]interface{}{
			"resource": map[string]interface{}{
				"attributes": []interface{}{map[string]interface{}{
					"key": "resource-attr",
					"value": map[string]interface{}{
						"stringValue": "resource-attr-val-1"}}}},
			"instrumentationLibrarySpans": []interface{}{
				map[string]interface{}{
					"spans": []interface{}{
						map[string]interface{}{
							"name":              "operationA",
							"startTimeUnixNano": "1581452772000000321",
							"status": map[string]interface{}{
								"code":    "Cancelled",
								"message": "status-cancelled"},
							"droppedAttributesCount": float64(1),
							"droppedEventsCount":     float64(1),
							"endTimeUnixNano":        "1581452773000000789",
							"events": []interface{}{
								map[string]interface{}{
									"attributes": []interface{}{
										map[string]interface{}{
											"key": "span-event-attr",
											"value": map[string]interface{}{
												"stringValue": "span-event-attr-val"}}},
									"droppedAttributesCount": float64(2),
									"name":                   "event-with-attr",
									"timeUnixNano":           "1581452773000000123"},
								map[string]interface{}{
									"droppedAttributesCount": float64(2),
									"name":                   "event",
									"timeUnixNano":           "1581452773000000123"}}},
						map[string]interface{}{
							"name":              "operationB",
							"startTimeUnixNano": "1581452772000000321",
							"droppedLinksCount": float64(3),
							"endTimeUnixNano":   "1581452773000000789",
							"links": []interface{}{
								map[string]interface{}{
									"attributes": []interface{}{
										map[string]interface{}{
											"key": "span-link-attr",
											"value": map[string]interface{}{
												"stringValue": "span-link-attr-val"}}},
									"droppedAttributesCount": float64(4)},
								map[string]interface{}{
									"droppedAttributesCount": float64(4)}}},
					}}}}, j)
}

func TestFileMetricsExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	lme := &Exporter{file: mf}
	require.NotNil(t, lme)

	md := pdatautil.MetricsFromInternalMetrics(testdata.GenerateMetricDataTwoMetrics())
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, lme.Shutdown(context.Background()))

	var j map[string]interface{}
	assert.NoError(t, json.Unmarshal(mf.Bytes(), &j))

	assert.EqualValues(t,
		map[string]interface{}{
			"resource": map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key": "resource-attr",
						"value": map[string]interface{}{
							"stringValue": "resource-attr-val-1"}}}},
			"instrumentationLibraryMetrics": []interface{}{
				map[string]interface{}{
					"metrics": []interface{}{
						map[string]interface{}{
							"int64DataPoints": []interface{}{
								map[string]interface{}{
									"labels": []interface{}{
										map[string]interface{}{
											"key":   "label-1",
											"value": "label-value-1"}},
									"startTimeUnixNano": "1581452772000000321",
									"timeUnixNano":      "1581452773000000789",
									"value":             "123"},
								map[string]interface{}{
									"labels": []interface{}{
										map[string]interface{}{
											"key":   "label-2",
											"value": "label-value-2"}},
									"startTimeUnixNano": "1581452772000000321",
									"timeUnixNano":      "1581452773000000789",
									"value":             "456"}},
							"metricDescriptor": map[string]interface{}{
								"name": "counter-int",
								"type": "MONOTONIC_INT64",
								"unit": "1"}},
						map[string]interface{}{
							"int64DataPoints": []interface{}{
								map[string]interface{}{
									"labels": []interface{}{
										map[string]interface{}{
											"key":   "label-1",
											"value": "label-value-1"}},
									"startTimeUnixNano": "1581452772000000321",
									"timeUnixNano":      "1581452773000000789",
									"value":             "123"},
								map[string]interface{}{
									"labels": []interface{}{
										map[string]interface{}{
											"key":   "label-2",
											"value": "label-value-2"}},
									"startTimeUnixNano": "1581452772000000321",
									"timeUnixNano":      "1581452773000000789",
									"value":             "456"}},
							"metricDescriptor": map[string]interface{}{
								"name": "counter-int",
								"type": "MONOTONIC_INT64",
								"unit": "1"}}}}}}, j)
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
