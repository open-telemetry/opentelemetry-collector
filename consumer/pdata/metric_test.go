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

package pdata

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
)

const (
	startTime = uint64(12578940000000012345)
	endTime   = uint64(12578940000000054321)
)

func TestCopyData(t *testing.T) {
	tests := []struct {
		name string
		src  *otlpmetrics.Metric
	}{
		{
			name: "IntGauge",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_IntGauge{
					IntGauge: &otlpmetrics.IntGauge{},
				},
			},
		},
		{
			name: "DoubleGauge",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_DoubleGauge{
					DoubleGauge: &otlpmetrics.DoubleGauge{},
				},
			},
		},
		{
			name: "IntSum",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_IntSum{
					IntSum: &otlpmetrics.IntSum{},
				},
			},
		},
		{
			name: "DoubleSum",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_DoubleSum{
					DoubleSum: &otlpmetrics.DoubleSum{},
				},
			},
		},
		{
			name: "IntHistogram",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_IntHistogram{
					IntHistogram: &otlpmetrics.IntHistogram{},
				},
			},
		},
		{
			name: "DoubleHistogram",
			src: &otlpmetrics.Metric{
				Data: &otlpmetrics.Metric_DoubleHistogram{
					DoubleHistogram: &otlpmetrics.DoubleHistogram{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dest := &otlpmetrics.Metric{}
			assert.Nil(t, dest.Data)
			assert.NotNil(t, test.src.Data)
			copyData(test.src, dest)
			assert.EqualValues(t, test.src, dest)
		})
	}
}

func TestDataType(t *testing.T) {
	m := NewMetric()
	assert.Equal(t, MetricDataTypeNone, m.DataType())
	m.SetDataType(MetricDataTypeIntGauge)
	assert.Equal(t, MetricDataTypeIntGauge, m.DataType())
	m.SetDataType(MetricDataTypeDoubleGauge)
	assert.Equal(t, MetricDataTypeDoubleGauge, m.DataType())
	m.SetDataType(MetricDataTypeIntSum)
	assert.Equal(t, MetricDataTypeIntSum, m.DataType())
	m.SetDataType(MetricDataTypeDoubleSum)
	assert.Equal(t, MetricDataTypeDoubleSum, m.DataType())
	m.SetDataType(MetricDataTypeIntHistogram)
	assert.Equal(t, MetricDataTypeIntHistogram, m.DataType())
	m.SetDataType(MetricDataTypeDoubleHistogram)
	assert.Equal(t, MetricDataTypeDoubleHistogram, m.DataType())
	m.SetDataType(MetricDataTypeDoubleSummary)
	assert.Equal(t, MetricDataTypeDoubleSummary, m.DataType())
}

func TestResourceMetricsWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceMetrics as pdata struct.
	pdataRM := generateTestResourceMetrics()

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(pdataRM.orig)
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
	var gogoprotoRM otlpmetrics.ResourceMetrics
	err = gogoproto.Unmarshal(wire2, &gogoprotoRM)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.True(t, assert.EqualValues(t, pdataRM.orig, &gogoprotoRM))
}

func TestMetricCount(t *testing.T) {
	md := NewMetrics()
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
	md := NewMetrics()
	assert.Equal(t, 0, md.Size())
	rms := md.ResourceMetrics()
	rms.Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().Resize(1)
	rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().Resize(1)
	metric := rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
	metric.SetDataType(MetricDataTypeDoubleHistogram)
	doubleHistogram := metric.DoubleHistogram()
	doubleHistogram.DataPoints().Resize(1)
	doubleHistogram.DataPoints().At(0).SetCount(123)
	doubleHistogram.DataPoints().At(0).SetSum(123)
	otlp := MetricsToOtlp(md)
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

func TestMetricsSizeWithNil(t *testing.T) {
	assert.Equal(t, 0, MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{nil}).Size())
}

func TestMetricCountWithEmpty(t *testing.T) {
	assert.EqualValues(t, 0, MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{{}}).MetricCount())
	assert.EqualValues(t, 0, MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{{}},
		},
	}).MetricCount())
	assert.EqualValues(t, 1, MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{{}},
				},
			},
		},
	}).MetricCount())
}

