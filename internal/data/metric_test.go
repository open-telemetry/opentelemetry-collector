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

package data

import (
	"testing"

	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/assert"
)

const (
	startTime = uint64(12578940000000012345)
	endTime   = uint64(12578940000000054321)
)

func TestMetricCount(t *testing.T) {
	md := NewMetricData()
	assert.EqualValues(t, 0, md.MetricCount())

	md.SetResourceMetrics(NewResourceMetricsSlice(1))
	assert.EqualValues(t, 0, md.MetricCount())

	md.ResourceMetrics().Get(0).SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	assert.EqualValues(t, 0, md.MetricCount())

	md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).SetMetrics(NewMetricSlice(1))
	assert.EqualValues(t, 1, md.MetricCount())

	rms := NewResourceMetricsSlice(3)
	rms.Get(0).SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	rms.Get(0).InstrumentationLibraryMetrics().Get(0).SetMetrics(NewMetricSlice(1))
	rms.Get(1).SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	rms.Get(2).SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	rms.Get(2).InstrumentationLibraryMetrics().Get(0).SetMetrics(NewMetricSlice(5))
	md.SetResourceMetrics(rms)
	assert.EqualValues(t, 6, md.MetricCount())
}

func TestOtlpToInternalReadOnly(t *testing.T) {
	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntMetric(), generateTestProtoDoubleMetric(), generateTestProtoHistogramMetric(), generateTestProtoSummaryMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	assert.EqualValues(t, 1, resourceMetrics.Len())
	resourceMetric := resourceMetrics.Get(0)
	assert.EqualValues(t, generateTestProtoResource(), *resourceMetric.Resource().orig)
	metrics := resourceMetric.InstrumentationLibraryMetrics().Get(0).Metrics()
	assert.EqualValues(t, 4, metrics.Len())
	// Check int64 metric
	metricInt := metrics.Get(0)
	assert.EqualValues(t, "my_metric_int", metricInt.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricInt.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricInt.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeCounterInt64, metricInt.MetricDescriptor().Type())
	assert.EqualValues(t, NewStringMap(map[string]string{}), metricInt.MetricDescriptor().LabelsMap())
	int64DataPoints := metricInt.Int64DataPoints()
	assert.EqualValues(t, 2, int64DataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, int64DataPoints.Get(0).StartTime())
	assert.EqualValues(t, endTime, int64DataPoints.Get(0).Timestamp())
	assert.EqualValues(t, 123, int64DataPoints.Get(0).Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), int64DataPoints.Get(0).LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, int64DataPoints.Get(1).StartTime())
	assert.EqualValues(t, endTime, int64DataPoints.Get(1).Timestamp())
	assert.EqualValues(t, 456, int64DataPoints.Get(1).Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), int64DataPoints.Get(1).LabelsMap())
	// Check double metric
	metricDouble := metrics.Get(1)
	assert.EqualValues(t, "my_metric_double", metricDouble.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricDouble.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricDouble.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeCounterDouble, metricDouble.MetricDescriptor().Type())
	doubleDataPoints := metricDouble.DoubleDataPoints()
	assert.EqualValues(t, 2, doubleDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, doubleDataPoints.Get(0).StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints.Get(0).Timestamp())
	assert.EqualValues(t, 123.1, doubleDataPoints.Get(0).Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), doubleDataPoints.Get(0).LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, doubleDataPoints.Get(1).StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints.Get(1).Timestamp())
	assert.EqualValues(t, 456.1, doubleDataPoints.Get(1).Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), doubleDataPoints.Get(1).LabelsMap())
	// Check histogram metric
	metricHistogram := metrics.Get(2)
	assert.EqualValues(t, "my_metric_histogram", metricHistogram.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricHistogram.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricHistogram.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeCumulativeHistogram, metricHistogram.MetricDescriptor().Type())
	histogramDataPoints := metricHistogram.HistogramDataPoints()
	assert.EqualValues(t, 2, histogramDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, histogramDataPoints.Get(0).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.Get(0).Timestamp())
	assert.EqualValues(t, []float64{1, 2}, histogramDataPoints.Get(0).ExplicitBounds())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), histogramDataPoints.Get(0).LabelsMap())
	assert.EqualValues(t, 3, histogramDataPoints.Get(0).Buckets().Len())
	assert.EqualValues(t, 10, histogramDataPoints.Get(0).Buckets().Get(0).Count())
	assert.EqualValues(t, 15, histogramDataPoints.Get(0).Buckets().Get(1).Count())
	assert.EqualValues(t, 1.5, histogramDataPoints.Get(0).Buckets().Get(1).Exemplar().Value())
	assert.EqualValues(t, startTime, histogramDataPoints.Get(0).Buckets().Get(1).Exemplar().Timestamp())
	assert.EqualValues(t, NewStringMap(map[string]string{"key_a1": "value_a1"}), histogramDataPoints.Get(0).Buckets().Get(1).Exemplar().Attachments())
	assert.EqualValues(t, 1, histogramDataPoints.Get(0).Buckets().Get(2).Count())
	// Second point
	assert.EqualValues(t, startTime, histogramDataPoints.Get(1).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.Get(1).Timestamp())
	assert.EqualValues(t, []float64{1}, histogramDataPoints.Get(1).ExplicitBounds())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), histogramDataPoints.Get(1).LabelsMap())
	assert.EqualValues(t, 2, histogramDataPoints.Get(1).Buckets().Len())
	assert.EqualValues(t, 10, histogramDataPoints.Get(1).Buckets().Get(0).Count())
	assert.EqualValues(t, 1, histogramDataPoints.Get(1).Buckets().Get(1).Count())
	// Check summary metric
	metricSummary := metrics.Get(3)
	assert.EqualValues(t, "my_metric_summary", metricSummary.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricSummary.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricSummary.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeSummary, metricSummary.MetricDescriptor().Type())
	summaryDataPoints := metricSummary.SummaryDataPoints()
	assert.EqualValues(t, 2, summaryDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, summaryDataPoints.Get(0).StartTime())
	assert.EqualValues(t, endTime, summaryDataPoints.Get(0).Timestamp())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), summaryDataPoints.Get(0).LabelsMap())
	assert.EqualValues(t, 2, summaryDataPoints.Get(0).ValueAtPercentiles().Len())
	assert.EqualValues(t, 0.0, summaryDataPoints.Get(0).ValueAtPercentiles().Get(0).Percentile())
	assert.EqualValues(t, 1.23, summaryDataPoints.Get(0).ValueAtPercentiles().Get(0).Value())
	assert.EqualValues(t, 1.0, summaryDataPoints.Get(0).ValueAtPercentiles().Get(1).Percentile())
	assert.EqualValues(t, 4.56, summaryDataPoints.Get(0).ValueAtPercentiles().Get(1).Value())
	// Second point
	assert.EqualValues(t, startTime, summaryDataPoints.Get(1).StartTime())
	assert.EqualValues(t, endTime, summaryDataPoints.Get(1).Timestamp())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), summaryDataPoints.Get(1).LabelsMap())
	assert.EqualValues(t, 2, summaryDataPoints.Get(1).ValueAtPercentiles().Len())
	assert.EqualValues(t, 0.5, summaryDataPoints.Get(1).ValueAtPercentiles().Get(0).Percentile())
	assert.EqualValues(t, 4.56, summaryDataPoints.Get(1).ValueAtPercentiles().Get(0).Value())
	assert.EqualValues(t, 0.9, summaryDataPoints.Get(1).ValueAtPercentiles().Get(1).Percentile())
	assert.EqualValues(t, 7.89, summaryDataPoints.Get(1).ValueAtPercentiles().Get(1).Value())
}
func TestOtlpToFromInternalReadOnly(t *testing.T) {
	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntMetric(), generateTestProtoDoubleMetric(), generateTestProtoHistogramMetric(), generateTestProtoSummaryMetric()},
				},
			},
		},
	})
	// Test that nothing changed
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntMetric(), generateTestProtoDoubleMetric(), generateTestProtoHistogramMetric(), generateTestProtoSummaryMetric()},
				},
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalIntPointsMutating(t *testing.T) {
	newLabels := NewStringMap(map[string]string{"k": "v"})

	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics.Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_int")
	assert.EqualValues(t, "new_my_metric_int", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetType(MetricTypeGaugeInt64)
	assert.EqualValues(t, MetricTypeGaugeInt64, metric.MetricDescriptor().Type())
	metric.MetricDescriptor().SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, metric.MetricDescriptor().LabelsMap())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.Int64DataPoints().Len())
	metric.Int64DataPoints().Resize(0, 1)
	assert.EqualValues(t, 1, metric.Int64DataPoints().Len())
	int64DataPoints := metric.Int64DataPoints()
	int64DataPoints.Get(0).SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, int64DataPoints.Get(0).StartTime())
	int64DataPoints.Get(0).SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, int64DataPoints.Get(0).Timestamp())
	int64DataPoints.Get(0).SetValue(124)
	assert.EqualValues(t, 124, int64DataPoints.Get(0).Value())
	int64DataPoints.Get(0).SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, int64DataPoints.Get(0).LabelsMap())

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics: []*otlpmetrics.Metric{
						{
							MetricDescriptor: &otlpmetrics.MetricDescriptor{
								Name:        "new_my_metric_int",
								Description: "My new metric",
								Unit:        "1",
								Type:        otlpmetrics.MetricDescriptor_GAUGE_INT64,
								Labels: []*otlpcommon.StringKeyValue{
									{
										Key:   "k",
										Value: "v",
									},
								},
							},
							Int64DataPoints: []*otlpmetrics.Int64DataPoint{
								{
									Labels: []*otlpcommon.StringKeyValue{
										{
											Key:   "k",
											Value: "v",
										},
									},
									StartTimeUnixNano: startTime + 1,
									TimeUnixNano:      endTime + 1,
									Value:             124,
								},
							},
						},
					},
				},
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalDoublePointsMutating(t *testing.T) {
	newLabels := NewStringMap(map[string]string{"k": "v"})

	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoDoubleMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics.Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_double")
	assert.EqualValues(t, "new_my_metric_double", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetType(MetricTypeGaugeDouble)
	assert.EqualValues(t, MetricTypeGaugeDouble, metric.MetricDescriptor().Type())
	metric.MetricDescriptor().SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, metric.MetricDescriptor().LabelsMap())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.DoubleDataPoints().Len())
	metric.DoubleDataPoints().Resize(0, 1)
	assert.EqualValues(t, 1, metric.DoubleDataPoints().Len())
	doubleDataPoints := metric.DoubleDataPoints()
	doubleDataPoints.Get(0).SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, doubleDataPoints.Get(0).StartTime())
	doubleDataPoints.Get(0).SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, doubleDataPoints.Get(0).Timestamp())
	doubleDataPoints.Get(0).SetValue(124.1)
	assert.EqualValues(t, 124.1, doubleDataPoints.Get(0).Value())
	doubleDataPoints.Get(0).SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, doubleDataPoints.Get(0).LabelsMap())

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics: []*otlpmetrics.Metric{
						{
							MetricDescriptor: &otlpmetrics.MetricDescriptor{
								Name:        "new_my_metric_double",
								Description: "My new metric",
								Unit:        "1",
								Type:        otlpmetrics.MetricDescriptor_GAUGE_DOUBLE,
								Labels: []*otlpcommon.StringKeyValue{
									{
										Key:   "k",
										Value: "v",
									},
								},
							},
							DoubleDataPoints: []*otlpmetrics.DoubleDataPoint{
								{
									Labels: []*otlpcommon.StringKeyValue{
										{
											Key:   "k",
											Value: "v",
										},
									},
									StartTimeUnixNano: startTime + 1,
									TimeUnixNano:      endTime + 1,
									Value:             124.1,
								},
							},
						},
					},
				},
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalHistogramPointsMutating(t *testing.T) {
	newLabels := NewStringMap(map[string]string{"k": "v"})

	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoHistogramMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics.Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_histogram")
	assert.EqualValues(t, "new_my_metric_histogram", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetType(MetricTypeGaugeHistogram)
	assert.EqualValues(t, MetricTypeGaugeHistogram, metric.MetricDescriptor().Type())
	metric.MetricDescriptor().SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, metric.MetricDescriptor().LabelsMap())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.HistogramDataPoints().Len())
	metric.HistogramDataPoints().Resize(0, 1)
	assert.EqualValues(t, 1, metric.HistogramDataPoints().Len())
	histogramDataPoints := metric.HistogramDataPoints()
	histogramDataPoints.Get(0).SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.Get(0).StartTime())
	histogramDataPoints.Get(0).SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints.Get(0).Timestamp())
	histogramDataPoints.Get(0).SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, histogramDataPoints.Get(0).LabelsMap())
	histogramDataPoints.Get(0).SetExplicitBounds([]float64{1})
	assert.EqualValues(t, []float64{1}, histogramDataPoints.Get(0).ExplicitBounds())
	assert.EqualValues(t, 3, histogramDataPoints.Get(0).Buckets().Len())
	histogramDataPoints.Get(0).Buckets().Resize(0, 2)
	assert.EqualValues(t, 2, histogramDataPoints.Get(0).Buckets().Len())
	buckets := histogramDataPoints.Get(0).Buckets()
	buckets.Get(0).SetCount(21)
	assert.EqualValues(t, 21, buckets.Get(0).Count())
	buckets.Get(1).SetCount(32)
	assert.EqualValues(t, 32, buckets.Get(1).Count())
	buckets.Get(1).Exemplar().SetTimestamp(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, buckets.Get(1).Exemplar().Timestamp())
	buckets.Get(1).Exemplar().SetValue(10.5)
	assert.EqualValues(t, 10.5, buckets.Get(1).Exemplar().Value())
	buckets.Get(1).Exemplar().SetAttachments(newLabels)
	assert.EqualValues(t, newLabels, buckets.Get(1).Exemplar().Attachments())

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics: []*otlpmetrics.Metric{
						{
							MetricDescriptor: &otlpmetrics.MetricDescriptor{
								Name:        "new_my_metric_histogram",
								Description: "My new metric",
								Unit:        "1",
								Type:        otlpmetrics.MetricDescriptor_GAUGE_HISTOGRAM,
								Labels: []*otlpcommon.StringKeyValue{
									{
										Key:   "k",
										Value: "v",
									},
								},
							},
							HistogramDataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Labels: []*otlpcommon.StringKeyValue{
										{
											Key:   "k",
											Value: "v",
										},
									},
									StartTimeUnixNano: startTime + 1,
									TimeUnixNano:      endTime + 1,
									Buckets: []*otlpmetrics.HistogramDataPoint_Bucket{
										{
											Count: 21,
										},
										{
											Count: 32,
											Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
												Value:        10.5,
												TimeUnixNano: startTime + 1,
												Attachments: []*otlpcommon.StringKeyValue{
													{
														Key:   "k",
														Value: "v",
													},
												},
											},
										},
									},
									ExplicitBounds: []float64{1},
								},
							},
						},
					},
				},
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalSummaryPointsMutating(t *testing.T) {
	newLabels := NewStringMap(map[string]string{"k": "v"})
	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoSummaryMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics.Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_summary")
	assert.EqualValues(t, "new_my_metric_summary", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, metric.MetricDescriptor().LabelsMap())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.SummaryDataPoints().Len())
	metric.SummaryDataPoints().Resize(0, 1)
	assert.EqualValues(t, 1, metric.SummaryDataPoints().Len())
	summaryDataPoints := metric.SummaryDataPoints()
	summaryDataPoints.Get(0).SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, summaryDataPoints.Get(0).StartTime())
	summaryDataPoints.Get(0).SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, summaryDataPoints.Get(0).Timestamp())
	summaryDataPoints.Get(0).SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, summaryDataPoints.Get(0).LabelsMap())
	// Mutate ValueAtPercentiles
	assert.EqualValues(t, 2, summaryDataPoints.Get(0).ValueAtPercentiles().Len())
	summaryDataPoints.Get(0).ValueAtPercentiles().Remove(1)
	assert.EqualValues(t, 1, summaryDataPoints.Get(0).ValueAtPercentiles().Len())
	valueAtPercentile := summaryDataPoints.Get(0).ValueAtPercentiles().Get(0)
	valueAtPercentile.SetValue(1.24)
	assert.EqualValues(t, 1.24, valueAtPercentile.Value())
	valueAtPercentile.SetPercentile(0.1)
	assert.EqualValues(t, 0.1, valueAtPercentile.Percentile())

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics: []*otlpmetrics.Metric{
						{
							MetricDescriptor: &otlpmetrics.MetricDescriptor{
								Name:        "new_my_metric_summary",
								Description: "My new metric",
								Unit:        "1",
								Type:        otlpmetrics.MetricDescriptor_SUMMARY,
								Labels: []*otlpcommon.StringKeyValue{
									{
										Key:   "k",
										Value: "v",
									},
								},
							},
							SummaryDataPoints: []*otlpmetrics.SummaryDataPoint{
								{
									Labels: []*otlpcommon.StringKeyValue{
										{
											Key:   "k",
											Value: "v",
										},
									},
									StartTimeUnixNano: startTime + 1,
									TimeUnixNano:      endTime + 1,
									PercentileValues: []*otlpmetrics.SummaryDataPoint_ValueAtPercentile{
										{
											Percentile: 0.1,
											Value:      1.24,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, MetricDataToOtlp(metricData))
}

func BenchmarkOtlpToFromInternal_PassThrough(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntMetric(), generateTestProtoDoubleMetric(), generateTestProtoHistogramMetric(), generateTestProtoSummaryMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_Int64Points_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0).Int64DataPoints().Get(0).LabelsMap().Upsert("key0", "value2")
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_DoublePoints_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoDoubleMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0).DoubleDataPoints().Get(0).LabelsMap().Upsert("key0", "value2")
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_HistogramPoints_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoHistogramMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0).HistogramDataPoints().Get(0).LabelsMap().Upsert("key0", "value2")
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_SummaryPoints_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoSummaryMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0).SummaryDataPoints().Get(0).LabelsMap().Upsert("key0", "value2")
		MetricDataToOtlp(md)
	}
}

