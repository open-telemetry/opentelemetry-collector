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

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

// MetricsToOtlp internal helper to convert Metrics to otlp request representation.
func MetricsToOtlp(mw Metrics) *otlpcollectormetrics.ExportMetricsServiceRequest {
	return mw.orig
}

// MetricsFromOtlp internal helper to convert otlp request representation to Metrics.
func MetricsFromOtlp(orig *otlpcollectormetrics.ExportMetricsServiceRequest) Metrics {
	return Metrics{orig: orig}
}

// MetricsToProto internal helper to convert Metrics to protobuf representation.
func MetricsToProto(l Metrics) otlpmetrics.MetricsData {
	return otlpmetrics.MetricsData{
		ResourceMetrics: l.orig.ResourceMetrics,
	}
}

// MetricsFromProto internal helper to convert protobuf representation to Metrics.
func MetricsFromProto(orig otlpmetrics.MetricsData) Metrics {
	return Metrics{orig: &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: orig.ResourceMetrics,
	}}
}

// Metrics is the top-level struct that is propagated through the metrics pipeline.
// Use NewMetrics to create new instance, zero-initialized instance is not valid for use.
type Metrics struct {
	orig *otlpcollectormetrics.ExportMetricsServiceRequest
}

// NewMetrics creates a new Metrics struct.
func NewMetrics() Metrics {
	return Metrics{orig: &otlpcollectormetrics.ExportMetricsServiceRequest{}}
}

// Clone returns a copy of MetricData.
func (md Metrics) Clone() Metrics {
	cloneMd := NewMetrics()
	md.ResourceMetrics().CopyTo(cloneMd.ResourceMetrics())
	return cloneMd
}

// MoveTo moves all properties from the current struct to dest
// resetting the current instance to its zero value.
func (md Metrics) MoveTo(dest Metrics) {
	*dest.orig = *md.orig
	*md.orig = otlpcollectormetrics.ExportMetricsServiceRequest{}
}

// ResourceMetrics returns the ResourceMetricsSlice associated with this Metrics.
func (md Metrics) ResourceMetrics() ResourceMetricsSlice {
	return newResourceMetricsSlice(&md.orig.ResourceMetrics)
}

// MetricCount calculates the total number of metrics.
func (md Metrics) MetricCount() int {
	metricCount := 0
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			metricCount += ilm.Metrics().Len()
		}
	}
	return metricCount
}

// DataPointCount calculates the total number of data points.
func (md Metrics) DataPointCount() (dataPointCount int) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				switch m.DataType() {
				case MetricDataTypeGauge:
					dataPointCount += m.Gauge().DataPoints().Len()
				case MetricDataTypeSum:
					dataPointCount += m.Sum().DataPoints().Len()
				case MetricDataTypeHistogram:
					dataPointCount += m.Histogram().DataPoints().Len()
				case MetricDataTypeExponentialHistogram:
					dataPointCount += m.ExponentialHistogram().DataPoints().Len()
				case MetricDataTypeSummary:
					dataPointCount += m.Summary().DataPoints().Len()
				}
			}
		}
	}
	return
}

// MetricDataType specifies the type of data in a Metric.
type MetricDataType int32

const (
	MetricDataTypeNone MetricDataType = iota
	MetricDataTypeGauge
	MetricDataTypeSum
	MetricDataTypeHistogram
	MetricDataTypeExponentialHistogram
	MetricDataTypeSummary
)

// String returns the string representation of the MetricDataType.
func (mdt MetricDataType) String() string {
	switch mdt {
	case MetricDataTypeNone:
		return "None"
	case MetricDataTypeGauge:
		return "Gauge"
	case MetricDataTypeSum:
		return "Sum"
	case MetricDataTypeHistogram:
		return "Histogram"
	case MetricDataTypeExponentialHistogram:
		return "ExponentialHistogram"
	case MetricDataTypeSummary:
		return "Summary"
	}
	return ""
}

// SetDataType clears any existing data and initialize it with an empty data of the given type.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDataType(ty MetricDataType) {
	switch ty {
	case MetricDataTypeGauge:
		ms.orig.Data = &otlpmetrics.Metric_Gauge{Gauge: &otlpmetrics.Gauge{}}
	case MetricDataTypeSum:
		ms.orig.Data = &otlpmetrics.Metric_Sum{Sum: &otlpmetrics.Sum{}}
	case MetricDataTypeHistogram:
		ms.orig.Data = &otlpmetrics.Metric_Histogram{Histogram: &otlpmetrics.Histogram{}}
	case MetricDataTypeExponentialHistogram:
		ms.orig.Data = &otlpmetrics.Metric_ExponentialHistogram{ExponentialHistogram: &otlpmetrics.ExponentialHistogram{}}
	case MetricDataTypeSummary:
		ms.orig.Data = &otlpmetrics.Metric_Summary{Summary: &otlpmetrics.Summary{}}
	}
}

// MetricAggregationTemporality defines how a metric aggregator reports aggregated values.
// It describes how those values relate to the time interval over which they are aggregated.
type MetricAggregationTemporality int32