func TestMetricAndDataPointCount(t *testing.T) {
	md := NewMetrics()
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
	ilms.At(0).Metrics().At(0).SetDataType(MetricDataTypeIntSum)
	intSum := ilms.At(0).Metrics().At(0).IntSum()
	intSum.DataPoints().Resize(3)
	_, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 3, dps)

	md = NewMetrics()
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
	ilms.At(0).Metrics().At(1).SetDataType(MetricDataTypeDoubleGauge)
	doubleGauge := ilms.At(0).Metrics().At(1).DoubleGauge()
	doubleGauge.DataPoints().Resize(1)
	ilms.At(0).Metrics().At(3).SetDataType(MetricDataTypeIntHistogram)
	intHistogram := ilms.At(0).Metrics().At(3).IntHistogram()
	intHistogram.DataPoints().Resize(3)
	ms, dps = md.MetricAndDataPointCount()
	assert.EqualValues(t, 6, ms)
	assert.EqualValues(t, 4, dps)
}

func TestMetricAndDataPointCountWithEmpty(t *testing.T) {
	ms, dps := MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{{}}).MetricAndDataPointCount()
	assert.EqualValues(t, 0, ms)
	assert.EqualValues(t, 0, dps)

	ms, dps = MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{{}},
		},
	}).MetricAndDataPointCount()
	assert.EqualValues(t, 0, ms)
	assert.EqualValues(t, 0, dps)

	ms, dps = MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{{}},
				},
			},
		},
	}).MetricAndDataPointCount()
	assert.EqualValues(t, 1, ms)
	assert.EqualValues(t, 0, dps)

	ms, dps = MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{{
						Data: &otlpmetrics.Metric_DoubleGauge{
							DoubleGauge: &otlpmetrics.DoubleGauge{
								DataPoints: []*otlpmetrics.DoubleDataPoint{
									{},
								},
							},
						},
					}},
				},
			},
		},
	}).MetricAndDataPointCount()
	assert.EqualValues(t, 1, ms)
	assert.EqualValues(t, 1, dps)

}

func TestMetricAndDataPointCountWithNilDataPoints(t *testing.T) {
	metrics := NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	rm := metrics.ResourceMetrics().At(0)
	rm.InstrumentationLibraryMetrics().Resize(1)
	ilm := rm.InstrumentationLibraryMetrics().At(0)
	intGauge := NewMetric()
	ilm.Metrics().Append(intGauge)
	intGauge.SetDataType(MetricDataTypeIntGauge)
	doubleGauge := NewMetric()
	ilm.Metrics().Append(doubleGauge)
	doubleGauge.SetDataType(MetricDataTypeDoubleGauge)
	intHistogram := NewMetric()
	ilm.Metrics().Append(intHistogram)
	intHistogram.SetDataType(MetricDataTypeIntHistogram)
	doubleHistogram := NewMetric()
	ilm.Metrics().Append(doubleHistogram)
	doubleHistogram.SetDataType(MetricDataTypeDoubleHistogram)
	intSum := NewMetric()
	ilm.Metrics().Append(intSum)
	intSum.SetDataType(MetricDataTypeIntSum)
	doubleSum := NewMetric()
	ilm.Metrics().Append(doubleSum)
	doubleSum.SetDataType(MetricDataTypeDoubleSum)

	ms, dps := metrics.MetricAndDataPointCount()

	assert.EqualValues(t, 6, ms)
	assert.EqualValues(t, 0, dps)
}

