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
	doubleHistogram := pdata.NewDoubleHistogram()
	doubleHistogram.InitEmpty()
	doubleHistogram.DataPoints().Resize(1)
	doubleHistogram.DataPoints().At(0).SetCount(123)
	doubleHistogram.DataPoints().At(0).SetSum(123)
	rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).SetDoubleHistogramData(doubleHistogram)
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
	intSum := pdata.NewIntSum()
	intSum.InitEmpty()
	intSum.DataPoints().Resize(3)
	ilms.At(0).Metrics().At(0).SetIntSumData(intSum)
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
	doubleGauge := pdata.NewDoubleGauge()
	doubleGauge.InitEmpty()
	doubleGauge.DataPoints().Resize(1)
	ilms.At(0).Metrics().At(1).SetDoubleGaugeData(doubleGauge)
	intHistogram := pdata.NewIntHistogram()
	intHistogram.InitEmpty()
	intHistogram.DataPoints().Resize(3)
	ilms.At(0).Metrics().At(3).SetIntHistogramData(intHistogram)
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
						Data: &otlpmetrics.Metric_DoubleGauge{
							DoubleGauge: &otlpmetrics.DoubleGauge{
								DataPoints: []*otlpmetrics.DoubleDataPoint{
									nil, {},
								},
							},
						},
					}},
				},
			},
		},
	}).MetricAndDataPointCount()
	assert.EqualValues(t, 1, ms)
	assert.EqualValues(t, 2, dps)

}

func TestOtlpToInternalReadOnly(t *testing.T) {
	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntGaugeMetric(), generateTestProtoDoubleSumMetric(), generateTestProtoDoubleHistogramMetric()},
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
	assert.EqualValues(t, 3, metrics.Len())

	// Check int64 metric
	metricInt := metrics.At(0)
	assert.EqualValues(t, "my_metric_int", metricInt.Name())
	assert.EqualValues(t, "My metric", metricInt.Description())
	assert.EqualValues(t, "ms", metricInt.Unit())
	assert.EqualValues(t, pdata.MetricDataIntGauge, metricInt.DataType())
	int64DataPoints := metricInt.IntGaugeData().DataPoints()
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
	assert.EqualValues(t, "my_metric_double", metricDouble.Name())
	assert.EqualValues(t, "My metric", metricDouble.Description())
	assert.EqualValues(t, "ms", metricDouble.Unit())
	assert.EqualValues(t, pdata.MetricDataDoubleSum, metricDouble.DataType())
	dsd := metricDouble.DoubleSumData()
	assert.EqualValues(t, pdata.AggregationTemporalityCumulative, dsd.AggregationTemporality())
	doubleDataPoints := dsd.DataPoints()
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
	assert.EqualValues(t, "my_metric_histogram", metricHistogram.Name())
	assert.EqualValues(t, "My metric", metricHistogram.Description())
	assert.EqualValues(t, "ms", metricHistogram.Unit())
	assert.EqualValues(t, pdata.MetricDataDoubleHistogram, metricHistogram.DataType())
	dhd := metricHistogram.DoubleHistogramData()
	assert.EqualValues(t, pdata.AggregationTemporalityDelta, dhd.AggregationTemporality())
	histogramDataPoints := dhd.DataPoints()
	assert.EqualValues(t, 2, histogramDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, histogramDataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.At(0).Timestamp())
	assert.EqualValues(t, []float64{1, 2}, histogramDataPoints.At(0).ExplicitBounds())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), histogramDataPoints.At(0).LabelsMap())
	assert.EqualValues(t, []uint64{10, 15, 1}, histogramDataPoints.At(0).BucketCounts())
	// Second point
	assert.EqualValues(t, startTime, histogramDataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.At(1).Timestamp())
	assert.EqualValues(t, []float64{1}, histogramDataPoints.At(1).ExplicitBounds())
	assert.EqualValues(t, pdata.NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), histogramDataPoints.At(1).LabelsMap())
	assert.EqualValues(t, []uint64{10, 1}, histogramDataPoints.At(1).BucketCounts())
}