const (
	// MetricAggregationTemporalityUnspecified is the default MetricAggregationTemporality, it MUST NOT be used.
	MetricAggregationTemporalityUnspecified = MetricAggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED)
	// MetricAggregationTemporalityDelta is a MetricAggregationTemporality for a metric aggregator which reports changes since last report time.
	MetricAggregationTemporalityDelta = MetricAggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA)
	// MetricAggregationTemporalityCumulative is a MetricAggregationTemporality for a metric aggregator which reports changes since a fixed start time.
	MetricAggregationTemporalityCumulative = MetricAggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE)
)

// String returns the string representation of the MetricAggregationTemporality.
func (at MetricAggregationTemporality) String() string {
	return otlpmetrics.AggregationTemporality(at).String()
}

// FlagsStruct returns the flagsstruct associated with this NumberDataPoint.
// Deprecated [0.58.0] Use Flags() instead
func (ms NumberDataPoint) FlagsStruct() MetricDataPointFlags {
	return ms.Flags()
}

// FlagsStruct returns the flagsstruct associated with this HistogramDataPoint.
// Deprecated [0.58.0] Use Flags() instead
func (ms HistogramDataPoint) FlagsStruct() MetricDataPointFlags {
	return ms.Flags()
}

// FlagsStruct returns the flagsstruct associated with this ExponentialHistogramDataPoint.
// Deprecated [0.58.0] Use Flags() instead
func (ms ExponentialHistogramDataPoint) FlagsStruct() MetricDataPointFlags {
	return ms.Flags()
}

// FlagsStruct returns the flagsstruct associated with this SummaryDataPoint.
// Deprecated [0.58.0] Use Flags() instead
func (ms SummaryDataPoint) FlagsStruct() MetricDataPointFlags {
	return ms.Flags()
}

// MetricDataPointFlags defines how a metric aggregator reports aggregated values.
// It describes how those values relate to the time interval over which they are aggregated.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewMetricDataPointFlagsStruct function to create new instances.
// Important: zero-initialized instance is not valid for use.
type MetricDataPointFlags struct {
	orig *uint32
}

func newMetricDataPointFlags(orig *uint32) MetricDataPointFlags {
	return MetricDataPointFlags{orig: orig}
}

// NewMetricDataPointFlags creates a new empty MetricDataPointFlags.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewMetricDataPointFlags() MetricDataPointFlags {
	return newMetricDataPointFlags(new(uint32))
}

// MoveTo moves all properties from the current struct to dest
// resetting the current instance to its zero value
func (ms MetricDataPointFlags) MoveTo(dest MetricDataPointFlags) {
	*dest.orig = *ms.orig
	*ms.orig = uint32(otlpmetrics.DataPointFlags_FLAG_NONE)
}

// CopyTo copies all properties from the current struct to the dest.
func (ms MetricDataPointFlags) CopyTo(dest MetricDataPointFlags) {
	*dest.orig = *ms.orig
}

// NoRecordedValue returns true if the MetricDataPointFlags contains the NO_RECORDED_VALUE flag.
func (ms MetricDataPointFlags) NoRecordedValue() bool {
	return *ms.orig&uint32(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE) != 0
}

// SetNoRecordedValue sets the FLAG_NO_RECORDED_VALUE flag if true and removes it if false.
// Setting this Flag when it is already set will change nothing.
func (ms MetricDataPointFlags) SetNoRecordedValue(b bool) {
	if b {
		*ms.orig |= uint32(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE)
	} else {
		*ms.orig &^= uint32(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE)
	}
}

// String returns the string representation of the MetricDataPointFlags.
func (ms MetricDataPointFlags) String() string {
	return otlpmetrics.DataPointFlags(*ms.orig).String()
}

// NumberDataPointValueType specifies the type of NumberDataPoint value.
type NumberDataPointValueType int32

const (
	NumberDataPointValueTypeNone NumberDataPointValueType = iota
	NumberDataPointValueTypeInt
	NumberDataPointValueTypeDouble
)

// String returns the string representation of the NumberDataPointValueType.
func (nt NumberDataPointValueType) String() string {
	switch nt {
	case NumberDataPointValueTypeNone:
		return "None"
	case NumberDataPointValueTypeInt:
		return "Int"
	case NumberDataPointValueTypeDouble:
		return "Double"
	}
	return ""
}

// ExemplarValueType specifies the type of Exemplar measurement value.
type ExemplarValueType int32

const (
	ExemplarValueTypeNone ExemplarValueType = iota
	ExemplarValueTypeInt
	ExemplarValueTypeDouble
)

// String returns the string representation of the ExemplarValueType.
func (nt ExemplarValueType) String() string {
	switch nt {
	case ExemplarValueTypeNone:
		return "None"
	case ExemplarValueTypeInt:
		return "Int"
	case ExemplarValueTypeDouble:
		return "Double"
	}
	return ""
}

// OptionalType wraps optional fields into oneof fields
type OptionalType int32

const (
	OptionalTypeNone OptionalType = iota
	OptionalTypeDouble
)

// String returns the string representation of the OptionalType.
func (ot OptionalType) String() string {
	switch ot {
	case OptionalTypeNone:
		return "None"
	case OptionalTypeDouble:
		return "Double"
	}
	return ""
}
