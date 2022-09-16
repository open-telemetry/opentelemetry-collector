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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

// Metrics is the top-level struct that is propagated through the metrics pipeline.
// Use NewMetrics to create new instance, zero-initialized instance is not valid for use.
type Metrics internal.Metrics

func newMetrics(orig *otlpcollectormetrics.ExportMetricsServiceRequest) Metrics {
	return Metrics(internal.NewMetrics(orig))
}

func (ms Metrics) getOrig() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return internal.GetOrigMetrics(internal.Metrics(ms))
}

// NewMetrics creates a new Metrics struct.
func NewMetrics() Metrics {
	return newMetrics(&otlpcollectormetrics.ExportMetricsServiceRequest{})
}

// Clone returns a copy of MetricData.
func (ms Metrics) Clone() Metrics {
	cloneMd := NewMetrics()
	ms.ResourceMetrics().CopyTo(cloneMd.ResourceMetrics())
	return cloneMd
}

// MoveTo moves all properties from the current struct to dest
// resetting the current instance to its zero value.
func (ms Metrics) MoveTo(dest Metrics) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpcollectormetrics.ExportMetricsServiceRequest{}
}

// ResourceMetrics returns the ResourceMetricsSlice associated with this Metrics.
func (ms Metrics) ResourceMetrics() ResourceMetricsSlice {
	return newResourceMetricsSlice(&ms.getOrig().ResourceMetrics)
}

// MetricCount calculates the total number of metrics.
func (ms Metrics) MetricCount() int {
	metricCount := 0
	rms := ms.ResourceMetrics()
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
func (ms Metrics) DataPointCount() (dataPointCount int) {
	rms := ms.ResourceMetrics()
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

// Deprecated: [v0.61.0] not used, will be deleted in next release.
type OptionalType int32

const (
	// Deprecated: [v0.61.0] not used, will be deleted in next release.
	OptionalTypeNone OptionalType = iota
	// Deprecated: [v0.61.0] not used, will be deleted in next release.
	OptionalTypeDouble
)

// Deprecated: [v0.61.0] not used, will be deleted in next release.
func (ot OptionalType) String() string {
	switch ot {
	case OptionalTypeNone:
		return "None"
	case OptionalTypeDouble:
		return "Double"
	}
	return ""
}
