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
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestResourceMetricsToMetricsData(t *testing.T) {

	tests := []struct {
		name     string
		internal data.MetricData
		oc       []consumerdata.MetricsData
	}{
		{
			name:     "none",
			internal: data.NewMetricData(),
			oc:       []consumerdata.MetricsData(nil),
		},

		{
			name:     "no-libraries",
			internal: generateInternalTestDataNoLibraries(),
			oc: []consumerdata.MetricsData{
				generateOcTestDataNoMetrics(),
			},
		},

		{
			name:     "no-metrics",
			internal: generateInternalTestDataNoMetrics(),
			oc: []consumerdata.MetricsData{
				generateOcTestDataNoMetrics(),
			},
		},

		{
			name:     "no-points",
			internal: generateInternalTestDataNoPoints(),
			oc: []consumerdata.MetricsData{
				generateOcTestDataNoPoints(),
			},
		},

		{
			name:     "sample-metric",
			internal: generateInternalTestData(),
			oc: []consumerdata.MetricsData{
				generateOcTestData(t),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MetricDataToOC(test.internal)
			if test.name == "sample-metric" {
				edv := test.oc[0].Metrics[2].Timeseries[1].Points[0].GetDistributionValue()
				gdv := got[0].Metrics[2].Timeseries[1].Points[0].GetDistributionValue()
				assert.EqualValues(t, edv, gdv)
				assert.EqualValues(t, edv.Buckets[0].Exemplar, gdv.Buckets[0].Exemplar)
			}
			assert.EqualValues(t, test.oc, got)
		})
	}
}

const startTimeUnixNano = data.TimestampUnixNano(1585012403000000789)
const timeUnixNano = data.TimestampUnixNano(1585012403000000789)

func generateInternalTestDataNoLibraries() data.MetricData {
	metricDataNoLibraries := data.NewMetricData()
	metricDataNoLibraries.SetResourceMetrics(data.NewResourceMetricsSlice(1))
	return metricDataNoLibraries
}

func generateInternalTestDataNoMetrics() data.MetricData {
	metricDataNoMetrics := data.NewMetricData()
	metricDataNoMetrics.SetResourceMetrics(data.NewResourceMetricsSlice(1))
	metricDataNoMetrics.ResourceMetrics().Get(0).SetInstrumentationLibraryMetrics(data.NewInstrumentationLibraryMetricsSlice(1))
	return metricDataNoMetrics
}

func generateInternalTestDataNoPoints() data.MetricData {
	metricDataNoPoints := data.NewMetricData()
	metricDataNoPoints.SetResourceMetrics(data.NewResourceMetricsSlice(1))
	metricDataNoPoints.ResourceMetrics().Get(0).SetInstrumentationLibraryMetrics(data.NewInstrumentationLibraryMetricsSlice(1))
	metricDataNoPoints.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).SetMetrics(data.NewMetricSlice(1))
	fillMetricDescriptor(
		metricDataNoPoints.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0).MetricDescriptor(), "no-points", data.MetricTypeGaugeDouble)
	return metricDataNoPoints
}

