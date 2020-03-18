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

package internaldata

import (
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestResourceMetricsToMetricsData(t *testing.T) {
	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	ts1 := data.TimestampUnixNano(12578940000000012345)
	ts2 := data.TimestampUnixNano(12578940000000054321)

	metrics := data.NewMetricSlice(1)
	metricDescriptor := data.NewMetricDescriptor()
	metricDescriptor.SetName("mymetric")
	metricDescriptor.SetDescription("My metric")
	metricDescriptor.SetUnit("ms")
	metricDescriptor.SetMetricType(data.MetricTypeCounterInt64)
	metrics[0].SetMetricDescriptor(metricDescriptor)

	int64DataPoints := data.NewInt64DataPointSlice(2)
	int64DataPoints[0].SetLabelsMap(data.NewStringMap(map[string]string{"key1": "value1"}))
	int64DataPoints[0].SetStartTime(ts1)
	int64DataPoints[0].SetTimestamp(ts2)
	int64DataPoints[0].SetValue(123)
	int64DataPoints[1].SetLabelsMap(data.NewStringMap(map[string]string{"key2": "value2"}))
	int64DataPoints[1].SetStartTime(ts1)
	int64DataPoints[1].SetTimestamp(ts2)
	int64DataPoints[1].SetValue(456)
	metrics[0].SetInt64DataPoints(int64DataPoints)

	doubleDataPoints := data.NewDoubleDataPointSlice(1)
	doubleDataPoints[0].SetLabelsMap(data.NewStringMap(map[string]string{
		"key1": "double-value1",
		"key3": "double-value3",
	}))
	doubleDataPoints[0].SetStartTime(ts1)
	doubleDataPoints[0].SetTimestamp(ts2)
	doubleDataPoints[0].SetValue(1.23)
	metrics[0].SetDoubleDataPoints(doubleDataPoints)

	resource := data.NewResource()
	attrs := data.AttributesMap{
		conventions.OCAttributeProcessStartTime: data.NewAttributeValueString("2020-02-11T20:26:00Z"),
		conventions.AttributeHostHostname:       data.NewAttributeValueString("host1"),
		conventions.OCAttributeProcessID:        data.NewAttributeValueString("123"),
		conventions.AttributeLibraryVersion:     data.NewAttributeValueString("v2.0.1"),
		conventions.OCAttributeExporterVersion:  data.NewAttributeValueString("v1.2.0"),
		conventions.AttributeLibraryLanguage:    data.NewAttributeValueString("CPP"),
		conventions.OCAttributeResourceType:     data.NewAttributeValueString("good-resource"),
		"str1":                                  data.NewAttributeValueString("text"),
		"int2":                                  data.NewAttributeValueInt(123),
	}
	resource.SetAttributes(attrs)

	ilm := data.NewInstrumentationLibraryMetricsSlice(1)
	ilm[0].SetMetrics(metrics)
	resourceMetricsSlice := data.NewResourceMetricsSlice(1)
	resourceMetricsSlice[0].SetInstrumentationLibraryMetrics(ilm)
	resourceMetricsSlice[0].SetResource(resource)

	metricData := data.NewMetricData()
	metricData.SetResourceMetrics(resourceMetricsSlice)

	emptyMetricData := data.NewMetricData()
	emptyMetricData.SetResourceMetrics([]data.ResourceMetrics{data.NewResourceMetrics()})

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
				StartTimestamp: internal.UnixnanoToTimestamp(ts1),
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
						Timestamp: internal.UnixnanoToTimestamp(ts2),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 123,
						},
					},
				},
			},
			{
				StartTimestamp: internal.UnixnanoToTimestamp(ts1),
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
						Timestamp: internal.UnixnanoToTimestamp(ts2),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 456,
						},
					},
				},
			},
			{
				StartTimestamp: internal.UnixnanoToTimestamp(ts1),
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
						Timestamp: internal.UnixnanoToTimestamp(ts2),
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
		name     string
		internal data.MetricData
		oc       []consumerdata.MetricsData
	}{
		{
			name:     "none",
			internal: data.NewMetricData(),
			oc:       []consumerdata.MetricsData{},
		},

		{
			name:     "empty",
			internal: emptyMetricData,
			oc: []consumerdata.MetricsData{
				{
					Node:     &occommon.Node{},
					Resource: &ocresource.Resource{},
					Metrics:  []*ocmetrics.Metric{},
				},
			},
		},

		{
			name:     "sample-metric",
			internal: metricData,
			oc: []consumerdata.MetricsData{
				{
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MetricDataToOC(test.internal)
			assert.EqualValues(t, test.oc, got)
		})
	}
}
