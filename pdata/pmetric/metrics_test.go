// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	startTime = uint64(12578940000000012345)
	endTime   = uint64(12578940000000054321)
)

func TestMetricCount(t *testing.T) {
	md := NewMetrics()
	assert.Equal(t, 0, md.MetricCount())

	rms := md.ResourceMetrics()
	rms.EnsureCapacity(3)
	rm := rms.AppendEmpty()
	assert.Equal(t, 0, md.MetricCount())

	ilm := rm.ScopeMetrics().AppendEmpty()
	assert.Equal(t, 0, md.MetricCount())

	ilm.Metrics().AppendEmpty()
	assert.Equal(t, 1, md.MetricCount())

	rms.AppendEmpty().ScopeMetrics().AppendEmpty()
	ilmm := rms.AppendEmpty().ScopeMetrics().AppendEmpty().Metrics()
	ilmm.EnsureCapacity(5)
	for range 5 {
		ilmm.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.Equal(t, 6, md.MetricCount())
}

func TestMetricCountWithEmpty(t *testing.T) {
	assert.Equal(t, 0, generateMetricsEmptyResource().MetricCount())
	assert.Equal(t, 0, generateMetricsEmptyInstrumentation().MetricCount())
	assert.Equal(t, 1, generateMetricsEmptyMetrics().MetricCount())
}

func TestMetricAndDataPointCount(t *testing.T) {
	md := NewMetrics()
	dps := md.DataPointCount()
	assert.Equal(t, 0, dps)

	rms := md.ResourceMetrics()
	rms.AppendEmpty()
	dps = md.DataPointCount()
	assert.Equal(t, 0, dps)

	ilms := md.ResourceMetrics().At(0).ScopeMetrics()
	ilms.AppendEmpty()
	dps = md.DataPointCount()
	assert.Equal(t, 0, dps)

	ilms.At(0).Metrics().AppendEmpty()
	dps = md.DataPointCount()
	assert.Equal(t, 0, dps)
	intSum := ilms.At(0).Metrics().At(0).SetEmptySum()
	intSum.DataPoints().AppendEmpty()
	intSum.DataPoints().AppendEmpty()
	intSum.DataPoints().AppendEmpty()
	assert.Equal(t, 3, md.DataPointCount())

	md = NewMetrics()
	rms = md.ResourceMetrics()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	rms.AppendEmpty().ScopeMetrics().AppendEmpty()
	rms.AppendEmpty().ScopeMetrics().AppendEmpty()
	ilms = rms.At(2).ScopeMetrics()
	ilm := ilms.At(0).Metrics()
	for range 5 {
		ilm.AppendEmpty()
	}
	assert.Equal(t, 0, md.DataPointCount())

	ilm.At(0).SetEmptyGauge().DataPoints().AppendEmpty()
	assert.Equal(t, 1, md.DataPointCount())

	ilm.At(1).SetEmptySum().DataPoints().AppendEmpty()
	assert.Equal(t, 2, md.DataPointCount())

	ilm.At(2).SetEmptyHistogram().DataPoints().AppendEmpty()
	assert.Equal(t, 3, md.DataPointCount())

	ilm.At(3).SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
	assert.Equal(t, 4, md.DataPointCount())

	ilm.At(4).SetEmptySummary().DataPoints().AppendEmpty()
	assert.Equal(t, 5, md.DataPointCount())
}

func TestDataPointCountWithEmpty(t *testing.T) {
	assert.Equal(t, 0, generateMetricsEmptyResource().DataPointCount())
	assert.Equal(t, 0, generateMetricsEmptyInstrumentation().DataPointCount())
	assert.Equal(t, 0, generateMetricsEmptyMetrics().DataPointCount())
	assert.Equal(t, 1, generateMetricsEmptyDataPoints().DataPointCount())
}

func TestDataPointCountWithNilDataPoints(t *testing.T) {
	md := NewMetrics()
	ilm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	ilm.Metrics().AppendEmpty().SetEmptyGauge()
	ilm.Metrics().AppendEmpty().SetEmptySum()
	ilm.Metrics().AppendEmpty().SetEmptyHistogram()
	ilm.Metrics().AppendEmpty().SetEmptyExponentialHistogram()
	ilm.Metrics().AppendEmpty().SetEmptySummary()
	assert.Equal(t, 0, md.DataPointCount())
}

func TestHistogramWithNilSum(t *testing.T) {
	md := NewMetrics()
	ilm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	histo := ilm.Metrics().AppendEmpty()
	histogramDataPoints := histo.SetEmptyHistogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	dest := ilm.Metrics().AppendEmpty()
	histo.CopyTo(dest)
	assert.Equal(t, histo, dest)
}

func TestHistogramWithValidSum(t *testing.T) {
	md := NewMetrics()
	ilm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	histo := ilm.Metrics().AppendEmpty()
	histogramDataPoints := histo.SetEmptyHistogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	histogramDataPoints.At(0).SetSum(10)
	dest := ilm.Metrics().AppendEmpty()
	histo.CopyTo(dest)
	assert.Equal(t, histo, dest)
}

func TestOtlpToInternalReadOnly(t *testing.T) {
	md := newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}, new(internal.State))
	resourceMetrics := md.ResourceMetrics()
	assert.Equal(t, 1, resourceMetrics.Len())

	resourceMetric := resourceMetrics.At(0)
	assert.Equal(t, map[string]any{
		"string": "string-resource",
	}, resourceMetric.Resource().Attributes().AsRaw())
	metrics := resourceMetric.ScopeMetrics().At(0).Metrics()
	assert.Equal(t, 3, metrics.Len())

	// Check int64 metric
	metricInt := metrics.At(0)
	assert.Equal(t, "my_metric_int", metricInt.Name())
	assert.Equal(t, "My metric", metricInt.Description())
	assert.Equal(t, "ms", metricInt.Unit())
	assert.Equal(t, MetricTypeGauge, metricInt.Type())
	gaugeDataPoints := metricInt.Gauge().DataPoints()
	assert.Equal(t, 2, gaugeDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, gaugeDataPoints.At(0).StartTimestamp())
	assert.EqualValues(t, endTime, gaugeDataPoints.At(0).Timestamp())
	assert.InDelta(t, 123.1, gaugeDataPoints.At(0).DoubleValue(), 0.01)
	assert.Equal(t, map[string]any{"key0": "value0"}, gaugeDataPoints.At(0).Attributes().AsRaw())
	// Second point
	assert.EqualValues(t, startTime, gaugeDataPoints.At(1).StartTimestamp())
	assert.EqualValues(t, endTime, gaugeDataPoints.At(1).Timestamp())
	assert.InDelta(t, 456.1, gaugeDataPoints.At(1).DoubleValue(), 0.01)
	assert.Equal(t, map[string]any{"key1": "value1"}, gaugeDataPoints.At(1).Attributes().AsRaw())

	// Check double metric
	metricDouble := metrics.At(1)
	assert.Equal(t, "my_metric_double", metricDouble.Name())
	assert.Equal(t, "My metric", metricDouble.Description())
	assert.Equal(t, "ms", metricDouble.Unit())
	assert.Equal(t, MetricTypeSum, metricDouble.Type())
	dsd := metricDouble.Sum()
	assert.Equal(t, AggregationTemporalityCumulative, dsd.AggregationTemporality())
	sumDataPoints := dsd.DataPoints()
	assert.Equal(t, 2, sumDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, sumDataPoints.At(0).StartTimestamp())
	assert.EqualValues(t, endTime, sumDataPoints.At(0).Timestamp())
	assert.InDelta(t, 123.1, sumDataPoints.At(0).DoubleValue(), 0.01)
	assert.Equal(t, map[string]any{"key0": "value0"}, sumDataPoints.At(0).Attributes().AsRaw())
	// Second point
	assert.EqualValues(t, startTime, sumDataPoints.At(1).StartTimestamp())
	assert.EqualValues(t, endTime, sumDataPoints.At(1).Timestamp())
	assert.InDelta(t, 456.1, sumDataPoints.At(1).DoubleValue(), 0.01)
	assert.Equal(t, map[string]any{"key1": "value1"}, sumDataPoints.At(1).Attributes().AsRaw())

	// Check histogram metric
	metricHistogram := metrics.At(2)
	assert.Equal(t, "my_metric_histogram", metricHistogram.Name())
	assert.Equal(t, "My metric", metricHistogram.Description())
	assert.Equal(t, "ms", metricHistogram.Unit())
	assert.Equal(t, MetricTypeHistogram, metricHistogram.Type())
	dhd := metricHistogram.Histogram()
	assert.Equal(t, AggregationTemporalityDelta, dhd.AggregationTemporality())
	histogramDataPoints := dhd.DataPoints()
	assert.Equal(t, 2, histogramDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, histogramDataPoints.At(0).StartTimestamp())
	assert.EqualValues(t, endTime, histogramDataPoints.At(0).Timestamp())
	assert.Equal(t, []float64{1, 2}, histogramDataPoints.At(0).ExplicitBounds().AsRaw())
	assert.Equal(t, map[string]any{"key0": "value0"}, histogramDataPoints.At(0).Attributes().AsRaw())
	assert.Equal(t, []uint64{10, 15, 1}, histogramDataPoints.At(0).BucketCounts().AsRaw())
	// Second point
	assert.EqualValues(t, startTime, histogramDataPoints.At(1).StartTimestamp())
	assert.EqualValues(t, endTime, histogramDataPoints.At(1).Timestamp())
	assert.Equal(t, []float64{1}, histogramDataPoints.At(1).ExplicitBounds().AsRaw())
	assert.Equal(t, map[string]any{"key1": "value1"}, histogramDataPoints.At(1).Attributes().AsRaw())
	assert.Equal(t, []uint64{10, 1}, histogramDataPoints.At(1).BucketCounts().AsRaw())
}

