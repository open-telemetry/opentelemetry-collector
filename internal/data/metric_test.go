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
	"math/rand"
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

	md.ResourceMetrics()[0].SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	assert.EqualValues(t, 0, md.MetricCount())

	rm := NewResourceMetrics()
	rm.SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	rm.InstrumentationLibraryMetrics()[0].SetMetrics(NewMetricSlice(1))
	md.SetResourceMetrics([]ResourceMetrics{rm})
	assert.EqualValues(t, 1, md.MetricCount())

	rm1 := NewResourceMetrics()
	rm1.SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	rm1.InstrumentationLibraryMetrics()[0].SetMetrics(NewMetricSlice(1))
	rm2 := NewResourceMetrics()
	rm2.SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	rm3 := NewResourceMetrics()
	rm3.SetInstrumentationLibraryMetrics(NewInstrumentationLibraryMetricsSlice(1))
	rm3.InstrumentationLibraryMetrics()[0].SetMetrics(NewMetricSlice(5))
	md.SetResourceMetrics([]ResourceMetrics{rm1, rm2, rm3})
	assert.EqualValues(t, 6, md.MetricCount())
}

func TestNewMetricSlice(t *testing.T) {
	ms := NewMetricSlice(0)
	assert.EqualValues(t, 0, len(ms))

	n := rand.Intn(10)
	ms = NewMetricSlice(n)
	assert.EqualValues(t, n, len(ms))
	for Metric := range ms {
		assert.NotNil(t, Metric)
	}
}

func TestNewInt64DataPointSlice(t *testing.T) {
	dps := NewInt64DataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewInt64DataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for event := range dps {
		assert.NotNil(t, event)
	}
}

func TestNewDoubleDataPointSlice(t *testing.T) {
	dps := NewDoubleDataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewDoubleDataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for event := range dps {
		assert.NotNil(t, event)
	}
}

func TestNewHistogramDataPointSlice(t *testing.T) {
	dps := NewHistogramDataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewHistogramDataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for event := range dps {
		assert.NotNil(t, event)
	}
}

func TestNewHistogramBucketSlice(t *testing.T) {
	hbs := NewHistogramBucketSlice(0)
	assert.EqualValues(t, 0, hbs.Len())

	n := rand.Intn(10)
	hbs = NewHistogramBucketSlice(n)
	defaultVal := NewHistogramBucket()
	assert.EqualValues(t, n, hbs.Len())
	for i := 0; i < hbs.Len(); i++ {
		assert.EqualValues(t, defaultVal, hbs.Get(i))
	}
}

func TestNewSummaryDataPointSlice(t *testing.T) {
	dps := NewSummaryDataPointSlice(0)
	assert.EqualValues(t, 0, len(dps))

	n := rand.Intn(10)
	dps = NewSummaryDataPointSlice(n)
	assert.EqualValues(t, n, len(dps))
	for link := range dps {
		assert.NotNil(t, link)
	}
}

func TestNewSummaryValueAtPercentileSlice(t *testing.T) {
	vps := NewSummaryValueAtPercentileSlice(0)
	assert.EqualValues(t, 0, vps.Len())

	n := rand.Intn(10)
	vps = NewSummaryValueAtPercentileSlice(n)
	defaultVal := NewSummaryValueAtPercentile()
	assert.EqualValues(t, n, vps.Len())
	for i := 0; i < vps.Len(); i++ {
		assert.EqualValues(t, defaultVal, vps.Get(i))
	}
}