func TestOtlpToInternalReadOnly(t *testing.T) {
	metricData := MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
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
	assert.EqualValues(t, NewAttributeMap().InitFromMap(map[string]AttributeValue{
		"string": NewAttributeValueString("string-resource"),
	}), resourceMetric.Resource().Attributes())
	metrics := resourceMetric.InstrumentationLibraryMetrics().At(0).Metrics()
	assert.EqualValues(t, 3, metrics.Len())

	// Check int64 metric
	metricInt := metrics.At(0)
	assert.EqualValues(t, "my_metric_int", metricInt.Name())
	assert.EqualValues(t, "My metric", metricInt.Description())
	assert.EqualValues(t, "ms", metricInt.Unit())
	assert.EqualValues(t, MetricDataTypeIntGauge, metricInt.DataType())
	int64DataPoints := metricInt.IntGauge().DataPoints()
	assert.EqualValues(t, 2, int64DataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, int64DataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, int64DataPoints.At(0).Timestamp())
	assert.EqualValues(t, 123, int64DataPoints.At(0).Value())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), int64DataPoints.At(0).LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, int64DataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, int64DataPoints.At(1).Timestamp())
	assert.EqualValues(t, 456, int64DataPoints.At(1).Value())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), int64DataPoints.At(1).LabelsMap())

	// Check double metric
	metricDouble := metrics.At(1)
	assert.EqualValues(t, "my_metric_double", metricDouble.Name())
	assert.EqualValues(t, "My metric", metricDouble.Description())
	assert.EqualValues(t, "ms", metricDouble.Unit())
	assert.EqualValues(t, MetricDataTypeDoubleSum, metricDouble.DataType())
	dsd := metricDouble.DoubleSum()
	assert.EqualValues(t, AggregationTemporalityCumulative, dsd.AggregationTemporality())
	doubleDataPoints := dsd.DataPoints()
	assert.EqualValues(t, 2, doubleDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, doubleDataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints.At(0).Timestamp())
	assert.EqualValues(t, 123.1, doubleDataPoints.At(0).Value())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), doubleDataPoints.At(0).LabelsMap())
	// Second point
	assert.EqualValues(t, startTime, doubleDataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, doubleDataPoints.At(1).Timestamp())
	assert.EqualValues(t, 456.1, doubleDataPoints.At(1).Value())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), doubleDataPoints.At(1).LabelsMap())

	// Check histogram metric
	metricHistogram := metrics.At(2)
	assert.EqualValues(t, "my_metric_histogram", metricHistogram.Name())
	assert.EqualValues(t, "My metric", metricHistogram.Description())
	assert.EqualValues(t, "ms", metricHistogram.Unit())
	assert.EqualValues(t, MetricDataTypeDoubleHistogram, metricHistogram.DataType())
	dhd := metricHistogram.DoubleHistogram()
	assert.EqualValues(t, AggregationTemporalityDelta, dhd.AggregationTemporality())
	histogramDataPoints := dhd.DataPoints()
	assert.EqualValues(t, 2, histogramDataPoints.Len())
	// First point
	assert.EqualValues(t, startTime, histogramDataPoints.At(0).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.At(0).Timestamp())
	assert.EqualValues(t, []float64{1, 2}, histogramDataPoints.At(0).ExplicitBounds())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"key0": "value0"}), histogramDataPoints.At(0).LabelsMap())
	assert.EqualValues(t, []uint64{10, 15, 1}, histogramDataPoints.At(0).BucketCounts())
	// Second point
	assert.EqualValues(t, startTime, histogramDataPoints.At(1).StartTime())
	assert.EqualValues(t, endTime, histogramDataPoints.At(1).Timestamp())
	assert.EqualValues(t, []float64{1}, histogramDataPoints.At(1).ExplicitBounds())
	assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{"key1": "value1"}), histogramDataPoints.At(1).LabelsMap())
	assert.EqualValues(t, []uint64{10, 1}, histogramDataPoints.At(1).BucketCounts())
}

func TestOtlpToFromInternalReadOnly(t *testing.T) {
	metricData := MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
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
	}, MetricsToOtlp(metricData))
}

func TestOtlpToFromInternalIntGaugeMutating(t *testing.T) {
	newLabels := NewStringMap().InitFromMap(map[string]string{"k": "v"})

	metricData := MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
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
	igd := metric.IntGauge()
	assert.EqualValues(t, 2, igd.DataPoints().Len())
	igd.DataPoints().Resize(1)
	assert.EqualValues(t, 1, igd.DataPoints().Len())
	int64DataPoints := igd.DataPoints()
	int64DataPoints.At(0).SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, int64DataPoints.At(0).StartTime())
	int64DataPoints.At(0).SetTimestamp(TimestampUnixNano(endTime + 1))
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
											Labels: []otlpcommon.StringKeyValue{
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
	}, MetricsToOtlp(metricData))
}

func TestOtlpToFromInternalDoubleSumMutating(t *testing.T) {
	newLabels := NewStringMap().InitFromMap(map[string]string{"k": "v"})

	metricData := MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
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
	dsd := metric.DoubleSum()
	assert.EqualValues(t, 2, dsd.DataPoints().Len())
	dsd.DataPoints().Resize(1)
	assert.EqualValues(t, 1, dsd.DataPoints().Len())
	doubleDataPoints := dsd.DataPoints()
	doubleDataPoints.At(0).SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, doubleDataPoints.At(0).StartTime())
	doubleDataPoints.At(0).SetTimestamp(TimestampUnixNano(endTime + 1))
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
											Labels: []otlpcommon.StringKeyValue{
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
	}, MetricsToOtlp(metricData))
}

