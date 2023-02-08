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

package pmetric

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	startTime = uint64(12578940000000012345)
	endTime   = uint64(12578940000000054321)
)

func TestResourceMetricsWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceMetrics as pdata struct.
	metrics := NewMetrics()
	fillTestResourceMetricsSlice(metrics.ResourceMetrics())

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(metrics.getOrig())
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage emptypb.Empty
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRM otlpcollectormetrics.ExportMetricsServiceRequest
	err = gogoproto.Unmarshal(wire2, &gogoprotoRM)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.True(t, assert.EqualValues(t, metrics.getOrig(), &gogoprotoRM))
}

func TestMetricCount(t *testing.T) {
	md := NewMetrics()
	assert.EqualValues(t, 0, md.MetricCount())

	rms := md.ResourceMetrics()
	rms.EnsureCapacity(3)
	rm := rms.AppendEmpty()
	assert.EqualValues(t, 0, md.MetricCount())

	ilm := rm.ScopeMetrics().AppendEmpty()
	assert.EqualValues(t, 0, md.MetricCount())

	ilm.Metrics().AppendEmpty()
	assert.EqualValues(t, 1, md.MetricCount())

	rms.AppendEmpty().ScopeMetrics().AppendEmpty()
	ilmm := rms.AppendEmpty().ScopeMetrics().AppendEmpty().Metrics()
	ilmm.EnsureCapacity(5)
	for i := 0; i < 5; i++ {
		ilmm.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.EqualValues(t, 6, md.MetricCount())
}

func TestMetricCountWithEmpty(t *testing.T) {
	assert.EqualValues(t, 0, generateMetricsEmptyResource().MetricCount())
	assert.EqualValues(t, 0, generateMetricsEmptyInstrumentation().MetricCount())
	assert.EqualValues(t, 1, generateMetricsEmptyMetrics().MetricCount())
}

func TestMetricAndDataPointCount(t *testing.T) {
	md := NewMetrics()
	dps := md.DataPointCount()
	assert.EqualValues(t, 0, dps)

	rms := md.ResourceMetrics()
	rms.AppendEmpty()
	dps = md.DataPointCount()
	assert.EqualValues(t, 0, dps)

	ilms := md.ResourceMetrics().At(0).ScopeMetrics()
	ilms.AppendEmpty()
	dps = md.DataPointCount()
	assert.EqualValues(t, 0, dps)

	ilms.At(0).Metrics().AppendEmpty()
	dps = md.DataPointCount()
	assert.EqualValues(t, 0, dps)
	intSum := ilms.At(0).Metrics().At(0).SetEmptySum()
	intSum.DataPoints().AppendEmpty()
	intSum.DataPoints().AppendEmpty()
	intSum.DataPoints().AppendEmpty()
	assert.EqualValues(t, 3, md.DataPointCount())

	md = NewMetrics()
	rms = md.ResourceMetrics()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	rms.AppendEmpty().ScopeMetrics().AppendEmpty()
	rms.AppendEmpty().ScopeMetrics().AppendEmpty()
	ilms = rms.At(2).ScopeMetrics()
	ilm := ilms.At(0).Metrics()
	for i := 0; i < 5; i++ {
		ilm.AppendEmpty()
	}
	assert.EqualValues(t, 0, md.DataPointCount())

	ilm.At(0).SetEmptyGauge().DataPoints().AppendEmpty()
	assert.EqualValues(t, 1, md.DataPointCount())

	ilm.At(1).SetEmptySum().DataPoints().AppendEmpty()
	assert.EqualValues(t, 2, md.DataPointCount())

	ilm.At(2).SetEmptyHistogram().DataPoints().AppendEmpty()
	assert.EqualValues(t, 3, md.DataPointCount())

	ilm.At(3).SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
	assert.EqualValues(t, 4, md.DataPointCount())

	ilm.At(4).SetEmptySummary().DataPoints().AppendEmpty()
	assert.EqualValues(t, 5, md.DataPointCount())
}

func TestDataPointCountWithEmpty(t *testing.T) {
	assert.EqualValues(t, 0, generateMetricsEmptyResource().DataPointCount())
	assert.EqualValues(t, 0, generateMetricsEmptyInstrumentation().DataPointCount())
	assert.EqualValues(t, 0, generateMetricsEmptyMetrics().DataPointCount())
	assert.EqualValues(t, 1, generateMetricsEmptyDataPoints().DataPointCount())
}

func TestDataPointCountWithNilDataPoints(t *testing.T) {
	metrics := NewMetrics()
	ilm := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	ilm.Metrics().AppendEmpty().SetEmptyGauge()
	ilm.Metrics().AppendEmpty().SetEmptySum()
	ilm.Metrics().AppendEmpty().SetEmptyHistogram()
	ilm.Metrics().AppendEmpty().SetEmptyExponentialHistogram()
	ilm.Metrics().AppendEmpty().SetEmptySummary()
	assert.EqualValues(t, 0, metrics.DataPointCount())
}

func TestHistogramWithNilSum(t *testing.T) {
	metrics := NewMetrics()
	ilm := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	histo := ilm.Metrics().AppendEmpty()
	histogramDataPoints := histo.SetEmptyHistogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	dest := ilm.Metrics().AppendEmpty()
	histo.CopyTo(dest)
	assert.EqualValues(t, histo, dest)
}

func TestHistogramWithValidSum(t *testing.T) {
	metrics := NewMetrics()
	ilm := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	histo := ilm.Metrics().AppendEmpty()
	histogramDataPoints := histo.SetEmptyHistogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	histogramDataPoints.At(0).SetSum(10)
	dest := ilm.Metrics().AppendEmpty()
	histo.CopyTo(dest)
	assert.EqualValues(t, histo, dest)
}

func TestOtlpToInternalReadOnly(t *testing.T) {
	md := newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	})
	resourceMetrics := md.ResourceMetrics()
	assert.EqualValues(t, 1, resourceMetrics.Len())

	resourceMetric := resourceMetrics.At(0)
	assert.EqualValues(t, map[string]any{
		"string": "string-resource",
	}, resourceMetric.Resource().Attributes().AsRaw())
	metrics := resourceMetric.ScopeMetrics().At(0).Metrics()
	assert.EqualValues(t, 3, metrics.Len())

	// Check int64 metric
	metricInt := metrics.At(0)
	assert.EqualValues(t, "my_metric_int", metricInt.Name())
	assert.EqualValues(t, "My metric", metricInt.Description())
	assert.EqualValues(t, "ms", metricInt.Unit())
	assert.EqualValues(t, MetricTypeGauge, metricInt.Type())
	gaugeDataPoints := metricInt.Gauge().DataPoints()
	assert.EqualValues(t, 2, gaugeDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, gaugeDataPoints.At(0).StartTimestamp())
	assert.EqualValues(t, endTime, gaugeDataPoints.At(0).Timestamp())
	assert.EqualValues(t, 123.1, gaugeDataPoints.At(0).DoubleValue())
	assert.EqualValues(t, map[string]any{"key0": "value0"}, gaugeDataPoints.At(0).Attributes().AsRaw())
	// Second point
	assert.EqualValues(t, startTime, gaugeDataPoints.At(1).StartTimestamp())
	assert.EqualValues(t, endTime, gaugeDataPoints.At(1).Timestamp())
	assert.EqualValues(t, 456.1, gaugeDataPoints.At(1).DoubleValue())
	assert.EqualValues(t, map[string]any{"key1": "value1"}, gaugeDataPoints.At(1).Attributes().AsRaw())

	// Check double metric
	metricDouble := metrics.At(1)
	assert.EqualValues(t, "my_metric_double", metricDouble.Name())
	assert.EqualValues(t, "My metric", metricDouble.Description())
	assert.EqualValues(t, "ms", metricDouble.Unit())
	assert.EqualValues(t, MetricTypeSum, metricDouble.Type())
	dsd := metricDouble.Sum()
	assert.EqualValues(t, AggregationTemporalityCumulative, dsd.AggregationTemporality())
	sumDataPoints := dsd.DataPoints()
	assert.EqualValues(t, 2, sumDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, sumDataPoints.At(0).StartTimestamp())
	assert.EqualValues(t, endTime, sumDataPoints.At(0).Timestamp())
	assert.EqualValues(t, 123.1, sumDataPoints.At(0).DoubleValue())
	assert.EqualValues(t, map[string]any{"key0": "value0"}, sumDataPoints.At(0).Attributes().AsRaw())
	// Second point
	assert.EqualValues(t, startTime, sumDataPoints.At(1).StartTimestamp())
	assert.EqualValues(t, endTime, sumDataPoints.At(1).Timestamp())
	assert.EqualValues(t, 456.1, sumDataPoints.At(1).DoubleValue())
	assert.EqualValues(t, map[string]any{"key1": "value1"}, sumDataPoints.At(1).Attributes().AsRaw())

	// Check histogram metric
	metricHistogram := metrics.At(2)
	assert.EqualValues(t, "my_metric_histogram", metricHistogram.Name())
	assert.EqualValues(t, "My metric", metricHistogram.Description())
	assert.EqualValues(t, "ms", metricHistogram.Unit())
	assert.EqualValues(t, MetricTypeHistogram, metricHistogram.Type())
	dhd := metricHistogram.Histogram()
	assert.EqualValues(t, AggregationTemporalityDelta, dhd.AggregationTemporality())
	histogramDataPoints := dhd.DataPoints()
	assert.EqualValues(t, 2, histogramDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, histogramDataPoints.At(0).StartTimestamp())
	assert.EqualValues(t, endTime, histogramDataPoints.At(0).Timestamp())
	assert.EqualValues(t, []float64{1, 2}, histogramDataPoints.At(0).ExplicitBounds().AsRaw())
	assert.EqualValues(t, map[string]any{"key0": "value0"}, histogramDataPoints.At(0).Attributes().AsRaw())
	assert.EqualValues(t, []uint64{10, 15, 1}, histogramDataPoints.At(0).BucketCounts().AsRaw())
	// Second point
	assert.EqualValues(t, startTime, histogramDataPoints.At(1).StartTimestamp())
	assert.EqualValues(t, endTime, histogramDataPoints.At(1).Timestamp())
	assert.EqualValues(t, []float64{1}, histogramDataPoints.At(1).ExplicitBounds().AsRaw())
	assert.EqualValues(t, map[string]any{"key1": "value1"}, histogramDataPoints.At(1).Attributes().AsRaw())
	assert.EqualValues(t, []uint64{10, 1}, histogramDataPoints.At(1).BucketCounts().AsRaw())
}