func TestOtlpToInternalReadOnly(t *testing.T) {
	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestIntMetric(), generateTestDoubleMetric(), generateTestHistogramMetric(), generateTestSummaryMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	assert.EqualValues(t, 1, len(resourceMetrics))
	resourceMetric := resourceMetrics[0]
	assert.EqualValues(t, newResource(generateTestResource()), resourceMetric.Resource())
	metrics := resourceMetric.InstrumentationLibraryMetrics()[0].Metrics()
	assert.EqualValues(t, 4, len(metrics))
	// Check int64 metric
	metricInt := metrics[0]
	assert.EqualValues(t, "my_metric_int", metricInt.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricInt.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricInt.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeCounterInt64, metricInt.MetricDescriptor().Type())
	assert.EqualValues(t, NewStringMap(map[string]string{}), metricInt.MetricDescriptor().LabelsMap())
	int64DataPoints := metricInt.Int64DataPoints()
	assert.EqualValues(t, 2, len(int64DataPoints))
	// First point
	assert.EqualValues(t, startTime, int64DataPoints[0].StartTime())
	assert.EqualValues(t, endTime, int64DataPoints[0].Timestamp())
	assert.EqualValues(t, 123, int64DataPoints[0].Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), int64DataPoints[0].LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, int64DataPoints[1].StartTime())
	assert.EqualValues(t, endTime, int64DataPoints[1].Timestamp())
	assert.EqualValues(t, 456, int64DataPoints[1].Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), int64DataPoints[1].LabelsMap())
	// Check double metric
	metricDouble := metrics[1]
	assert.EqualValues(t, "my_metric_double", metricDouble.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricDouble.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricDouble.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeCounterDouble, metricDouble.MetricDescriptor().Type())
	doubleDataPoints := metricDouble.DoubleDataPoints()
	assert.EqualValues(t, 2, len(doubleDataPoints))
	// First point
	assert.EqualValues(t, startTime, doubleDataPoints[0].StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints[0].Timestamp())
	assert.EqualValues(t, 123.1, doubleDataPoints[0].Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), doubleDataPoints[0].LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, doubleDataPoints[1].StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints[1].Timestamp())
	assert.EqualValues(t, 456.1, doubleDataPoints[1].Value())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), doubleDataPoints[1].LabelsMap())
	// Check histogram metric
	metricHistogram := metrics[2]
	assert.EqualValues(t, "my_metric_histogram", metricHistogram.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricHistogram.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricHistogram.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeCumulativeHistogram, metricHistogram.MetricDescriptor().Type())
	histogramDataPoints := metricHistogram.HistogramDataPoints()
	assert.EqualValues(t, 2, len(histogramDataPoints))
	// First point
	assert.EqualValues(t, startTime, histogramDataPoints[0].StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints[0].Timestamp())
	assert.EqualValues(t, []float64{1, 2}, histogramDataPoints[0].ExplicitBounds())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), histogramDataPoints[0].LabelsMap())
	assert.EqualValues(t, 3, histogramDataPoints[0].Buckets().Len())
	assert.EqualValues(t, 10, histogramDataPoints[0].Buckets().Get(0).Count())
	assert.EqualValues(t, 15, histogramDataPoints[0].Buckets().Get(1).Count())
	assert.EqualValues(t, 1.5, histogramDataPoints[0].Buckets().Get(1).Exemplar().Value())
	assert.EqualValues(t, startTime, histogramDataPoints[0].Buckets().Get(1).Exemplar().Timestamp())
	assert.EqualValues(t, NewStringMap(map[string]string{"key_a1": "value_a1"}), histogramDataPoints[0].Buckets().Get(1).Exemplar().Attachments())
	assert.EqualValues(t, 1, histogramDataPoints[0].Buckets().Get(2).Count())
	// Second point
	assert.EqualValues(t, startTime, histogramDataPoints[1].StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints[1].Timestamp())
	assert.EqualValues(t, []float64{1}, histogramDataPoints[1].ExplicitBounds())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), histogramDataPoints[1].LabelsMap())
	assert.EqualValues(t, 2, histogramDataPoints[1].Buckets().Len())
	assert.EqualValues(t, 10, histogramDataPoints[1].Buckets().Get(0).Count())
	assert.EqualValues(t, 1, histogramDataPoints[1].Buckets().Get(1).Count())
	// Check summary metric
	metricSummary := metrics[3]
	assert.EqualValues(t, "my_metric_summary", metricSummary.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricSummary.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricSummary.MetricDescriptor().Unit())
	assert.EqualValues(t, MetricTypeSummary, metricSummary.MetricDescriptor().Type())
	summaryDataPoints := metricSummary.SummaryDataPoints()
	assert.EqualValues(t, 2, len(summaryDataPoints))
	// First point
	assert.EqualValues(t, startTime, summaryDataPoints[0].StartTime())
	assert.EqualValues(t, endTime, summaryDataPoints[0].Timestamp())
	assert.EqualValues(t, NewStringMap(map[string]string{"key0": "value0"}), summaryDataPoints[0].LabelsMap())
	assert.EqualValues(t, 2, summaryDataPoints[0].ValueAtPercentiles().Len())
	assert.EqualValues(t, 0.0, summaryDataPoints[0].ValueAtPercentiles().Get(0).Percentile())
	assert.EqualValues(t, 1.23, summaryDataPoints[0].ValueAtPercentiles().Get(0).Value())
	assert.EqualValues(t, 1.0, summaryDataPoints[0].ValueAtPercentiles().Get(1).Percentile())
	assert.EqualValues(t, 4.56, summaryDataPoints[0].ValueAtPercentiles().Get(1).Value())
	// Second point
	assert.EqualValues(t, startTime, summaryDataPoints[1].StartTime())
	assert.EqualValues(t, endTime, summaryDataPoints[1].Timestamp())
	assert.EqualValues(t, NewStringMap(map[string]string{"key1": "value1"}), summaryDataPoints[1].LabelsMap())
	assert.EqualValues(t, 2, summaryDataPoints[1].ValueAtPercentiles().Len())
	assert.EqualValues(t, 0.5, summaryDataPoints[1].ValueAtPercentiles().Get(0).Percentile())
	assert.EqualValues(t, 4.56, summaryDataPoints[1].ValueAtPercentiles().Get(0).Value())
	assert.EqualValues(t, 0.9, summaryDataPoints[1].ValueAtPercentiles().Get(1).Percentile())
	assert.EqualValues(t, 7.89, summaryDataPoints[1].ValueAtPercentiles().Get(1).Value())
}
func TestOtlpToFromInternalReadOnly(t *testing.T) {
	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestIntMetric(), generateTestDoubleMetric(), generateTestHistogramMetric(), generateTestSummaryMetric()},
				},
			},
		},
	})
	// Test that nothing changed
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestIntMetric(), generateTestDoubleMetric(), generateTestHistogramMetric(), generateTestSummaryMetric()},
				},
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalIntPointsMutating(t *testing.T) {
	newLabels := NewStringMap(map[string]string{"k": "v"})

	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestIntMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics[0].InstrumentationLibraryMetrics()[0].Metrics()[0]
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_int")
	assert.EqualValues(t, "new_my_metric_int", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetMetricType(MetricTypeGaugeInt64)
	assert.EqualValues(t, MetricTypeGaugeInt64, metric.MetricDescriptor().Type())
	metric.MetricDescriptor().SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, metric.MetricDescriptor().LabelsMap())
	// Mutate DataPoints
	int64DataPoints := metric.Int64DataPoints()[:1]
	int64DataPoints[0].SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, int64DataPoints[0].StartTime())
	int64DataPoints[0].SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, int64DataPoints[0].Timestamp())
	int64DataPoints[0].SetValue(124)
	assert.EqualValues(t, 124, int64DataPoints[0].Value())
	int64DataPoints[0].SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, int64DataPoints[0].LabelsMap())

	assert.EqualValues(t, 2, len(metric.Int64DataPoints()))
	metric.SetInt64DataPoints(int64DataPoints)
	assert.EqualValues(t, 1, len(metric.Int64DataPoints()))

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
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
									StartTimeUnixnano: startTime + 1,
									TimestampUnixnano: endTime + 1,
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
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestDoubleMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics[0].InstrumentationLibraryMetrics()[0].Metrics()[0]
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_double")
	assert.EqualValues(t, "new_my_metric_double", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetMetricType(MetricTypeGaugeDouble)
	assert.EqualValues(t, MetricTypeGaugeDouble, metric.MetricDescriptor().Type())
	metric.MetricDescriptor().SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, metric.MetricDescriptor().LabelsMap())
	// Mutate DataPoints
	doubleDataPoints := metric.DoubleDataPoints()[:1]
	doubleDataPoints[0].SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, doubleDataPoints[0].StartTime())
	doubleDataPoints[0].SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, doubleDataPoints[0].Timestamp())
	doubleDataPoints[0].SetValue(124.1)
	assert.EqualValues(t, 124.1, doubleDataPoints[0].Value())
	doubleDataPoints[0].SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, doubleDataPoints[0].LabelsMap())

	assert.EqualValues(t, 2, len(metric.DoubleDataPoints()))
	metric.SetDoubleDataPoints(doubleDataPoints)
	assert.EqualValues(t, 1, len(metric.DoubleDataPoints()))

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
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
									StartTimeUnixnano: startTime + 1,
									TimestampUnixnano: endTime + 1,
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
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestHistogramMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics[0].InstrumentationLibraryMetrics()[0].Metrics()[0]
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_histogram")
	assert.EqualValues(t, "new_my_metric_histogram", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetMetricType(MetricTypeGaugeHistogram)
	assert.EqualValues(t, MetricTypeGaugeHistogram, metric.MetricDescriptor().Type())
	metric.MetricDescriptor().SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, metric.MetricDescriptor().LabelsMap())
	// Mutate DataPoints
	histogramDataPoints := metric.HistogramDataPoints()[:1]
	histogramDataPoints[0].SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints[0].StartTime())
	histogramDataPoints[0].SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints[0].Timestamp())
	histogramDataPoints[0].SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, histogramDataPoints[0].LabelsMap())
	histogramDataPoints[0].SetExplicitBounds([]float64{1})
	assert.EqualValues(t, []float64{1}, histogramDataPoints[0].ExplicitBounds())
	assert.EqualValues(t, 3, histogramDataPoints[0].Buckets().Len())
	histogramDataPoints[0].Buckets().Resize(0, 2)
	assert.EqualValues(t, 2, histogramDataPoints[0].Buckets().Len())
	buckets := histogramDataPoints[0].Buckets()
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

	assert.EqualValues(t, 2, len(metric.HistogramDataPoints()))
	metric.SetHistogramDataPoints(histogramDataPoints)
	assert.EqualValues(t, 1, len(metric.HistogramDataPoints()))

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
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
									StartTimeUnixnano: startTime + 1,
									TimestampUnixnano: endTime + 1,
									Buckets: []*otlpmetrics.HistogramDataPoint_Bucket{
										{
											Count: 21,
										},
										{
											Count: 32,
											Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
												Value:             10.5,
												TimestampUnixnano: startTime + 1,
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
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestSummaryMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics[0].InstrumentationLibraryMetrics()[0].Metrics()[0]
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
	summaryDataPoints := metric.SummaryDataPoints()[:1]
	summaryDataPoints[0].SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, summaryDataPoints[0].StartTime())
	summaryDataPoints[0].SetTimestamp(TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, summaryDataPoints[0].Timestamp())
	summaryDataPoints[0].SetLabelsMap(newLabels)
	assert.EqualValues(t, newLabels, summaryDataPoints[0].LabelsMap())
	// Mutate ValueAtPercentiles
	assert.EqualValues(t, 2, summaryDataPoints[0].ValueAtPercentiles().Len())
	summaryDataPoints[0].ValueAtPercentiles().Remove(1)
	assert.EqualValues(t, 1, summaryDataPoints[0].ValueAtPercentiles().Len())
	valueAtPercentile := summaryDataPoints[0].ValueAtPercentiles().Get(0)
	valueAtPercentile.SetValue(1.24)
	assert.EqualValues(t, 1.24, valueAtPercentile.Value())
	valueAtPercentile.SetPercentile(0.1)
	assert.EqualValues(t, 0.1, valueAtPercentile.Percentile())
	assert.EqualValues(t, 2, len(metric.SummaryDataPoints()))
	metric.SetSummaryDataPoints(summaryDataPoints)
	assert.EqualValues(t, 1, len(metric.SummaryDataPoints()))

	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
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
									StartTimeUnixnano: startTime + 1,
									TimestampUnixnano: endTime + 1,
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
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestIntMetric(), generateTestDoubleMetric(), generateTestHistogramMetric(), generateTestSummaryMetric()},
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
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestIntMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics()[0].InstrumentationLibraryMetrics()[0].Metrics()[0].Int64DataPoints()[0].LabelsMap().Upsert("key0", "value0")
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_DoublePoints_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestDoubleMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics()[0].InstrumentationLibraryMetrics()[0].Metrics()[0].DoubleDataPoints()[0].LabelsMap().Upsert("key0", "value0")
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_HistogramPoints_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestHistogramMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics()[0].InstrumentationLibraryMetrics()[0].Metrics()[0].HistogramDataPoints()[0].LabelsMap().Upsert("key0", "value0")
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_SummaryPoints_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestSummaryMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics()[0].InstrumentationLibraryMetrics()[0].Metrics()[0].SummaryDataPoints()[0].LabelsMap().Upsert("key0", "value0")
		MetricDataToOtlp(md)
	}
}

func generateTestResource() *otlpresource.Resource {
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

func generateTestInstrumentationLibrary() *otlpcommon.InstrumentationLibrary {
	return &otlpcommon.InstrumentationLibrary{
		Name:    "test",
		Version: "",
	}
}

func generateTestIntMetric() *otlpmetrics.Metric {
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
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
				Value:             123,
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
				Value:             456,
			},
		},
	}
}
func generateTestDoubleMetric() *otlpmetrics.Metric {
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
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
				Value:             123.1,
			},
			{
				Labels: []*otlpcommon.StringKeyValue{
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
				Value:             456.1,
			},
		},
	}
}

func generateTestHistogramMetric() *otlpmetrics.Metric {
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
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
				Buckets: []*otlpmetrics.HistogramDataPoint_Bucket{
					{
						Count: 10,
					},
					{
						Count: 15,
						Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
							Value:             1.5,
							TimestampUnixnano: startTime,
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
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
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

func generateTestSummaryMetric() *otlpmetrics.Metric {
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
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
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
				StartTimeUnixnano: startTime,
				TimestampUnixnano: endTime,
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