func generateInternalTestData() data.MetricData {
	metricData := data.NewMetricData()
	metricData.SetResourceMetrics(data.NewResourceMetricsSlice(1))

	rms := metricData.ResourceMetrics()
	rms.Get(0).SetInstrumentationLibraryMetrics(data.NewInstrumentationLibraryMetricsSlice(1))
	rms.Get(0).Resource().SetAttributes(data.NewAttributeMap(data.AttributesMap{
		conventions.OCAttributeProcessStartTime: data.NewAttributeValueString("2020-02-11T20:26:00Z"),
		conventions.AttributeHostHostname:       data.NewAttributeValueString("host1"),
		conventions.OCAttributeProcessID:        data.NewAttributeValueString("123"),
		conventions.AttributeLibraryVersion:     data.NewAttributeValueString("v2.0.1"),
		conventions.OCAttributeExporterVersion:  data.NewAttributeValueString("v1.2.0"),
		conventions.AttributeLibraryLanguage:    data.NewAttributeValueString("CPP"),
		conventions.OCAttributeResourceType:     data.NewAttributeValueString("good-resource"),
		"str1":                                  data.NewAttributeValueString("text"),
		"int2":                                  data.NewAttributeValueInt(123),
	}))

	ilms := rms.Get(0).InstrumentationLibraryMetrics()
	ilms.Get(0).SetMetrics(data.NewMetricSlice(4))

	metrics := ilms.Get(0).Metrics()

	intMetric := metrics.Get(0)
	fillMetricDescriptor(intMetric.MetricDescriptor(), "mymetric-int", data.MetricTypeCounterInt64)
	intMetric.SetInt64DataPoints(data.NewInt64DataPointSlice(2))
	int64DataPoints := intMetric.Int64DataPoints()
	int64DataPoints.Get(0).SetLabelsMap(data.NewStringMap(map[string]string{"key1": "value1"}))
	int64DataPoints.Get(0).SetStartTime(startTimeUnixNano)
	int64DataPoints.Get(0).SetTimestamp(timeUnixNano)
	int64DataPoints.Get(0).SetValue(123)
	int64DataPoints.Get(1).SetLabelsMap(data.NewStringMap(map[string]string{"key2": "value2"}))
	int64DataPoints.Get(1).SetStartTime(startTimeUnixNano)
	int64DataPoints.Get(1).SetTimestamp(timeUnixNano)
	int64DataPoints.Get(1).SetValue(456)

	doubleMetric := metrics.Get(1)
	doubleMetric.SetDoubleDataPoints(data.NewDoubleDataPointSlice(1))
	fillMetricDescriptor(doubleMetric.MetricDescriptor(), "mymetric-double", data.MetricTypeCounterDouble)
	doubleDataPoints := doubleMetric.DoubleDataPoints()
	doubleDataPoints.Get(0).SetStartTime(startTimeUnixNano)
	doubleDataPoints.Get(0).SetTimestamp(timeUnixNano)
	doubleDataPoints.Get(0).SetValue(1.23)

	histogramMetric := metrics.Get(2)
	histogramMetric.SetHistogramDataPoints(data.NewHistogramDataPointSlice(2))
	fillMetricDescriptor(histogramMetric.MetricDescriptor(), "mymetric-histogram", data.MetricTypeCumulativeHistogram)

	histogramDataPoints := histogramMetric.HistogramDataPoints()
	histogramDataPoints.Get(0).SetLabelsMap(data.NewStringMap(map[string]string{
		"key1": "histogram-value1",
		"key3": "histogram-value3",
	}))
	histogramDataPoints.Get(0).SetStartTime(startTimeUnixNano)
	histogramDataPoints.Get(0).SetTimestamp(timeUnixNano)
	histogramDataPoints.Get(0).SetCount(1)
	histogramDataPoints.Get(0).SetSum(15)
	histogramDataPoints.Get(1).SetLabelsMap(data.NewStringMap(map[string]string{
		"key2": "histogram-value2",
	}))
	histogramDataPoints.Get(1).SetStartTime(startTimeUnixNano)
	histogramDataPoints.Get(1).SetTimestamp(timeUnixNano)
	histogramDataPoints.Get(1).SetCount(1)
	histogramDataPoints.Get(1).SetSum(15)
	histogramDataPoints.Get(1).SetBuckets(data.NewHistogramBucketSlice(2))
	histogramDataPoints.Get(1).Buckets().Get(0).SetCount(0)
	histogramDataPoints.Get(1).Buckets().Get(1).SetCount(1)
	histogramDataPoints.Get(1).Buckets().Get(1).Exemplar().SetTimestamp(startTimeUnixNano)
	histogramDataPoints.Get(1).Buckets().Get(1).Exemplar().SetValue(15)
	histogramDataPoints.Get(1).Buckets().Get(1).Exemplar().SetAttachments(data.NewStringMap(map[string]string{
		"key": "value",
	}))
	histogramDataPoints.Get(1).SetExplicitBounds([]float64{1})

	summaryMetric := metrics.Get(3)
	summaryMetric.SetSummaryDataPoints(data.NewSummaryDataPointSlice(2))
	fillMetricDescriptor(summaryMetric.MetricDescriptor(), "mymetric-summary", data.MetricTypeSummary)

	summaryDataPoints := summaryMetric.SummaryDataPoints()
	summaryDataPoints.Get(0).SetLabelsMap(data.NewStringMap(map[string]string{
		"key1": "summary-value1",
	}))
	summaryDataPoints.Get(0).SetStartTime(startTimeUnixNano)
	summaryDataPoints.Get(0).SetTimestamp(timeUnixNano)
	summaryDataPoints.Get(0).SetCount(1)
	summaryDataPoints.Get(0).SetSum(15)
	summaryDataPoints.Get(1).SetLabelsMap(data.NewStringMap(map[string]string{
		"key1": "summary-value2",
	}))
	summaryDataPoints.Get(1).SetStartTime(startTimeUnixNano)
	summaryDataPoints.Get(1).SetTimestamp(timeUnixNano)
	summaryDataPoints.Get(1).SetCount(1)
	summaryDataPoints.Get(1).SetSum(15)
	summaryDataPoints.Get(1).SetValueAtPercentiles(data.NewSummaryValueAtPercentileSlice(1))
	summaryDataPoints.Get(1).ValueAtPercentiles().Get(0).SetPercentile(1)
	summaryDataPoints.Get(1).ValueAtPercentiles().Get(0).SetValue(15)

	return metricData
}

func generateOcTestDataNoMetrics() consumerdata.MetricsData {
	return consumerdata.MetricsData{
		Node:     &occommon.Node{},
		Resource: &ocresource.Resource{},
		Metrics:  []*ocmetrics.Metric(nil),
	}
}

