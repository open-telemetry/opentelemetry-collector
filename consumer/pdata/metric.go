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
	MetricDataTypeNone MetricDataType = iota
	MetricDataTypeIntGauge
	MetricDataTypeDoubleGauge
	MetricDataTypeIntSum
	MetricDataTypeDoubleSum
	MetricDataTypeIntHistogram
	MetricDataTypeDoubleHistogram
)

func (mdt MetricDataType) String() string {
	switch mdt {
	case MetricDataTypeNone:
		return "None"
	case MetricDataTypeIntGauge:
		return "IntGauge"
	case MetricDataTypeDoubleGauge:
		return "DoubleGauge"
	case MetricDataTypeIntSum:
		return "IntSum"
	case MetricDataTypeDoubleSum:
		return "DoubleSum"
	case MetricDataTypeIntHistogram:
		return "IntHistogram"
	case MetricDataTypeDoubleHistogram:
		return "DoubleHistogram"
	}
	return ""
}

// Type returns the type of the data for this Metric.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DataType() MetricDataType {
	if *ms.orig == nil || (*ms.orig).Data == nil {
		return MetricDataTypeNone
	}
	switch (*ms.orig).Data.(type) {
	case *otlpmetrics.Metric_IntGauge:
		return MetricDataTypeIntGauge
	case *otlpmetrics.Metric_DoubleGauge:
		return MetricDataTypeDoubleGauge
	case *otlpmetrics.Metric_IntSum:
		return MetricDataTypeIntSum
	case *otlpmetrics.Metric_DoubleSum:
		return MetricDataTypeDoubleSum
	case *otlpmetrics.Metric_IntHistogram:
		return MetricDataTypeIntHistogram
	case *otlpmetrics.Metric_DoubleHistogram:
		return MetricDataTypeDoubleHistogram
	}
	return MetricDataTypeNone
}

// IntGauge returns the data as IntGauge. This should be called iff DataType() == MetricDataTypeIntGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntGauge() IntGauge {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_IntGauge); ok {
		return newIntGauge(&orig.IntGauge)
	}
	return NewIntGauge()
}

// DoubleGauge returns the data as DoubleGauge. This should be called iff DataType() == MetricDataTypeDoubleGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleGauge() DoubleGauge {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_DoubleGauge); ok {
		return newDoubleGauge(&orig.DoubleGauge)
	}
	return NewDoubleGauge()
}

// IntSum returns the data as IntSum. This should be called iff DataType() == MetricDataTypeIntSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntSum() IntSum {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_IntSum); ok {
		return newIntSum(&orig.IntSum)
	}
	return NewIntSum()
}

// DoubleSum returns the data as DoubleSum. This should be called iff DataType() == MetricDataTypeDoubleSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleSum() DoubleSum {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_DoubleSum); ok {
		return newDoubleSum(&orig.DoubleSum)
	}
	return NewDoubleSum()
}

// IntHistogram returns the data as IntHistogram. This should be called iff DataType() == MetricDataTypeIntHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntHistogram() IntHistogram {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_IntHistogram); ok {
		return newIntHistogram(&orig.IntHistogram)
	}
	return NewIntHistogram()
}

// DoubleHistogram returns the data as DoubleHistogram. This should be called iff DataType() == MetricDataTypeDoubleHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleHistogram() DoubleHistogram {
	if orig, ok := (*ms.orig).Data.(*otlpmetrics.Metric_DoubleHistogram); ok {
		return newDoubleHistogram(&orig.DoubleHistogram)
	}
	return NewDoubleHistogram()
}

// SetIntGauge replaces the metric data with the given IntGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetIntGauge(data IntGauge) {
	(*ms.orig).Data = &otlpmetrics.Metric_IntGauge{
		IntGauge: *data.orig,
	}
}

// SetDoubleGauge replaces the metric data with the given DoubleGauge.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDoubleGauge(data DoubleGauge) {
	(*ms.orig).Data = &otlpmetrics.Metric_DoubleGauge{
		DoubleGauge: *data.orig,
	}
}

// SetIntSum replaces the metric data with the given IntSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetIntSum(data IntSum) {
	(*ms.orig).Data = &otlpmetrics.Metric_IntSum{
		IntSum: *data.orig,
	}
}

// SetDoubleSum replaces the metric data with the given DoubleSum.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDoubleSum(data DoubleSum) {
	(*ms.orig).Data = &otlpmetrics.Metric_DoubleSum{
		DoubleSum: *data.orig,
	}
}

// SetIntHistogram replaces the metric data with the given IntHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetIntHistogram(data IntHistogram) {
	(*ms.orig).Data = &otlpmetrics.Metric_IntHistogram{
		IntHistogram: *data.orig,
	}
}

// SetDoubleHistogram replaces the metric data with the given DoubleHistogram.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDoubleHistogram(data DoubleHistogram) {
	(*ms.orig).Data = &otlpmetrics.Metric_DoubleHistogram{
		DoubleHistogram: *data.orig,
	}
}

func copyData(src, dest *otlpmetrics.Metric) {
	switch srcData := (src).Data.(type) {
	case *otlpmetrics.Metric_IntGauge:
		data := &otlpmetrics.Metric_IntGauge{}
		newIntGauge(&srcData.IntGauge).CopyTo(newIntGauge(&data.IntGauge))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleGauge:
		data := &otlpmetrics.Metric_DoubleGauge{}
		newDoubleGauge(&srcData.DoubleGauge).CopyTo(newDoubleGauge(&data.DoubleGauge))
		dest.Data = data
	case *otlpmetrics.Metric_IntSum:
		data := &otlpmetrics.Metric_IntSum{}
		newIntSum(&srcData.IntSum).CopyTo(newIntSum(&data.IntSum))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleSum:
		data := &otlpmetrics.Metric_DoubleSum{}
		newDoubleSum(&srcData.DoubleSum).CopyTo(newDoubleSum(&data.DoubleSum))
		dest.Data = data
	case *otlpmetrics.Metric_IntHistogram:
		data := &otlpmetrics.Metric_IntHistogram{}
		newIntHistogram(&srcData.IntHistogram).CopyTo(newIntHistogram(&data.IntHistogram))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleHistogram:
		data := &otlpmetrics.Metric_DoubleHistogram{}
		newDoubleHistogram(&srcData.DoubleHistogram).CopyTo(newDoubleHistogram(&data.DoubleHistogram))
		dest.Data = data
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
