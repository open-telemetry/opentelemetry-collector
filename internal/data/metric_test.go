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

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
)

const (
	startTime = uint64(12578940000000012345)
	endTime   = uint64(12578940000000054321)
)

func TestMetricCount(t *testing.T) {
	md := NewMetricData()
	assert.EqualValues(t, 0, md.MetricCount())

	md.ResourceMetrics().Resize(1)
	assert.EqualValues(t, 0, md.MetricCount())

	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(1)
	assert.EqualValues(t, 0, md.MetricCount())

	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().Resize(1)
	assert.EqualValues(t, 1, md.MetricCount())

	rms := md.ResourceMetrics()
	rms.Resize(3)
	rms.At(0).InstrumentationLibraryMetrics().Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().Resize(1)
	rms.At(1).InstrumentationLibraryMetrics().Resize(1)
	rms.At(2).InstrumentationLibraryMetrics().Resize(1)
	rms.At(2).InstrumentationLibraryMetrics().At(0).Metrics().Resize(5)
	assert.EqualValues(t, 6, md.MetricCount())
}

func TestMetricSize(t *testing.T) {
	md := NewMetricData()
	assert.Equal(t, 0, md.Size())
	rms := md.ResourceMetrics()
	rms.Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleDataPoints().Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleDataPoints().At(0).SetValue(16.6)
	otlp := MetricDataToOtlp(md)
	size := 0
	sizeBytes := 0
	for _, rmerics := range otlp {
		size += rmerics.Size()
		bts, err := rmerics.Marshal()
		require.NoError(t, err)
		sizeBytes += len(bts)
	}
	assert.Equal(t, size, md.Size())
	assert.Equal(t, sizeBytes, md.Size())
}

func TestSizeWithNils(t *testing.T) {
	assert.Equal(t, 0, MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{nil, {}}).Size())
}

func TestMetricCountWithNils(t *testing.T) {
	assert.EqualValues(t, 0, MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{nil, {}}).MetricCount())
	assert.EqualValues(t, 0, MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{nil, {}},
		},
	}).MetricCount())
	assert.EqualValues(t, 2, MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{nil, {}},
				},
			},
		},
	}).MetricCount())
}

func TestMetricAndDataPointCount(t *testing.T) {
	md := NewMetricData()
	ms, dps := md.MetricAndDataPointCount()
	assert.EqualValues(t, 0, ms)
	assert.EqualValues(t, 0, dps)

	rms := md.ResourceMetrics()
	rms.Resize(1)
	ms, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 0, ms)
	assert.EqualValues(t, 0, dps)

	ilms := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics()
	ilms.Resize(1)
	ms, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 0, ms)
	assert.EqualValues(t, 0, dps)

	ilms.At(0).Metrics().Resize(1)
	ms, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 1, ms)
	assert.EqualValues(t, 0, dps)
	ilms.At(0).Metrics().At(0).Int64DataPoints().Resize(3)
	_, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 3, dps)

	md = NewMetricData()
	rms = md.ResourceMetrics()
	rms.Resize(3)
	rms.At(0).InstrumentationLibraryMetrics().Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().Resize(1)
	rms.At(1).InstrumentationLibraryMetrics().Resize(1)
	rms.At(2).InstrumentationLibraryMetrics().Resize(1)
	ilms = rms.At(2).InstrumentationLibraryMetrics()
	ilms.Resize(1)
	ilms.At(0).Metrics().Resize(5)
	ms, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 6, ms)
	assert.EqualValues(t, 0, dps)
	ilms.At(0).Metrics().At(1).Int64DataPoints().Resize(1)
	ilms.At(0).Metrics().At(3).Int64DataPoints().Resize(3)
	ms, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 6, ms)
	assert.EqualValues(t, 4, dps)
}