func generateTestProtoResource() *otlpresource.Resource {
	return &otlpresource.Resource{
		Attributes: []*otlpcommon.AttributeKeyValue{
			{
				Key:         "string",
				Type:        otlpcommon.AttributeKeyValue_STRING,
				StringValue: "string-resource",
			},
		},
	}
}

func generateTestProtoInstrumentationLibrary() *otlpcommon.InstrumentationLibrary {
	return &otlpcommon.InstrumentationLibrary{
		Name:    "test",
		Version: "",
	}
}

func generateTestProtoIntMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "my_metric_int",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_COUNTER_INT64,
		},
		Int64DataPoints: []*otlpmetrics.Int64DataPoint{
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key0",
						Value: "value0",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				Value:             123,
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				Value:             456,
			},
		},
	}
}
func generateTestProtoDoubleMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "my_metric_double",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_COUNTER_DOUBLE,
		},
		DoubleDataPoints: []*otlpmetrics.DoubleDataPoint{
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key0",
						Value: "value0",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				Value:             123.1,
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				Value:             456.1,
			},
		},
	}
}

func generateTestProtoHistogramMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "my_metric_histogram",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_CUMULATIVE_HISTOGRAM,
		},
		HistogramDataPoints: []*otlpmetrics.HistogramDataPoint{
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key0",
						Value: "value0",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				Buckets: []*otlpmetrics.HistogramDataPoint_Bucket{
					{
						Count: 10,
					},
					{
						Count: 15,
						Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
							Value:        1.5,
							TimeUnixNano: startTime,
							Attachments: []*otlpcommon.StringKeyValue{
								{
									Key:   "key_a1",
									Value: "value_a1",
								},
							},
						},
					},
					{
						Count: 1,
					},
				},
				ExplicitBounds: []float64{1, 2},
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				Buckets: []*otlpmetrics.HistogramDataPoint_Bucket{
					{
						Count: 10,
					},
					{
						Count: 1,
					},
				},
				ExplicitBounds: []float64{1},
			},
		},
	}
}

func generateTestProtoSummaryMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: &otlpmetrics.MetricDescriptor{
			Name:        "my_metric_summary",
			Description: "My metric",
			Unit:        "ms",
			Type:        otlpmetrics.MetricDescriptor_SUMMARY,
		},
		SummaryDataPoints: []*otlpmetrics.SummaryDataPoint{
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key0",
						Value: "value0",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				PercentileValues: []*otlpmetrics.SummaryDataPoint_ValueAtPercentile{
					{
						Percentile: 0.0,
						Value:      1.23,
					},
					{
						Percentile: 1.0,
						Value:      4.56,
					},
				},
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				StartTimeUnixNano: startTime,
				TimeUnixNano:      endTime,
				PercentileValues: []*otlpmetrics.SummaryDataPoint_ValueAtPercentile{
					{
						Percentile: 0.5,
						Value:      4.56,
					},
					{
						Percentile: 0.9,
						Value:      7.89,
					},
				},
			},
		},
	}
}
