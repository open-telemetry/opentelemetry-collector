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

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestOCToOTLP(t *testing.T) {
	unixnanos1 := data.TimestampUnixNano(uint64(12578940000000012345))
	unixnanos2 := data.TimestampUnixNano(uint64(12578940000000054321))

	int64OCMetric := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "key1"},
				{Key: "key2"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.UnixNanoToTimestamp(unixnanos1),
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
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 123,
						},
					},
				},
			},
			{
				StartTimestamp: internal.UnixNanoToTimestamp(unixnanos1),
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
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 456,
						},
					},
				},
			},
		},
	}

	noLabelsOCMetric := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.UnixNanoToTimestamp(unixnanos1),
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 123,
						},
					},
				},
			},
		},
	}

	doubleOCMetric := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_GAUGE_DOUBLE,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "key1"},
				{Key: "key2"},
				{Key: "key3"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.UnixNanoToTimestamp(unixnanos1),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "value1",
						HasValue: true,
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
						Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_DoubleValue{
							DoubleValue: 1.23,
						},
					},
				},
			},
			{
				StartTimestamp: internal.UnixNanoToTimestamp(unixnanos1),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "another-value1",
						HasValue: true,
					},
					{
						// key2
						HasValue: false,
					},
					{
						// key3
						Value:    "value3",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_DoubleValue{
							DoubleValue: 3.45,
						},
					},
				},
			},
		},
	}

	histogramOCMetric := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "key1"},
				{Key: "key2"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.UnixNanoToTimestamp(unixnanos1),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "value1",
						HasValue: true,
					},
					{
						// key2
						Value:    "value2",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
						Value: &ocmetrics.Point_DistributionValue{
							DistributionValue: &ocmetrics.DistributionValue{
								Count: 48,
								Sum:   123.45,
								BucketOptions: &ocmetrics.DistributionValue_BucketOptions{
									Type: &ocmetrics.DistributionValue_BucketOptions_Explicit_{
										Explicit: &ocmetrics.DistributionValue_BucketOptions_Explicit{
											Bounds: []float64{1.2, 4.5},
										},
									},
								},
								Buckets: []*ocmetrics.DistributionValue_Bucket{
									{
										Count: 12,
										Exemplar: &ocmetrics.DistributionValue_Exemplar{
											Value:     1.1,
											Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
											Attachments: map[string]string{
												"key1": "value1",
											},
										},
									},
									{
										Count: 24,
										Exemplar: &ocmetrics.DistributionValue_Exemplar{
											Value:     2.2,
											Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
											Attachments: map[string]string{
												"key2": "value2",
											},
										},
									},
									{
										Count: 12,
										Exemplar: &ocmetrics.DistributionValue_Exemplar{
											Value:     7.1,
											Timestamp: internal.UnixNanoToTimestamp(unixnanos2),
											Attachments: map[string]string{
												"key3": "value3",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	int64Metric := &otlpmetrics.Metric{
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
				StartTimeUnixNano: uint64(unixnanos1),
				TimeUnixNano:      uint64(unixnanos2),
				Value:             123,
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key2",
						Value: "value2",
					},
				},
				StartTimeUnixNano: uint64(unixnanos1),
				TimeUnixNano:      uint64(unixnanos2),
				Value:             456,
			},
		},
	}

	noLabelsMetric := &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_COUNTER_INT64,
		},
		Int64DataPoints: []*otlpmetrics.Int64DataPoint{
			{
				StartTimeUnixNano: uint64(unixnanos1),
				TimeUnixNano:      uint64(unixnanos2),
				Value:             123,
			},
		},
	}

	doubleMetric := &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_GAUGE_DOUBLE,
		},
		DoubleDataPoints: []*otlpmetrics.DoubleDataPoint{
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
				StartTimeUnixNano: uint64(unixnanos1),
				TimeUnixNano:      uint64(unixnanos2),
				Value:             1.23,
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "another-value1",
					},
					{
						Key:   "key3",
						Value: "value3",
					},
				},
				StartTimeUnixNano: uint64(unixnanos1),
				TimeUnixNano:      uint64(unixnanos2),
				Value:             3.45,
			},
		},
	}

	histogramMetric := &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "mymetric",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_GAUGE_HISTOGRAM,
		},
		HistogramDataPoints: []*otlpmetrics.HistogramDataPoint{
			{
				StartTimeUnixNano: uint64(unixnanos1),
				TimeUnixNano:      uint64(unixnanos2),
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
				Count:          48,
				Sum:            123.45,
				ExplicitBounds: []float64{1.2, 4.5},
				Buckets: []*otlpmetrics.HistogramDataPoint_Bucket{
					{
						Count: 12,
						Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
							Value:        1.1,
							TimeUnixNano: uint64(unixnanos2),
							Attachments: []*otlpcommon.StringKeyValue{
								{
									Key:   "key1",
									Value: "value1",
								},
							},
						},
					},
					{
						Count: 24,
						Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
							Value:        2.2,
							TimeUnixNano: uint64(unixnanos2),
							Attachments: []*otlpcommon.StringKeyValue{
								{
									Key:   "key2",
									Value: "value2",
								},
							},
						},
					},
					{
						Count: 12,
						Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
							Value:        7.1,
							TimeUnixNano: uint64(unixnanos2),
							Attachments: []*otlpcommon.StringKeyValue{
								{
									Key:   "key3",
									Value: "value3",
								},
							},
						},
					},
				},
			},
		},
	}

	otlpAttributes := []*otlpcommon.AttributeKeyValue{
		{
			Key:         conventions.OCAttributeResourceType,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "good-resource",
		},
	}

	tests := []struct {
		name string
		oc   consumerdata.MetricsData
		otlp []*otlpmetrics.ResourceMetrics
	}{
		{
			name: "empty",
			oc:   consumerdata.MetricsData{},
			otlp: nil,
		},

		{
			name: "empty-metrics",
			oc: consumerdata.MetricsData{
				Node:     &occommon.Node{},
				Resource: &ocresource.Resource{},
			},
			otlp: []*otlpmetrics.ResourceMetrics{
				{
					Resource: &otlpresource.Resource{},
				},
			},
		},

		{
			name: "no-metrics",
			oc: consumerdata.MetricsData{
				Resource: &ocresource.Resource{
					Type: "good-resource",
				},
				Metrics: []*ocmetrics.Metric{},
			},
			otlp: []*otlpmetrics.ResourceMetrics{
				{
					Resource: &otlpresource.Resource{
						Attributes: otlpAttributes,
					},
				},
			},
		},

		{
			name: "int64-metric",
			oc: consumerdata.MetricsData{
				Resource: &ocresource.Resource{
					Type: "good-resource",
				},
				Metrics: []*ocmetrics.Metric{int64OCMetric},
			},
			otlp: []*otlpmetrics.ResourceMetrics{
				{
					Resource: &otlpresource.Resource{
						Attributes: otlpAttributes,
					},
					InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
						{
							Metrics: []*otlpmetrics.Metric{int64Metric},
						},
					},
				},
			},
		},

		{
			name: "no-labels-metric",
			oc: consumerdata.MetricsData{
				Resource: &ocresource.Resource{
					Type: "good-resource",
				},
				Metrics: []*ocmetrics.Metric{noLabelsOCMetric},
			},
			otlp: []*otlpmetrics.ResourceMetrics{
				{
					Resource: &otlpresource.Resource{
						Attributes: otlpAttributes,
					},
					InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
						{
							Metrics: []*otlpmetrics.Metric{noLabelsMetric},
						},
					},
				},
			},
		},

		{
			name: "double-metric",
			oc: consumerdata.MetricsData{
				Resource: &ocresource.Resource{
					Type: "good-resource",
				},
				Metrics: []*ocmetrics.Metric{doubleOCMetric},
			},
			otlp: []*otlpmetrics.ResourceMetrics{
				{
					Resource: &otlpresource.Resource{
						Attributes: otlpAttributes,
					},
					InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
						{
							Metrics: []*otlpmetrics.Metric{doubleMetric},
						},
					},
				},
			},
		},

		{
			name: "histogram-metric",
			oc: consumerdata.MetricsData{
				Resource: &ocresource.Resource{
					Type: "good-resource",
				},
				Metrics: []*ocmetrics.Metric{histogramOCMetric},
			},
			otlp: []*otlpmetrics.ResourceMetrics{
				{
					Resource: &otlpresource.Resource{
						Attributes: otlpAttributes,
					},
					InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
						{
							Metrics: []*otlpmetrics.Metric{histogramMetric},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := OCToOTLP(test.oc)
			assert.EqualValues(t, test.otlp, got)
		})
	}
}