func TestMetricAndDataPointCountWithNil(t *testing.T) {
	ms, dps := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{nil, {}}).MetricAndDataPointCount()
	assert.EqualValues(t, 0, ms)
	assert.EqualValues(t, 0, dps)

	ms, dps = MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{nil, {}},
		},
	}).MetricAndDataPointCount()
	assert.EqualValues(t, 0, ms)
	assert.EqualValues(t, 0, dps)

	ms, dps = MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{nil, {}},
				},
			},
		},
	}).MetricAndDataPointCount()
	assert.EqualValues(t, 2, ms)
	assert.EqualValues(t, 0, dps)

	ms, dps = MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{{
						Int64DataPoints: []*otlpmetrics.Int64DataPoint{
							nil, {},
						},
						DoubleDataPoints: []*otlpmetrics.DoubleDataPoint{
							nil, {},
						},
						HistogramDataPoints: []*otlpmetrics.HistogramDataPoint{
							nil, {},
						},
						SummaryDataPoints: []*otlpmetrics.SummaryDataPoint{
							nil, {},
						},
					}},
				},
			},
		},
	}).MetricAndDataPointCount()
	assert.EqualValues(t, 1, ms)
	assert.EqualValues(t, 8, dps)

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

	resourceMetric := resourceMetrics.At(0)
	assert.EqualValues(t, pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
		"string": pdata.NewAttributeValueString("string-resource"),
	}), resourceMetric.Resource().Attributes())
	metrics := resourceMetric.InstrumentationLibraryMetrics().At(0).Metrics()
	assert.EqualValues(t, 4, metrics.Len())

	// Check int64 metric
	metricInt := metrics.At(0)
	assert.EqualValues(t, "my_metric_int", metricInt.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricInt.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricInt.MetricDescriptor().Unit())
	assert.EqualValues(t, pdata.MetricTypeMonotonicInt64, metricInt.MetricDescriptor().Type())
	int64DataPoints := metricInt.Int64DataPoints()
	assert.EqualValues(t, 2, int64DataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, int64DataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, int64DataPoints.At(0).Timestamp())
	assert.EqualValues(t, 123, int64DataPoints.At(0).Value())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), int64DataPoints.At(0).LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, int64DataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, int64DataPoints.At(1).Timestamp())
	assert.EqualValues(t, 456, int64DataPoints.At(1).Value())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), int64DataPoints.At(1).LabelsMap())

	// Check double metric
	metricDouble := metrics.At(1)
	assert.EqualValues(t, "my_metric_double", metricDouble.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricDouble.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricDouble.MetricDescriptor().Unit())
	assert.EqualValues(t, pdata.MetricTypeMonotonicDouble, metricDouble.MetricDescriptor().Type())
	doubleDataPoints := metricDouble.DoubleDataPoints()
	assert.EqualValues(t, 2, doubleDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, doubleDataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints.At(0).Timestamp())
	assert.EqualValues(t, 123.1, doubleDataPoints.At(0).Value())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), doubleDataPoints.At(0).LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, doubleDataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints.At(1).Timestamp())
	assert.EqualValues(t, 456.1, doubleDataPoints.At(1).Value())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), doubleDataPoints.At(1).LabelsMap())

	// Check histogram metric
	metricHistogram := metrics.At(2)
	assert.EqualValues(t, "my_metric_histogram", metricHistogram.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricHistogram.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricHistogram.MetricDescriptor().Unit())
	assert.EqualValues(t, pdata.MetricTypeHistogram, metricHistogram.MetricDescriptor().Type())
	histogramDataPoints := metricHistogram.HistogramDataPoints()
	assert.EqualValues(t, 2, histogramDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, histogramDataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.At(0).Timestamp())
	assert.EqualValues(t, []float64{1, 2}, histogramDataPoints.At(0).ExplicitBounds())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), histogramDataPoints.At(0).LabelsMap())
	assert.EqualValues(t, 3, histogramDataPoints.At(0).Buckets().Len())
	assert.EqualValues(t, 10, histogramDataPoints.At(0).Buckets().At(0).Count())
	assert.EqualValues(t, 15, histogramDataPoints.At(0).Buckets().At(1).Count())
	assert.EqualValues(t, 1.5, histogramDataPoints.At(0).Buckets().At(1).Exemplar().Value())
	assert.EqualValues(t, startTime, histogramDataPoints.At(0).Buckets().At(1).Exemplar().Timestamp())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key_a1": "value_a1"}), histogramDataPoints.At(0).Buckets().At(1).Exemplar().Attachments())
	assert.EqualValues(t, 1, histogramDataPoints.At(0).Buckets().At(2).Count())
	// Second point
	assert.EqualValues(t, startTime, histogramDataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.At(1).Timestamp())
	assert.EqualValues(t, []float64{1}, histogramDataPoints.At(1).ExplicitBounds())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), histogramDataPoints.At(1).LabelsMap())
	assert.EqualValues(t, 2, histogramDataPoints.At(1).Buckets().Len())
	assert.EqualValues(t, 10, histogramDataPoints.At(1).Buckets().At(0).Count())
	assert.EqualValues(t, 1, histogramDataPoints.At(1).Buckets().At(1).Count())

	// Check summary metric
	metricSummary := metrics.At(3)
	assert.EqualValues(t, "my_metric_summary", metricSummary.MetricDescriptor().Name())
	assert.EqualValues(t, "My metric", metricSummary.MetricDescriptor().Description())
	assert.EqualValues(t, "ms", metricSummary.MetricDescriptor().Unit())
	assert.EqualValues(t, pdata.MetricTypeSummary, metricSummary.MetricDescriptor().Type())
	summaryDataPoints := metricSummary.SummaryDataPoints()
	assert.EqualValues(t, 2, summaryDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, summaryDataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, summaryDataPoints.At(0).Timestamp())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), summaryDataPoints.At(0).LabelsMap())
	assert.EqualValues(t, 2, summaryDataPoints.At(0).ValueAtPercentiles().Len())
	assert.EqualValues(t, 0.0, summaryDataPoints.At(0).ValueAtPercentiles().At(0).Percentile())
	assert.EqualValues(t, 1.23, summaryDataPoints.At(0).ValueAtPercentiles().At(0).Value())
	assert.EqualValues(t, 1.0, summaryDataPoints.At(0).ValueAtPercentiles().At(1).Percentile())
	assert.EqualValues(t, 4.56, summaryDataPoints.At(0).ValueAtPercentiles().At(1).Value())
	// Second point
	assert.EqualValues(t, startTime, summaryDataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, summaryDataPoints.At(1).Timestamp())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), summaryDataPoints.At(1).LabelsMap())
	assert.EqualValues(t, 2, summaryDataPoints.At(1).ValueAtPercentiles().Len())
	assert.EqualValues(t, 0.5, summaryDataPoints.At(1).ValueAtPercentiles().At(0).Percentile())
	assert.EqualValues(t, 4.56, summaryDataPoints.At(1).ValueAtPercentiles().At(0).Value())
	assert.EqualValues(t, 0.9, summaryDataPoints.At(1).ValueAtPercentiles().At(1).Percentile())
	assert.EqualValues(t, 7.89, summaryDataPoints.At(1).ValueAtPercentiles().At(1).Value())
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
	newLabels := pdata.NewStringMap().InitFromMap(map[string]string{"k": "v"})

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
	metric := resourceMetrics.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_int")
	assert.EqualValues(t, "new_my_metric_int", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetType(pdata.MetricTypeInt64)
	assert.EqualValues(t, pdata.MetricTypeInt64, metric.MetricDescriptor().Type())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.Int64DataPoints().Len())
	metric.Int64DataPoints().Resize(1)
	assert.EqualValues(t, 1, metric.Int64DataPoints().Len())
	int64DataPoints := metric.Int64DataPoints()
	int64DataPoints.At(0).SetStartTime(pdata.TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, int64DataPoints.At(0).StartTime())
	int64DataPoints.At(0).SetTimestamp(pdata.TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, int64DataPoints.At(0).Timestamp())
	int64DataPoints.At(0).SetValue(124)
	assert.EqualValues(t, 124, int64DataPoints.At(0).Value())
	int64DataPoints.At(0).LabelsMap().Delete("key0")
	int64DataPoints.At(0).LabelsMap().Upsert("k", "v")
	assert.EqualValues(t, newLabels, int64DataPoints.At(0).LabelsMap())

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
								Type:        otlpmetrics.MetricDescriptor_INT64,
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
	newLabels := pdata.NewStringMap().InitFromMap(map[string]string{"k": "v"})

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
	metric := resourceMetrics.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_double")
	assert.EqualValues(t, "new_my_metric_double", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetType(pdata.MetricTypeDouble)
	assert.EqualValues(t, pdata.MetricTypeDouble, metric.MetricDescriptor().Type())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.DoubleDataPoints().Len())
	metric.DoubleDataPoints().Resize(1)
	assert.EqualValues(t, 1, metric.DoubleDataPoints().Len())
	doubleDataPoints := metric.DoubleDataPoints()
	doubleDataPoints.At(0).SetStartTime(pdata.TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, doubleDataPoints.At(0).StartTime())
	doubleDataPoints.At(0).SetTimestamp(pdata.TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, doubleDataPoints.At(0).Timestamp())
	doubleDataPoints.At(0).SetValue(124.1)
	assert.EqualValues(t, 124.1, doubleDataPoints.At(0).Value())
	doubleDataPoints.At(0).LabelsMap().Delete("key0")
	doubleDataPoints.At(0).LabelsMap().Upsert("k", "v")
	assert.EqualValues(t, newLabels, doubleDataPoints.At(0).LabelsMap())

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
								Type:        otlpmetrics.MetricDescriptor_DOUBLE,
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
	newLabels := pdata.NewStringMap().InitFromMap(map[string]string{"k": "v"})

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
	metric := resourceMetrics.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_histogram")
	assert.EqualValues(t, "new_my_metric_histogram", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	metric.MetricDescriptor().SetType(pdata.MetricTypeHistogram)
	assert.EqualValues(t, pdata.MetricTypeHistogram, metric.MetricDescriptor().Type())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.HistogramDataPoints().Len())
	metric.HistogramDataPoints().Resize(1)
	assert.EqualValues(t, 1, metric.HistogramDataPoints().Len())
	histogramDataPoints := metric.HistogramDataPoints()
	histogramDataPoints.At(0).SetStartTime(pdata.TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.At(0).StartTime())
	histogramDataPoints.At(0).SetTimestamp(pdata.TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints.At(0).Timestamp())
	histogramDataPoints.At(0).LabelsMap().Delete("key0")
	histogramDataPoints.At(0).LabelsMap().Upsert("k", "v")
	assert.EqualValues(t, newLabels, histogramDataPoints.At(0).LabelsMap())
	histogramDataPoints.At(0).SetExplicitBounds([]float64{1})
	assert.EqualValues(t, []float64{1}, histogramDataPoints.At(0).ExplicitBounds())
	assert.EqualValues(t, 3, histogramDataPoints.At(0).Buckets().Len())
	histogramDataPoints.At(0).Buckets().Resize(2)
	assert.EqualValues(t, 2, histogramDataPoints.At(0).Buckets().Len())
	buckets := histogramDataPoints.At(0).Buckets()
	buckets.At(0).SetCount(21)
	assert.EqualValues(t, 21, buckets.At(0).Count())
	buckets.At(1).SetCount(32)
	assert.EqualValues(t, 32, buckets.At(1).Count())
	buckets.At(1).Exemplar().SetTimestamp(pdata.TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, buckets.At(1).Exemplar().Timestamp())
	buckets.At(1).Exemplar().SetValue(10.5)
	assert.EqualValues(t, 10.5, buckets.At(1).Exemplar().Value())
	buckets.At(1).Exemplar().Attachments().Delete("key_a1")
	buckets.At(1).Exemplar().Attachments().Upsert("k", "v")
	assert.EqualValues(t, newLabels, buckets.At(1).Exemplar().Attachments())

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
								Type:        otlpmetrics.MetricDescriptor_HISTOGRAM,
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
	newLabels := pdata.NewStringMap().InitFromMap(map[string]string{"k": "v"})
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
	metric := resourceMetrics.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.MetricDescriptor().SetName("new_my_metric_summary")
	assert.EqualValues(t, "new_my_metric_summary", metric.MetricDescriptor().Name())
	metric.MetricDescriptor().SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.MetricDescriptor().Description())
	metric.MetricDescriptor().SetUnit("1")
	assert.EqualValues(t, "1", metric.MetricDescriptor().Unit())
	// Mutate DataPoints
	assert.EqualValues(t, 2, metric.SummaryDataPoints().Len())
	metric.SummaryDataPoints().Resize(1)
	assert.EqualValues(t, 1, metric.SummaryDataPoints().Len())
	summaryDataPoints := metric.SummaryDataPoints()
	summaryDataPoints.At(0).SetStartTime(pdata.TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, summaryDataPoints.At(0).StartTime())
	summaryDataPoints.At(0).SetTimestamp(pdata.TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, summaryDataPoints.At(0).Timestamp())
	summaryDataPoints.At(0).LabelsMap().Delete("key0")
	summaryDataPoints.At(0).LabelsMap().Upsert("k", "v")
	assert.EqualValues(t, newLabels, summaryDataPoints.At(0).LabelsMap())
	// Mutate ValueAtPercentiles
	assert.EqualValues(t, 2, summaryDataPoints.At(0).ValueAtPercentiles().Len())
	summaryDataPoints.At(0).ValueAtPercentiles().Resize(1)
	assert.EqualValues(t, 1, summaryDataPoints.At(0).ValueAtPercentiles().Len())
	valueAtPercentile := summaryDataPoints.At(0).ValueAtPercentiles().At(0)
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
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Int64DataPoints().At(0).LabelsMap().Upsert("key0", "value2")
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
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleDataPoints().At(0).LabelsMap().Upsert("key0", "value2")
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
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).HistogramDataPoints().At(0).LabelsMap().Upsert("key0", "value2")
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
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).SummaryDataPoints().At(0).LabelsMap().Upsert("key0", "value2")
		MetricDataToOtlp(md)
	}
}

func generateTestProtoResource() *otlpresource.Resource {
	return &otlpresource.Resource{
		Attributes: []*otlpcommon.KeyValue{
			{
				Key:   "string",
				Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "string-resource"}},
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
			Type:        otlpmetrics.MetricDescriptor_MONOTONIC_INT64,
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
			Type:        otlpmetrics.MetricDescriptor_MONOTONIC_DOUBLE,
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
			Type:        otlpmetrics.MetricDescriptor_HISTOGRAM,
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
