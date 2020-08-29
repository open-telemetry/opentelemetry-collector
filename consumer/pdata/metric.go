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
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
)

type AggregationTemporality otlpmetrics.AggregationTemporality

const (
	AggregationTemporalityUnspecified = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED)
	AggregationTemporalityDelta       = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA)
	AggregationTemporalityCumulative  = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE)
)

func (at AggregationTemporality) String() string {
	return otlpmetrics.AggregationTemporality(at).String()
}

// Metrics is an opaque interface that allows transition to the new internal Metrics data, but also facilitate the
// transition to the new components especially for traces.
//
// Outside of the core repository the metrics pipeline cannot be converted to the new model since data.MetricData is
// part of the internal package.
//
// IMPORTANT: Do not try to convert to/from this manually, use the helper functions in the pdatautil instead.
type Metrics struct {
	InternalOpaque interface{}
}

// MetricDataType specifies the type of data in a Metric.
type MetricDataType int

const (
	MetricDataNone MetricDataType = iota
	MetricDataIntGauge
	MetricDataDoubleGauge
	MetricDataIntSum
	MetricDataDoubleSum
	MetricDataIntHistogram
	MetricDataDoubleHistogram
)

func (mdt MetricDataType) String() string {
	switch mdt {
	case MetricDataNone:
		return "None"
	case MetricDataIntGauge:
		return "IntGauge"
	case MetricDataDoubleGauge:
		return "DoubleGauge"
	case MetricDataIntSum:
		return "IntSum"
	case MetricDataDoubleSum:
		return "DoubleSum"
	case MetricDataIntHistogram:
		return "IntHistogram"
	case MetricDataDoubleHistogram:
		return "DoubleHistogram"
	}
	return ""
}

// Type returns the type of the data for this Metric.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DataType() MetricDataType {
	if *ms.orig == nil || (*ms.orig).Data == nil {
		return MetricDataNone
	}
	switch (*ms.orig).Data.(type) {
	case *otlpmetrics.Metric_IntGauge:
		return MetricDataIntGauge
	case *otlpmetrics.Metric_DoubleGauge:
		return MetricDataDoubleGauge
	case *otlpmetrics.Metric_IntSum:
		return MetricDataIntSum
	case *otlpmetrics.Metric_DoubleSum:
		return MetricDataDoubleSum
	case *otlpmetrics.Metric_IntHistogram:
		return MetricDataIntHistogram
	case *otlpmetrics.Metric_DoubleHistogram:
		return MetricDataDoubleHistogram
	}
	return MetricDataNone
}

// IntGaugeData returns the data as IntGauge. This should be called iff DataType() == MetricDataIntGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntGaugeData() IntGauge {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_IntGauge); ok {
		return newIntGauge(&orig.IntGauge)
	}
	return NewIntGauge()
}

// DoubleGaugeData returns the data as DoubleGauge. This should be called iff DataType() == MetricDataDoubleGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleGaugeData() DoubleGauge {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_DoubleGauge); ok {
		return newDoubleGauge(&orig.DoubleGauge)
	}
	return NewDoubleGauge()
}

// IntSumData returns the data as IntSum. This should be called iff DataType() == MetricDataIntSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntSumData() IntSum {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_IntSum); ok {
		return newIntSum(&orig.IntSum)
	}
	return NewIntSum()
}

// DoubleSumData returns the data as DoubleSum. This should be called iff DataType() == MetricDataDoubleSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleSumData() DoubleSum {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_DoubleSum); ok {
		return newDoubleSum(&orig.DoubleSum)
	}
	return NewDoubleSum()
}

// IntHistogramData returns the data as IntHistogram. This should be called iff DataType() == MetricDataIntHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntHistogramData() IntHistogram {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_IntHistogram); ok {
		return newIntHistogram(&orig.IntHistogram)
	}
	return NewIntHistogram()
}

// DoubleHistogramData returns the data as DoubleHistogram. This should be called iff DataType() == MetricDataDoubleHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleHistogramData() DoubleHistogram {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_DoubleHistogram); ok {
		return newDoubleHistogram(&orig.DoubleHistogram)
	}
	return NewDoubleHistogram()
}

// SetIntGaugeData replaces the metric data with the given IntGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetIntGaugeData(data IntGauge) {
	(*ms.orig).Data = &otlpmetrics.Metric_IntGauge{
		IntGauge: *data.orig,
	}
}

// SetDoubleGaugeData replaces the metric data with the given DoubleGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDoubleGaugeData(data DoubleGauge) {
	(*ms.orig).Data = &otlpmetrics.Metric_DoubleGauge{
		DoubleGauge: *data.orig,
	}
}

// SetIntSumData replaces the metric data with the given IntSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetIntSumData(data IntSum) {
	(*ms.orig).Data = &otlpmetrics.Metric_IntSum{
		IntSum: *data.orig,
	}
}

// SetDoubleSumData replaces the metric data with the given DoubleSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDoubleSumData(data DoubleSum) {
	(*ms.orig).Data = &otlpmetrics.Metric_DoubleSum{
		DoubleSum: *data.orig,
	}
}

// SetIntHistogramData replaces the metric data with the given IntHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetIntHistogramData(data IntHistogram) {
	(*ms.orig).Data = &otlpmetrics.Metric_IntHistogram{
		IntHistogram: *data.orig,
	}
}

// SetDoubleHistogramData replaces the metric data with the given DoubleHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDoubleHistogramData(data DoubleHistogram) {
	(*ms.orig).Data = &otlpmetrics.Metric_DoubleHistogram{
		DoubleHistogram: *data.orig,
	}
}

// DeprecatedNewResource temporary public function.
func DeprecatedNewResource(orig **otlpresource.Resource) Resource {
	return newResource(orig)
}

// DeprecatedNewInstrumentationLibrary temporary public function.
func DeprecatedNewInstrumentationLibrary(orig **otlpcommon.InstrumentationLibrary) InstrumentationLibrary {
	return newInstrumentationLibrary(orig)
}

// DeprecatedNewStringMap temporary public function.
func DeprecatedNewStringMap(orig *[]*otlpcommon.StringKeyValue) StringMap {
	return newStringMap(orig)
}

// DeprecatedNewMetricsResourceSlice temporary public function.
func DeprecatedNewMetricsResourceSlice(orig *[]*otlpmetrics.ResourceMetrics) ResourceMetricsSlice {
	return newResourceMetricsSlice(orig)
}