func TestOtlpToFromInternalReadOnly(t *testing.T) {
	md := newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	})
	// Test that nothing changed
	assert.EqualValues(t, &otlpmetrics.MetricsData{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}, md.getOrig())
}

func TestOtlpToFromInternalGaugeMutating(t *testing.T) {
	newAttributes := map[string]any{"k": "v"}

	md := newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoGaugeMetric()},
					},
				},
			},
		},
	})
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_int")
	assert.EqualValues(t, "new_my_metric_int", metric.Name())
	metric.SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.EqualValues(t, "1", metric.Unit())
	// Mutate DataPoints
	igd := metric.Gauge()
	assert.EqualValues(t, 2, igd.DataPoints().Len())
	gaugeDataPoints := metric.SetEmptyGauge().DataPoints()
	gaugeDataPoints.AppendEmpty()
	assert.EqualValues(t, 1, gaugeDataPoints.Len())
	gaugeDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, gaugeDataPoints.At(0).StartTimestamp())
	gaugeDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, gaugeDataPoints.At(0).Timestamp())
	gaugeDataPoints.At(0).SetDoubleValue(124.1)
	assert.EqualValues(t, 124.1, gaugeDataPoints.At(0).DoubleValue())
	gaugeDataPoints.At(0).Attributes().Remove("key0")
	gaugeDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.EqualValues(t, newAttributes, gaugeDataPoints.At(0).Attributes().AsRaw())

	// Test that everything is updated.
	assert.EqualValues(t, &otlpmetrics.MetricsData{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{
							{
								Name:        "new_my_metric_int",
								Description: "My new metric",
								Unit:        "1",
								Data: &otlpmetrics.Metric_Gauge{
									Gauge: &otlpmetrics.Gauge{
										DataPoints: []*otlpmetrics.NumberDataPoint{
											{
												Attributes: []otlpcommon.KeyValue{
													{
														Key:   "k",
														Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}},
													},
												},
												StartTimeUnixNano: startTime + 1,
												TimeUnixNano:      endTime + 1,
												Value: &otlpmetrics.NumberDataPoint_AsDouble{
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

	md := newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoSumMetric()},
					},
				},
			},
		},
	})
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_double")
	assert.EqualValues(t, "new_my_metric_double", metric.Name())
	metric.SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.EqualValues(t, "1", metric.Unit())
	// Mutate DataPoints
	dsd := metric.Sum()
	assert.EqualValues(t, 2, dsd.DataPoints().Len())
	metric.SetEmptySum().SetAggregationTemporality(AggregationTemporalityCumulative)
	doubleDataPoints := metric.Sum().DataPoints()
	doubleDataPoints.AppendEmpty()
	assert.EqualValues(t, 1, doubleDataPoints.Len())
	doubleDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, doubleDataPoints.At(0).StartTimestamp())
	doubleDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, doubleDataPoints.At(0).Timestamp())
	doubleDataPoints.At(0).SetDoubleValue(124.1)
	assert.EqualValues(t, 124.1, doubleDataPoints.At(0).DoubleValue())
	doubleDataPoints.At(0).Attributes().Remove("key0")
	doubleDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.EqualValues(t, newAttributes, doubleDataPoints.At(0).Attributes().AsRaw())

	// Test that everything is updated.
	assert.EqualValues(t, &otlpmetrics.MetricsData{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{
							{
								Name:        "new_my_metric_double",
								Description: "My new metric",
								Unit:        "1",
								Data: &otlpmetrics.Metric_Sum{
									Sum: &otlpmetrics.Sum{
										AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
										DataPoints: []*otlpmetrics.NumberDataPoint{
											{
												Attributes: []otlpcommon.KeyValue{
													{
														Key:   "k",
														Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}},
													},
												},
												StartTimeUnixNano: startTime + 1,
												TimeUnixNano:      endTime + 1,
												Value: &otlpmetrics.NumberDataPoint_AsDouble{
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

	md := newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	})
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_histogram")
	assert.EqualValues(t, "new_my_metric_histogram", metric.Name())
	metric.SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.EqualValues(t, "1", metric.Unit())
	// Mutate DataPoints
	dhd := metric.Histogram()
	assert.EqualValues(t, 2, dhd.DataPoints().Len())
	metric.SetEmptyHistogram().SetAggregationTemporality(AggregationTemporalityDelta)
	histogramDataPoints := metric.Histogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	assert.EqualValues(t, 1, histogramDataPoints.Len())
	histogramDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.At(0).StartTimestamp())
	histogramDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints.At(0).Timestamp())
	histogramDataPoints.At(0).Attributes().Remove("key0")
	histogramDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.EqualValues(t, newAttributes, histogramDataPoints.At(0).Attributes().AsRaw())
	histogramDataPoints.At(0).ExplicitBounds().FromRaw([]float64{1})
	assert.EqualValues(t, []float64{1}, histogramDataPoints.At(0).ExplicitBounds().AsRaw())
	histogramDataPoints.At(0).BucketCounts().FromRaw([]uint64{21, 32})
	// Test that everything is updated.
	assert.EqualValues(t, &otlpmetrics.MetricsData{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{
							{
								Name:        "new_my_metric_histogram",
								Description: "My new metric",
								Unit:        "1",
								Data: &otlpmetrics.Metric_Histogram{
									Histogram: &otlpmetrics.Histogram{
										AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
										DataPoints: []*otlpmetrics.HistogramDataPoint{
											{
												Attributes: []otlpcommon.KeyValue{
													{
														Key:   "k",
														Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}},
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

	md := newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	})
	resourceMetrics := md.ResourceMetrics()
	metric := resourceMetrics.At(0).ScopeMetrics().At(0).Metrics().At(0)
	// Mutate MetricDescriptor
	metric.SetName("new_my_metric_exponential_histogram")
	assert.EqualValues(t, "new_my_metric_exponential_histogram", metric.Name())
	metric.SetDescription("My new metric")
	assert.EqualValues(t, "My new metric", metric.Description())
	metric.SetUnit("1")
	assert.EqualValues(t, "1", metric.Unit())
	// Mutate DataPoints
	dhd := metric.Histogram()
	assert.EqualValues(t, 2, dhd.DataPoints().Len())
	metric.SetEmptyExponentialHistogram().SetAggregationTemporality(AggregationTemporalityDelta)
	histogramDataPoints := metric.ExponentialHistogram().DataPoints()
	histogramDataPoints.AppendEmpty()
	assert.EqualValues(t, 1, histogramDataPoints.Len())
	histogramDataPoints.At(0).SetStartTimestamp(pcommon.Timestamp(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.At(0).StartTimestamp())
	histogramDataPoints.At(0).SetTimestamp(pcommon.Timestamp(endTime + 1))
	assert.EqualValues(t, endTime+1, histogramDataPoints.At(0).Timestamp())
	histogramDataPoints.At(0).Attributes().Remove("key0")
	histogramDataPoints.At(0).Attributes().PutStr("k", "v")
	assert.EqualValues(t, newAttributes, histogramDataPoints.At(0).Attributes().AsRaw())
	// Test that everything is updated.
	assert.EqualValues(t, &otlpmetrics.MetricsData{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope: generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{
							{
								Name:        "new_my_metric_exponential_histogram",
								Description: "My new metric",
								Unit:        "1",
								Data: &otlpmetrics.Metric_ExponentialHistogram{
									ExponentialHistogram: &otlpmetrics.ExponentialHistogram{
										AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
										DataPoints: []*otlpmetrics.ExponentialHistogramDataPoint{
											{
												Attributes: []otlpcommon.KeyValue{
													{
														Key:   "k",
														Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}},
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
	metrics := NewMetrics()
	fillTestResourceMetricsSlice(metrics.ResourceMetrics())
	metricsCopy := NewMetrics()
	metrics.CopyTo(metricsCopy)
	assert.EqualValues(t, metrics, metricsCopy)
}

func BenchmarkOtlpToFromInternal_PassThrough(b *testing.B) {
	req := &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoGaugeMetric(), generateTestProtoSumMetric(), generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := newMetrics(req)
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func BenchmarkOtlpToFromInternal_Gauge_MutateOneLabel(b *testing.B) {
	req := &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoGaugeMetric()},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := newMetrics(req)
		md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0).Attributes().
			PutStr("key0", "value2")
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func BenchmarkOtlpToFromInternal_Sum_MutateOneLabel(b *testing.B) {
	req := &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoSumMetric()},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := newMetrics(req)
		md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Sum().DataPoints().At(0).Attributes().
			PutStr("key0", "value2")
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func BenchmarkOtlpToFromInternal_HistogramPoints_MutateOneLabel(b *testing.B) {
	req := &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateTestProtoResource(),
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Scope:   generateTestProtoInstrumentationScope(),
						Metrics: []*otlpmetrics.Metric{generateTestProtoHistogramMetric()},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		md := newMetrics(req)
		md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Histogram().DataPoints().At(0).Attributes().
			PutStr("key0", "value2")
		newReq := md.getOrig()
		if len(req.ResourceMetrics) != len(newReq.ResourceMetrics) {
			b.Fail()
		}
	}
}

func generateTestProtoResource() otlpresource.Resource {
	return otlpresource.Resource{
		Attributes: []otlpcommon.KeyValue{
			{
				Key:   "string",
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "string-resource"}},
			},
		},
	}
}

func generateTestProtoInstrumentationScope() otlpcommon.InstrumentationScope {
	return otlpcommon.InstrumentationScope{
		Name:    "test",
		Version: "",
	}
}

func generateTestProtoGaugeMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		Name:        "my_metric_int",
		Description: "My metric",
		Unit:        "ms",
		Data: &otlpmetrics.Metric_Gauge{
			Gauge: &otlpmetrics.Gauge{
				DataPoints: []*otlpmetrics.NumberDataPoint{
					{
						Attributes: []otlpcommon.KeyValue{
							{
								Key:   "key0",
								Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value0"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &otlpmetrics.NumberDataPoint_AsDouble{
							AsDouble: 123.1,
						},
					},
					{
						Attributes: []otlpcommon.KeyValue{
							{
								Key:   "key1",
								Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &otlpmetrics.NumberDataPoint_AsDouble{
							AsDouble: 456.1,
						},
					},
				},
			},
		},
	}
}
func generateTestProtoSumMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		Name:        "my_metric_double",
		Description: "My metric",
		Unit:        "ms",
		Data: &otlpmetrics.Metric_Sum{
			Sum: &otlpmetrics.Sum{
				AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				DataPoints: []*otlpmetrics.NumberDataPoint{
					{
						Attributes: []otlpcommon.KeyValue{
							{
								Key:   "key0",
								Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value0"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &otlpmetrics.NumberDataPoint_AsDouble{
							AsDouble: 123.1,
						},
					},
					{
						Attributes: []otlpcommon.KeyValue{
							{
								Key:   "key1",
								Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						Value: &otlpmetrics.NumberDataPoint_AsDouble{
							AsDouble: 456.1,
						},
					},
				},
			},
		},
	}
}

func generateTestProtoHistogramMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		Name:        "my_metric_histogram",
		Description: "My metric",
		Unit:        "ms",
		Data: &otlpmetrics.Metric_Histogram{
			Histogram: &otlpmetrics.Histogram{
				AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				DataPoints: []*otlpmetrics.HistogramDataPoint{
					{
						Attributes: []otlpcommon.KeyValue{
							{
								Key:   "key0",
								Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value0"}},
							},
						},
						StartTimeUnixNano: startTime,
						TimeUnixNano:      endTime,
						BucketCounts:      []uint64{10, 15, 1},
						ExplicitBounds:    []float64{1, 2},
					},
					{
						Attributes: []otlpcommon.KeyValue{
							{
								Key:   "key1",
								Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
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
	return newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{{}},
	})
}

func generateMetricsEmptyInstrumentation() Metrics {
	return newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{{}},
			},
		},
	})
}

func generateMetricsEmptyMetrics() Metrics {
	return newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Metrics: []*otlpmetrics.Metric{{}},
					},
				},
			},
		},
	})
}

func generateMetricsEmptyDataPoints() Metrics {
	return newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				ScopeMetrics: []*otlpmetrics.ScopeMetrics{
					{
						Metrics: []*otlpmetrics.Metric{
							{
								Data: &otlpmetrics.Metric_Gauge{
									Gauge: &otlpmetrics.Gauge{
										DataPoints: []*otlpmetrics.NumberDataPoint{
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
	})
}