func generateOcTestDataNoPoints() consumerdata.MetricsData {
	return consumerdata.MetricsData{
		Node:     &occommon.Node{},
		Resource: &ocresource.Resource{},
		Metrics: []*ocmetrics.Metric{
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "no-points",
					Description: "My metric",
					Unit:        "ms",
					Type:        ocmetrics.MetricDescriptor_GAUGE_DOUBLE,
				},
			},
		},
	}
}

func generateOcTestData(t *testing.T) consumerdata.MetricsData {
	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	ocMetricInt := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric-int",
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
				StartTimestamp: internal.UnixNanoToTimestamp(startTimeUnixNano),
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
						Timestamp: internal.UnixNanoToTimestamp(timeUnixNano),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 123,
						},
					},
				},
			},
			{
				StartTimestamp: internal.UnixNanoToTimestamp(startTimeUnixNano),
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
						Timestamp: internal.UnixNanoToTimestamp(timeUnixNano),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 456,
						},
					},
				},
			},
		},
	}
	ocMetricDouble := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric-double",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE,
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.UnixNanoToTimestamp(startTimeUnixNano),
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(timeUnixNano),
						Value: &ocmetrics.Point_DoubleValue{
							DoubleValue: 1.23,
						},
					},
				},
			},
		},
	}
	ocMetricHistogram := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric-histogram",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "key1"},
				{Key: "key2"},
				{Key: "key3"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.UnixNanoToTimestamp(startTimeUnixNano),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "histogram-value1",
						HasValue: true,
					},
					{
						// key2
						HasValue: false,
					},
					{
						// key3
						Value:    "histogram-value3",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(timeUnixNano),
						Value: &ocmetrics.Point_DistributionValue{
							DistributionValue: &ocmetrics.DistributionValue{
								Count: 1,
								Sum:   15,
							},
						},
					},
				},
			},
			{
				StartTimestamp: internal.UnixNanoToTimestamp(startTimeUnixNano),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						HasValue: false,
					},
					{
						// key2
						Value:    "histogram-value2",
						HasValue: true,
					},
					{
						// key3
						HasValue: false,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(timeUnixNano),
						Value: &ocmetrics.Point_DistributionValue{
							DistributionValue: &ocmetrics.DistributionValue{
								Count: 1,
								Sum:   15,
								BucketOptions: &ocmetrics.DistributionValue_BucketOptions{
									Type: &ocmetrics.DistributionValue_BucketOptions_Explicit_{
										Explicit: &ocmetrics.DistributionValue_BucketOptions_Explicit{
											Bounds: []float64{1},
										},
									},
								},
								Buckets: []*ocmetrics.DistributionValue_Bucket{
									{
										Count: 0,
										// TODO: Fix this when we have a way to know if the value is default.
										// https://github.com/open-telemetry/opentelemetry-collector/pull/691
										Exemplar: &ocmetrics.DistributionValue_Exemplar{
											Timestamp: &timestamp.Timestamp{},
										},
									},
									{
										Count: 1,
										Exemplar: &ocmetrics.DistributionValue_Exemplar{
											Timestamp:   internal.UnixNanoToTimestamp(timeUnixNano),
											Value:       15,
											Attachments: map[string]string{"key": "value"},
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
	ocMetricSummary := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        "mymetric-summary",
			Description: "My metric",
			Unit:        "ms",
			Type:        ocmetrics.MetricDescriptor_SUMMARY,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "key1"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.UnixNanoToTimestamp(startTimeUnixNano),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "summary-value1",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(timeUnixNano),
						Value: &ocmetrics.Point_SummaryValue{
							SummaryValue: &ocmetrics.SummaryValue{
								Count: &wrappers.Int64Value{
									Value: 1,
								},
								Sum: &wrappers.DoubleValue{
									Value: 15,
								},
							},
						},
					},
				},
			},
			{
				StartTimestamp: internal.UnixNanoToTimestamp(startTimeUnixNano),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "summary-value2",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.UnixNanoToTimestamp(timeUnixNano),
						Value: &ocmetrics.Point_SummaryValue{
							SummaryValue: &ocmetrics.SummaryValue{
								Count: &wrappers.Int64Value{
									Value: 1,
								},
								Sum: &wrappers.DoubleValue{
									Value: 15,
								},
								Snapshot: &ocmetrics.SummaryValue_Snapshot{
									PercentileValues: []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile{
										{
											Percentile: 1,
											Value:      15,
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

	return consumerdata.MetricsData{
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
			Type: "good-resource",
			Labels: map[string]string{
				"str1": "text",
				"int2": "123",
			},
		},
		Metrics: []*ocmetrics.Metric{ocMetricInt, ocMetricDouble, ocMetricHistogram, ocMetricSummary},
	}
}

func fillMetricDescriptor(md data.MetricDescriptor, name string, ty data.MetricType) {
	md.SetName(name)
	md.SetDescription("My metric")
	md.SetUnit("ms")
	md.SetType(ty)
}