func TestOtlpToFromInternalReadOnly(t *testing.T) {
	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntGaugeMetric(), generateTestProtoDoubleSumMetric(), generateTestProtoDoubleHistogramMetric()},
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
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntGaugeMetric(), generateTestProtoDoubleSumMetric(), generateTestProtoDoubleHistogramMetric()},
				},
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalIntGaugeMutating(t *testing.T) {
	newLabels := pdata.NewStringMap().InitFromMap(map[string]string{"k": "v"})

	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntGaugeMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_int")
	assert.EqualValues(t, "new_my_metric_int", metric.Name())
	metric.SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.EqualValues(t, "1", metric.Unit())
	// Mutate DataPoints
	igd := metric.IntGaugeData()
	assert.EqualValues(t, 2, igd.DataPoints().Len())
	igd.DataPoints().Resize(1)
	assert.EqualValues(t, 1, igd.DataPoints().Len())
	int64DataPoints := igd.DataPoints()
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
							Name:        "new_my_metric_int",
							Description: "My new metric",
							Unit:        "1",
							Data: &otlpmetrics.Metric_IntGauge{
								IntGauge: &otlpmetrics.IntGauge{
									DataPoints: []*otlpmetrics.IntDataPoint{
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
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalDoubleSumMutating(t *testing.T) {
	newLabels := pdata.NewStringMap().InitFromMap(map[string]string{"k": "v"})

	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoDoubleSumMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_double")
	assert.EqualValues(t, "new_my_metric_double", metric.Name())
	metric.SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.EqualValues(t, "1", metric.Unit())
	// Mutate DataPoints
	dsd := metric.DoubleSumData()
	assert.EqualValues(t, 2, dsd.DataPoints().Len())
	dsd.DataPoints().Resize(1)
	assert.EqualValues(t, 1, dsd.DataPoints().Len())
	doubleDataPoints := dsd.DataPoints()
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
							Name:        "new_my_metric_double",
							Description: "My new metric",
							Unit:        "1",
							Data: &otlpmetrics.Metric_DoubleSum{
								DoubleSum: &otlpmetrics.DoubleSum{
									AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
									DataPoints: []*otlpmetrics.DoubleDataPoint{
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
			},
		},
	}, MetricDataToOtlp(metricData))
}

func TestOtlpToFromInternalHistogramMutating(t *testing.T) {
	newLabels := pdata.NewStringMap().InitFromMap(map[string]string{"k": "v"})

	metricData := MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoDoubleHistogramMetric()},
				},
			},
		},
	})
	resourceMetrics := metricData.ResourceMetrics()
	metric := resourceMetrics.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_histogram")
	assert.EqualValues(t, "new_my_metric_histogram", metric.Name())
	metric.SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.EqualValues(t, "1", metric.Unit())
	// Mutate DataPoints
	dhd := metric.DoubleHistogramData()
	assert.EqualValues(t, 2, dhd.DataPoints().Len())
	dhd.DataPoints().Resize(1)
	assert.EqualValues(t, 1, dhd.DataPoints().Len())
	histogramDataPoints := dhd.DataPoints()
	histogramDataPoints.At(0).SetStartTime(pdata.TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.At(0).StartTime())
	histogramDataPoints.At(0).SetTimestamp(pdata.TimestampUnixNano(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints.At(0).Timestamp())
	histogramDataPoints.At(0).LabelsMap().Delete("key0")
	histogramDataPoints.At(0).LabelsMap().Upsert("k", "v")
	assert.EqualValues(t, newLabels, histogramDataPoints.At(0).LabelsMap())
	histogramDataPoints.At(0).SetExplicitBounds([]float64{1})
	assert.EqualValues(t, []float64{1}, histogramDataPoints.At(0).ExplicitBounds())
	histogramDataPoints.At(0).SetBucketCounts([]uint64{21, 32})
	// Test that everything is updated.
	assert.EqualValues(t, []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics: []*otlpmetrics.Metric{
						{
							Name:        "new_my_metric_histogram",
							Description: "My new metric",
							Unit:        "1",
							Data: &otlpmetrics.Metric_DoubleHistogram{
								DoubleHistogram: &otlpmetrics.DoubleHistogram{
									AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
									DataPoints: []*otlpmetrics.DoubleHistogramDataPoint{
										{
											Labels: []*otlpcommon.StringKeyValue{
												{
													Key:   "k",
													Value: "v",
												},
											},
											StartTimeUnixNano: startTime + 1,
											TimeUnixNano:      endTime + 1,
											BucketCounts:      []uint64{21, 32},
											ExplicitBounds:    []float64{1},
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
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntGaugeMetric(), generateTestProtoDoubleSumMetric(), generateTestProtoDoubleHistogramMetric()},
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

func BenchmarkOtlpToFromInternal_IntGauge_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoIntGaugeMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntGaugeData().DataPoints().At(0).LabelsMap().Upsert("key0", "value2")
		MetricDataToOtlp(md)
	}
}

func BenchmarkOtlpToFromInternal_DoubleSum_MutateOneLabel(b *testing.B) {
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateTestProtoResource(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					InstrumentationLibrary: generateTestProtoInstrumentationLibrary(),
					Metrics:                []*otlpmetrics.Metric{generateTestProtoDoubleSumMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleSumData().DataPoints().At(0).LabelsMap().Upsert("key0", "value2")
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
					Metrics:                []*otlpmetrics.Metric{generateTestProtoDoubleHistogramMetric()},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := MetricDataFromOtlp(resourceMetricsList)
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleHistogramData().DataPoints().At(0).LabelsMap().Upsert("key0", "value2")
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

func generateTestProtoIntGaugeMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		Name:        "my_metric_int",
		Description: "My metric",
		Unit:        "ms",
		Data: &otlpmetrics.Metric_IntGauge{
			IntGauge: &otlpmetrics.IntGauge{
				DataPoints: []*otlpmetrics.IntDataPoint{
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
			},
		},
	}
}
func generateTestProtoDoubleSumMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		Name:        "my_metric_double",
		Description: "My metric",
		Unit:        "ms",
		Data: &otlpmetrics.Metric_DoubleSum{
			DoubleSum: &otlpmetrics.DoubleSum{
				AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				DataPoints: []*otlpmetrics.DoubleDataPoint{
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
			},
		},
	}
}

func generateTestProtoDoubleHistogramMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		Name:        "my_metric_histogram",
		Description: "My metric",
		Unit:        "ms",
		Data: &otlpmetrics.Metric_DoubleHistogram{
			DoubleHistogram: &otlpmetrics.DoubleHistogram{
				AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				DataPoints: []*otlpmetrics.DoubleHistogramDataPoint{
					{
						Labels: []*otlpcommon.StringKeyValue{
							{
								Key:   "key0",
								Value: "value0",
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						BucketCounts:      []uint64{10, 15, 1},
						ExplicitBounds:    []float64{1, 2},
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
						BucketCounts:      []uint64{10, 1},
						ExplicitBounds:    []float64{1},
					},
				},
			},
		},
	}
}