func TestOtlpToFromInternalReadOnly(t *testing.T) {
	md := newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}, new(internal.State))
	// Test that nothing changed
	assert.EqualValues(t, &internal.MetricsData{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}, md.getOrig())
}

func TestOtlpToFromInternalGaugeMutating(t *testing.T) {
	newAttributes := map[string]any{"k": "v"}

	md := newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoGaugeMetric()},
					},
				},
			},
		},
	}, new(internal.State))
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_int")
	assert.Equal(t, "new_my_metric_int", metric.Name())
	metric.SetDescription("My new metric")
	assert.Equal(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.Equal(t, "1", metric.Unit())
	// Mutate DataPoints
	igd := metric.Gauge()
	assert.Equal(t, 2, igd.DataPoints().Len())
	gaugeDataPoints := metric.SetEmptyGauge().DataPoints()
	gaugeDataPoints.AppendEmpty()
	assert.Equal(t, 1, gaugeDataPoints.Len())
	gaugeDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, gaugeDataPoints.At(0).StartTimestamp())
	gaugeDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, gaugeDataPoints.At(0).Timestamp())
	gaugeDataPoints.At(0).SetDoubleValue(124.1)
	assert.InDelta(t, 124.1, gaugeDataPoints.At(0).DoubleValue(), 0.01)
	gaugeDataPoints.At(0).Attributes().Remove("key0")
	gaugeDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.Equal(t, newAttributes, gaugeDataPoints.At(0).Attributes().AsRaw())

	// Test that everything is updated.
	assert.EqualValues(t, &internal.MetricsData{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{
							{
								Name:        "new_my_metric_int",
								Description: "My new metric",
								Unit:        "1",
								Data: &internal.Metric_Gauge{
									Gauge: &internal.Gauge{
										DataPoints: []*internal.NumberDataPoint{
											{
												Attributes: []internal.KeyValue{
													{
														Key:   "k",
														Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "v"}},
													},
												},
												StartTimeUnixNano: startTime + 1,
												TimeUnixNano:      endTime + 1,
												Value: &internal.NumberDataPoint_AsDouble{
													AsDouble: 124.1,
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
		},
	}, md.getOrig())
}

func TestOtlpToFromInternalSumMutating(t *testing.T) {
	newAttributes := map[string]any{"k": "v"}

	md := newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoSumMetric()},
					},
				},
			},
		},
	}, new(internal.State))
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_double")
	assert.Equal(t, "new_my_metric_double", metric.Name())
	metric.SetDescription("My new metric")
	assert.Equal(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.Equal(t, "1", metric.Unit())
	// Mutate DataPoints
	dsd := metric.Sum()
	assert.Equal(t, 2, dsd.DataPoints().Len())
	metric.SetEmptySum().SetAggregationTemporality(AggregationTemporalityCumulative)
	doubleDataPoints := metric.Sum().DataPoints()
	doubleDataPoints.AppendEmpty()
	assert.Equal(t, 1, doubleDataPoints.Len())
	doubleDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, doubleDataPoints.At(0).StartTimestamp())
	doubleDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, doubleDataPoints.At(0).Timestamp())
	doubleDataPoints.At(0).SetDoubleValue(124.1)
	assert.InDelta(t, 124.1, doubleDataPoints.At(0).DoubleValue(), 0.01)
	doubleDataPoints.At(0).Attributes().Remove("key0")
	doubleDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.Equal(t, newAttributes, doubleDataPoints.At(0).Attributes().AsRaw())

	// Test that everything is updated.
	assert.EqualValues(t, &internal.MetricsData{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{
							{
								Name:        "new_my_metric_double",
								Description: "My new metric",
								Unit:        "1",
								Data: &internal.Metric_Sum{
									Sum: &internal.Sum{
										AggregationTemporality: internal.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
										DataPoints: []*internal.NumberDataPoint{
											{
												Attributes: []internal.KeyValue{
													{
														Key:   "k",
														Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "v"}},
													},
												},
												StartTimeUnixNano: startTime + 1,
												TimeUnixNano:      endTime + 1,
												Value: &internal.NumberDataPoint_AsDouble{
													AsDouble: 124.1,
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
		},
	}, md.getOrig())
}

func TestOtlpToFromInternalHistogramMutating(t *testing.T) {
	newAttributes := map[string]any{"k": "v"}

	md := newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}, new(internal.State))
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_histogram")
	assert.Equal(t, "new_my_metric_histogram", metric.Name())
	metric.SetDescription("My new metric")
	assert.Equal(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.Equal(t, "1", metric.Unit())
	// Mutate DataPoints
	dhd := metric.Histogram()
	assert.Equal(t, 2, dhd.DataPoints().Len())
	metric.SetEmptyHistogram().SetAggregationTemporality(AggregationTemporalityDelta)
	histogramDataPoints := metric.Histogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	assert.Equal(t, 1, histogramDataPoints.Len())
	histogramDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.At(0).StartTimestamp())
	histogramDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints.At(0).Timestamp())
	histogramDataPoints.At(0).Attributes().Remove("key0")
	histogramDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.Equal(t, newAttributes, histogramDataPoints.At(0).Attributes().AsRaw())
	histogramDataPoints.At(0).ExplicitBounds().FromRaw([]float64{1})
	assert.Equal(t, []float64{1}, histogramDataPoints.At(0).ExplicitBounds().AsRaw())
	histogramDataPoints.At(0).BucketCounts().FromRaw([]uint64{21, 32})
	// Test that everything is updated.
	assert.EqualValues(t, &internal.MetricsData{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{
							{
								Name:        "new_my_metric_histogram",
								Description: "My new metric",
								Unit:        "1",
								Data: &internal.Metric_Histogram{
									Histogram: &internal.Histogram{
										AggregationTemporality: internal.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
										DataPoints: []*internal.HistogramDataPoint{
											{
												Attributes: []internal.KeyValue{
													{
														Key:   "k",
														Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "v"}},
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
		},
	}, md.getOrig())
}

func TestOtlpToFromInternalExponentialHistogramMutating(t *testing.T) {
	newAttributes := map[string]any{"k": "v"}

	md := newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}, new(internal.State))
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_exponential_histogram")
	assert.Equal(t, "new_my_metric_exponential_histogram", metric.Name())
	metric.SetDescription("My new metric")
	assert.Equal(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.Equal(t, "1", metric.Unit())
	// Mutate DataPoints
	dhd := metric.Histogram()
	assert.Equal(t, 2, dhd.DataPoints().Len())
	metric.SetEmptyExponentialHistogram().SetAggregationTemporality(AggregationTemporalityDelta)
	histogramDataPoints := metric.ExponentialHistogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	assert.Equal(t, 1, histogramDataPoints.Len())
	histogramDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.At(0).StartTimestamp())
	histogramDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints.At(0).Timestamp())
	histogramDataPoints.At(0).Attributes().Remove("key0")
	histogramDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.Equal(t, newAttributes, histogramDataPoints.At(0).Attributes().AsRaw())
	// Test that everything is updated.
	assert.EqualValues(t, &internal.MetricsData{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{
							{
								Name:        "new_my_metric_exponential_histogram",
								Description: "My new metric",
								Unit:        "1",
								Data: &internal.Metric_ExponentialHistogram{
									ExponentialHistogram: &internal.ExponentialHistogram{
										AggregationTemporality: internal.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
										DataPoints: []*internal.ExponentialHistogramDataPoint{
											{
												Attributes: []internal.KeyValue{
													{
														Key:   "k",
														Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "v"}},
													},
												},
												StartTimeUnixNano: startTime + 1,
												TimeUnixNano:      endTime + 1,
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
	}, md.getOrig())
}

func TestMetricsCopyTo(t *testing.T) {
	md := generateTestMetrics()
	metricsCopy := NewMetrics()
	md.CopyTo(metricsCopy)
	assert.Equal(t, md, metricsCopy)
}

func TestReadOnlyMetricsInvalidUsage(t *testing.T) {
	metrics := NewMetrics()
	assert.False(t, metrics.IsReadOnly())
	res := metrics.ResourceMetrics().AppendEmpty().Resource()
	res.Attributes().PutStr("k1", "v1")
	metrics.MarkReadOnly()
	assert.True(t, metrics.IsReadOnly())
	assert.Panics(t, func() { res.Attributes().PutStr("k2", "v2") })
}

func BenchmarkOtlpToFromInternal_PassThrough(b *testing.B) {
	testutil.SkipMemoryBench(b)
	req := &internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}
	var state internal.State

	for b.Loop() {
		md := newMetrics(req, &state)
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func BenchmarkOtlpToFromInternal_Gauge_MutateOneLabel(b *testing.B) {
	testutil.SkipMemoryBench(b)
	req := &internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoGaugeMetric()},
					},
				},
			},
		},
	}
	var state internal.State

	for b.Loop() {
		md := newMetrics(req, &state)
		md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0).Attributes().
			PutStr("key0", "value2")
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func BenchmarkOtlpToFromInternal_Sum_MutateOneLabel(b *testing.B) {
	testutil.SkipMemoryBench(b)
	req := &internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoSumMetric()},
					},
				},
			},
		},
	}
	var state internal.State

	for b.Loop() {
		md := newMetrics(req, &state)
		md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Sum().DataPoints().At(0).Attributes().
			PutStr("key0", "value2")
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func BenchmarkOtlpToFromInternal_HistogramPoints_MutateOneLabel(b *testing.B) {
	testutil.SkipMemoryBench(b)
	req := &internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*internal.Metric{generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}
	var state internal.State

	for b.Loop() {
		md := newMetrics(req, &state)
		md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Histogram().DataPoints().At(0).Attributes().
			PutStr("key0", "value2")
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func generateTestProtoResource() internal.Resource {
	return internal.Resource{
		Attributes: []internal.KeyValue{
			{
				Key:   "string",
				Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "string-resource"}},
			},
		},
	}
}

func generateTestProtoInstrumentationScope() internal.InstrumentationScope {
	return internal.InstrumentationScope{
		Name:    "test",
		Version: "",
	}
}

func generateTestProtoGaugeMetric() *internal.Metric {
	return &internal.Metric{
		Name:        "my_metric_int",
		Description: "My metric",
		Unit:        "ms",
		Data: &internal.Metric_Gauge{
			Gauge: &internal.Gauge{
				DataPoints: []*internal.NumberDataPoint{
					{
						Attributes: []internal.KeyValue{
							{
								Key:   "key0",
								Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "value0"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &internal.NumberDataPoint_AsDouble{
							AsDouble: 123.1,
						},
					},
					{
						Attributes: []internal.KeyValue{
							{
								Key:   "key1",
								Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "value1"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &internal.NumberDataPoint_AsDouble{
							AsDouble: 456.1,
						},
					},
				},
			},
		},
	}
}

func generateTestProtoSumMetric() *internal.Metric {
	return &internal.Metric{
		Name:        "my_metric_double",
		Description: "My metric",
		Unit:        "ms",
		Data: &internal.Metric_Sum{
			Sum: &internal.Sum{
				AggregationTemporality: internal.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				DataPoints: []*internal.NumberDataPoint{
					{
						Attributes: []internal.KeyValue{
							{
								Key:   "key0",
								Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "value0"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &internal.NumberDataPoint_AsDouble{
							AsDouble: 123.1,
						},
					},
					{
						Attributes: []internal.KeyValue{
							{
								Key:   "key1",
								Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "value1"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &internal.NumberDataPoint_AsDouble{
							AsDouble: 456.1,
						},
					},
				},
			},
		},
	}
}

func generateTestProtoHistogramMetric() *internal.Metric {
	return &internal.Metric{
		Name:        "my_metric_histogram",
		Description: "My metric",
		Unit:        "ms",
		Data: &internal.Metric_Histogram{
			Histogram: &internal.Histogram{
				AggregationTemporality: internal.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				DataPoints: []*internal.HistogramDataPoint{
					{
						Attributes: []internal.KeyValue{
							{
								Key:   "key0",
								Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "value0"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						BucketCounts:      []uint64{10, 15, 1},
						ExplicitBounds:    []float64{1, 2},
					},
					{
						Attributes: []internal.KeyValue{
							{
								Key:   "key1",
								Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "value1"}},
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

func generateMetricsEmptyResource() Metrics {
	return newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{{}},
	}, new(internal.State))
}

func generateMetricsEmptyInstrumentation() Metrics {
	return newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				ScopeMetrics: []*internal.ScopeMetrics{{}},
			},
		},
	}, new(internal.State))
}

func generateMetricsEmptyMetrics() Metrics {
	return newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Metrics: []*internal.Metric{{}},
					},
				},
			},
		},
	}, new(internal.State))
}

func generateMetricsEmptyDataPoints() Metrics {
	return newMetrics(&internal.ExportMetricsServiceRequest{
		ResourceMetrics: []*internal.ResourceMetrics{
			{
				ScopeMetrics: []*internal.ScopeMetrics{
					{
						Metrics: []*internal.Metric{
							{
								Data: &internal.Metric_Gauge{
									Gauge: &internal.Gauge{
										DataPoints: []*internal.NumberDataPoint{
											{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, new(internal.State))
}

func BenchmarkMetricsUsage(b *testing.B) {
	md := generateTestMetrics()

	ts := pcommon.NewTimestampFromTime(time.Now())

	b.ReportAllocs()

	for b.Loop() {
		for i := 0; i < md.ResourceMetrics().Len(); i++ {
			rm := md.ResourceMetrics().At(i)
			res := rm.Resource()
			res.Attributes().PutStr("foo", "bar")
			v, ok := res.Attributes().Get("foo")
			assert.True(b, ok)
			assert.Equal(b, "bar", v.Str())
			v.SetStr("new-bar")
			assert.Equal(b, "new-bar", v.Str())
			res.Attributes().Remove("foo")
			for j := 0; j < rm.ScopeMetrics().Len(); j++ {
				sm := rm.ScopeMetrics().At(j)
				for k := 0; k < sm.Metrics().Len(); k++ {
					m := sm.Metrics().At(k)
					m.SetName("new_metric_name")
					assert.Equal(b, "new_metric_name", m.Name())
					// Only process Sum metrics to avoid nil pointer dereference
					if m.Type() == MetricTypeSum {
						assert.Equal(b, MetricTypeSum, m.Type())
						m.Sum().SetAggregationTemporality(AggregationTemporalityCumulative)
						assert.Equal(b, AggregationTemporalityCumulative, m.Sum().AggregationTemporality())
						m.Sum().SetIsMonotonic(true)
						assert.True(b, m.Sum().IsMonotonic())
						for l := 0; l < m.Sum().DataPoints().Len(); l++ {
							dp := m.Sum().DataPoints().At(l)
							dp.SetIntValue(123)
							assert.Equal(b, int64(123), dp.IntValue())
							assert.Equal(b, NumberDataPointValueTypeInt, dp.ValueType())
							dp.SetStartTimestamp(ts)
							assert.Equal(b, ts, dp.StartTimestamp())
						}
						dp := m.Sum().DataPoints().AppendEmpty()
						dp.Attributes().PutStr("foo", "bar")
						dp.SetDoubleValue(123)
						dp.SetStartTimestamp(ts)
						dp.SetTimestamp(ts)
						m.Sum().DataPoints().RemoveIf(func(dp NumberDataPoint) bool {
							_, ok := dp.Attributes().Get("foo")
							return ok
						})
					}
				}
			}
		}
	}
}

func BenchmarkMetricsMarshalJSON(b *testing.B) {
	md := generateTestMetrics()
	encoder := &JSONMarshaler{}

	b.ReportAllocs()

	for b.Loop() {
		jsonBuf, err := encoder.MarshalMetrics(md)
		require.NoError(b, err)
		require.NotNil(b, jsonBuf)
	}
}