func TestOtlpToFromInternalHistogramMutating(t *testing.T) {
	newLabels := NewStringMap().InitFromMap(map[string]string{"k": "v"})

	metricData := MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
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
	dhd := metric.DoubleHistogram()
	assert.EqualValues(t, 2, dhd.DataPoints().Len())
	dhd.DataPoints().Resize(1)
	assert.EqualValues(t, 1, dhd.DataPoints().Len())
	histogramDataPoints := dhd.DataPoints()
	histogramDataPoints.At(0).SetStartTime(TimestampUnixNano(startTime + 1))
	assert.EqualValues(t, startTime+1, histogramDataPoints.At(0).StartTime())
	histogramDataPoints.At(0).SetTimestamp(TimestampUnixNano(endTime + 1))
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
											Labels: []otlpcommon.StringKeyValue{
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
	}, MetricsToOtlp(metricData))
}

func TestMetricsToFromOtlpProtoBytes(t *testing.T) {
	send := NewMetrics()
	fillTestResourceMetricsSlice(send.ResourceMetrics())
	bytes, err := send.ToOtlpProtoBytes()
	assert.NoError(t, err)

	recv := NewMetrics()
	err = recv.FromOtlpProtoBytes(bytes)
	assert.NoError(t, err)
	assert.EqualValues(t, send, recv)
}

func TestMetricsFromInvalidOtlpProtoBytes(t *testing.T) {
	err := NewMetrics().FromOtlpProtoBytes([]byte{0xFF})
	assert.EqualError(t, err, "unexpected EOF")
}

func TestMetricsClone(t *testing.T) {
	metrics := NewMetrics()
	fillTestResourceMetricsSlice(metrics.ResourceMetrics())
	assert.EqualValues(t, metrics, metrics.Clone())
}

func BenchmarkMetricsClone(b *testing.B) {
	metrics := NewMetrics()
	fillTestResourceMetricsSlice(metrics.ResourceMetrics())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		clone := metrics.Clone()
		if clone.ResourceMetrics().Len() != metrics.ResourceMetrics().Len() {
			b.Fail()
		}
	}
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
		md := MetricsFromOtlp(resourceMetricsList)
		MetricsToOtlp(md)
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
		md := MetricsFromOtlp(resourceMetricsList)
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntGauge().DataPoints().At(0).LabelsMap().Upsert("key0", "value2")
		MetricsToOtlp(md)
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
		md := MetricsFromOtlp(resourceMetricsList)
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleSum().DataPoints().At(0).LabelsMap().Upsert("key0", "value2")
		MetricsToOtlp(md)
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
		md := MetricsFromOtlp(resourceMetricsList)
		md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).DoubleHistogram().DataPoints().At(0).LabelsMap().Upsert("key0", "value2")
		MetricsToOtlp(md)
	}
}

func BenchmarkMetrics_ToOtlpProtoBytes_PassThrough(b *testing.B) {
	metrics := MetricsFromOtlp([]*otlpmetrics.ResourceMetrics{
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

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = metrics.ToOtlpProtoBytes()
	}
}

func BenchmarkMetricsToOtlp(b *testing.B) {
	traces := NewMetrics()
	fillTestResourceMetricsSlice(traces.ResourceMetrics())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := traces.ToOtlpProtoBytes()
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkMetricsFromOtlp(b *testing.B) {
	baseMetrics := NewMetrics()
	fillTestResourceMetricsSlice(baseMetrics.ResourceMetrics())
	buf, err := baseMetrics.ToOtlpProtoBytes()
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		traces := NewMetrics()
		require.NoError(b, traces.FromOtlpProtoBytes(buf))
		assert.Equal(b, baseMetrics.ResourceMetrics().Len(), traces.ResourceMetrics().Len())
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

func generateTestProtoInstrumentationLibrary() otlpcommon.InstrumentationLibrary {
	return otlpcommon.InstrumentationLibrary{
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
						Labels: []otlpcommon.StringKeyValue{
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
						Labels: []otlpcommon.StringKeyValue{
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
						Labels: []otlpcommon.StringKeyValue{
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
						Labels: []otlpcommon.StringKeyValue{
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
						Labels: []otlpcommon.StringKeyValue{
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
						Labels: []otlpcommon.StringKeyValue{
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
