// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestResourceMetricsToMetricsData(t *testing.T) {
	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	unixnanos1 := uint64(12578940000000012345)
	unixnanos2 := uint64(12578940000000054321)

	otlpMetric := &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_COUNTER_INT64,
		},
		Int64DataPoints: []*otlpmetrics.Int64DataPoint{
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				StartTimeUnixNano: unixnanos1,
				TimeUnixNano:      unixnanos2,
				Value:             123,
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key2",
						Value: "value2",
					},
				},
				StartTimeUnixNano: unixnanos1,
				TimeUnixNano:      unixnanos2,
				Value:             456,
			},
		},
		// NOTE: According to spec, only one type of data points can be set at a time,
		// but we still support translating "as is"
		DoubleDataPoints: []*otlpmetrics.DoubleDataPoint{
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "double-value1",
					},
					{
						Key:   "key3",
						Value: "double-value3",
					},
				},
				StartTimeUnixNano: unixnanos1,
				TimeUnixNano:      unixnanos2,
				Value:             1.23,
			},
		},
	}

	otlpAttributes := []*otlpcommon.AttributeKeyValue{
		{
			Key:         conventions.OCAttributeResourceType,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "good-resource",
		},
		{
			Key:         conventions.OCAttributeProcessStartTime,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "2020-02-11T20:26:00Z",
		},
		{
			Key:         conventions.AttributeHostHostname,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "host1",
		},
		{
			Key:         conventions.OCAttributeProcessID,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "123",
		},
		{
			Key:         conventions.AttributeLibraryVersion,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "v2.0.1",
		},
		{
			Key:         conventions.OCAttributeExporterVersion,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "v1.2.0",
		},
		{
			Key:         conventions.AttributeLibraryLanguage,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "CPP",
		},
		{
			Key:         "str1",
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "text",
		},
		{
			Key:      "int2",
			Type:     otlpcommon.AttributeKeyValue_INT,
			IntValue: 123,
		},
	}

	ocMetric := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "key1"},
				{Key: "key2"},
				{Key: "key3"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: unixnanoToTimestamp(unixnanos1),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "value1",
						HasValue: true,
					},
					{
						// key2
						HasValue: false,
					},
					{
						// key3
						HasValue: false,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: unixnanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 123,
						},
					},
				},
			},
			{
				StartTimestamp: unixnanoToTimestamp(unixnanos1),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						HasValue: false,
					},
					{
						// key2
						Value:    "value2",
						HasValue: true,
					},
					{
						// key3
						HasValue: false,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: unixnanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 456,
						},
					},
				},
			},
			{
				StartTimestamp: unixnanoToTimestamp(unixnanos1),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "double-value1",
						HasValue: true,
					},
					{
						// key2
						HasValue: false,
					},
					{
						// key3
						Value:    "double-value3",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: unixnanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_DoubleValue{
							DoubleValue: 1.23,
						},
					},
				},
			},
		},
	}

	ocAttributes := map[string]string{
		"str1": "text",
		"int2": "123",
	}

	tests := []struct {
		name string
		otlp otlpmetrics.ResourceMetrics
		oc   consumerdata.MetricsData
	}{
		{
			name: "empty",
			otlp: otlpmetrics.ResourceMetrics{},
			oc:   consumerdata.MetricsData{},
		},

		{
			name: "no-metrics",
			otlp: otlpmetrics.ResourceMetrics{
				Resource:                      &otlpresource.Resource{},
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{},
			},
			oc: consumerdata.MetricsData{
				Node:     &occommon.Node{},
				Resource: &ocresource.Resource{},
				Metrics:  nil,
			},
		},

		{
			name: "sample-metric",
			otlp: otlpmetrics.ResourceMetrics{
				Resource: &otlpresource.Resource{
					Attributes: otlpAttributes,
				},
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{otlpMetric},
					},
				},
			},
			oc: consumerdata.MetricsData{
				Node: &occommon.Node{
					Identifier: &occommon.ProcessIdentifier{
						HostName:       "host1",
						Pid:            123,
						StartTimestamp: ts,
					},
					LibraryInfo: &occommon.LibraryInfo{
						Language:           occommon.LibraryInfo_CPP,
						ExporterVersion:    "v1.2.0",
						CoreLibraryVersion: "v2.0.1",
					},
				},
				Resource: &ocresource.Resource{
					Type:   "good-resource",
					Labels: ocAttributes,
				},
				Metrics: []*ocmetrics.Metric{ocMetric},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResourceMetricsToMetricsData(&test.otlp)
			assert.EqualValues(t, test.oc, got)
		})
	}
}
