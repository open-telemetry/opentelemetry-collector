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
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
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
type Metrics struct {
	orig *[]*otlpmetrics.ResourceMetrics
}

// NewMetricData creates a new MetricData.
func NewMetrics() Metrics {
	orig := []*otlpmetrics.ResourceMetrics(nil)
	return Metrics{&orig}
}

// MetricDataFromOtlp creates the internal MetricData representation from the OTLP.
func MetricsFromOtlp(orig []*otlpmetrics.ResourceMetrics) Metrics {
	return Metrics{&orig}
}

// MetricDataToOtlp converts the internal MetricData to the OTLP.
func MetricsToOtlp(md Metrics) []*otlpmetrics.ResourceMetrics {
	return *md.orig
}

// ToOtlpProtoBytes returns the internal MetricData to the OTLP Collector
// ExportMetricsServiceRequest ProtoBuf bytes. This is intended to export
// OTLP Protobuf bytes for OTLP/HTTP transports.
func (md Metrics) ToOtlpProtoBytes() ([]byte, error) {
	metrics := otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: *md.orig,
	}
	return metrics.Marshal()
}

// FromOtlpProtoBytes converts OTLP Collector ExportMetricsServiceRequest
// ProtoBuf bytes to the internal Metrics. Overrides current data.
// Calling this function on zero-initialized structure causes panic.
// Use it with NewMetrics or on existing initialized Metrics.
func (md Metrics) FromOtlpProtoBytes(data []byte) error {
	metrics := otlpcollectormetrics.ExportMetricsServiceRequest{}
	if err := metrics.Unmarshal(data); err != nil {
		return err
	}
	*md.orig = metrics.ResourceMetrics
	return nil
}

// Clone returns a copy of MetricData.
func (md Metrics) Clone() Metrics {
	rms := NewResourceMetricsSlice()
	md.ResourceMetrics().CopyTo(rms)
	return Metrics(rms)
}

func (md Metrics) ResourceMetrics() ResourceMetricsSlice {
	return newResourceMetricsSlice(md.orig)
}

// MetricCount calculates the total number of metrics.
func (md Metrics) MetricCount() int {
	metricCount := 0
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			metricCount += ilm.Metrics().Len()
		}
	}
	return metricCount
}

// Size returns size in bytes.
func (md Metrics) Size() int {
	size := 0
	for i := 0; i < len(*md.orig); i++ {
		if (*md.orig)[i] == nil {
			continue
		}
		size += (*(*md.orig)[i]).Size()
	}
	return size
}

// MetricAndDataPointCount calculates the total number of metrics and data points.
func (md Metrics) MetricAndDataPointCount() (metricCount int, dataPointCount int) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			metrics := ilm.Metrics()
			metricCount += metrics.Len()
			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				switch m.DataType() {
				case MetricDataTypeIntGauge:
					dataPointCount += m.IntGauge().DataPoints().Len()
				case MetricDataTypeDoubleGauge:
					dataPointCount += m.DoubleGauge().DataPoints().Len()
				case MetricDataTypeIntSum:
					dataPointCount += m.IntSum().DataPoints().Len()
				case MetricDataTypeDoubleSum:
					dataPointCount += m.DoubleSum().DataPoints().Len()
				case MetricDataTypeIntHistogram:
					dataPointCount += m.IntHistogram().DataPoints().Len()
				case MetricDataTypeDoubleHistogram:
					dataPointCount += m.DoubleHistogram().DataPoints().Len()
				case MetricDataTypeDoubleSummary:
					dataPointCount += m.DoubleSummary().DataPoints().Len()
				}
			}
		}
	}
	return
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
	MetricDataTypeDoubleSummary
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
	case MetricDataTypeDoubleSummary:
		return "DoubleSummary"
	}
	return ""
}

// DataType returns the type of the data for this Metric.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DataType() MetricDataType {
	switch ms.orig.Data.(type) {
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
	case *otlpmetrics.Metric_DoubleSummary:
		return MetricDataTypeDoubleSummary
	}
	return MetricDataTypeNone
}

// SetDataType clears any existing data and initialize it with an empty data of the given type.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDataType(ty MetricDataType) {
	switch ty {
	case MetricDataTypeIntGauge:
		ms.orig.Data = &otlpmetrics.Metric_IntGauge{IntGauge: &otlpmetrics.IntGauge{}}
	case MetricDataTypeDoubleGauge:
		ms.orig.Data = &otlpmetrics.Metric_DoubleGauge{DoubleGauge: &otlpmetrics.DoubleGauge{}}
	case MetricDataTypeIntSum:
		ms.orig.Data = &otlpmetrics.Metric_IntSum{IntSum: &otlpmetrics.IntSum{}}
	case MetricDataTypeDoubleSum:
		ms.orig.Data = &otlpmetrics.Metric_DoubleSum{DoubleSum: &otlpmetrics.DoubleSum{}}
	case MetricDataTypeIntHistogram:
		ms.orig.Data = &otlpmetrics.Metric_IntHistogram{IntHistogram: &otlpmetrics.IntHistogram{}}
	case MetricDataTypeDoubleHistogram:
		ms.orig.Data = &otlpmetrics.Metric_DoubleHistogram{DoubleHistogram: &otlpmetrics.DoubleHistogram{}}
	case MetricDataTypeDoubleSummary:
		ms.orig.Data = &otlpmetrics.Metric_DoubleSummary{DoubleSummary: &otlpmetrics.DoubleSummary{}}
	}
}

// IntGauge returns the data as IntGauge.
// Calling this function when DataType() != MetricDataTypeIntGauge will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntGauge() IntGauge {
	return newIntGauge(ms.orig.Data.(*otlpmetrics.Metric_IntGauge).IntGauge)
}

// DoubleGauge returns the data as DoubleGauge.
// Calling this function when DataType() != MetricDataTypeDoubleGauge will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleGauge() DoubleGauge {
	return newDoubleGauge(ms.orig.Data.(*otlpmetrics.Metric_DoubleGauge).DoubleGauge)
}

// IntSum returns the data as IntSum.
// Calling this function when DataType() != MetricDataTypeIntSum  will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntSum() IntSum {
	return newIntSum(ms.orig.Data.(*otlpmetrics.Metric_IntSum).IntSum)
}

// DoubleSum returns the data as DoubleSum.
// Calling this function when DataType() != MetricDataTypeDoubleSum will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleSum() DoubleSum {
	return newDoubleSum(ms.orig.Data.(*otlpmetrics.Metric_DoubleSum).DoubleSum)
}

// IntHistogram returns the data as IntHistogram.
// Calling this function when DataType() != MetricDataTypeIntHistogram will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntHistogram() IntHistogram {
	return newIntHistogram(ms.orig.Data.(*otlpmetrics.Metric_IntHistogram).IntHistogram)
}

// DoubleHistogram returns the data as DoubleHistogram.
// Calling this function when DataType() != MetricDataTypeDoubleHistogram will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleHistogram() DoubleHistogram {
	return newDoubleHistogram(ms.orig.Data.(*otlpmetrics.Metric_DoubleHistogram).DoubleHistogram)
}

// DoubleSummary returns the data as DoubleSummary.
// Calling this function when DataType() != MetricDataTypeDoubleSummary will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleSummary() DoubleSummary {
	return newDoubleSummary(ms.orig.Data.(*otlpmetrics.Metric_DoubleSummary).DoubleSummary)
}

func copyData(src, dest *otlpmetrics.Metric) {
	switch srcData := (src).Data.(type) {
	case *otlpmetrics.Metric_IntGauge:
		data := &otlpmetrics.Metric_IntGauge{IntGauge: &otlpmetrics.IntGauge{}}
		newIntGauge(srcData.IntGauge).CopyTo(newIntGauge(data.IntGauge))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleGauge:
		data := &otlpmetrics.Metric_DoubleGauge{DoubleGauge: &otlpmetrics.DoubleGauge{}}
		newDoubleGauge(srcData.DoubleGauge).CopyTo(newDoubleGauge(data.DoubleGauge))
		dest.Data = data
	case *otlpmetrics.Metric_IntSum:
		data := &otlpmetrics.Metric_IntSum{IntSum: &otlpmetrics.IntSum{}}
		newIntSum(srcData.IntSum).CopyTo(newIntSum(data.IntSum))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleSum:
		data := &otlpmetrics.Metric_DoubleSum{DoubleSum: &otlpmetrics.DoubleSum{}}
		newDoubleSum(srcData.DoubleSum).CopyTo(newDoubleSum(data.DoubleSum))
		dest.Data = data
	case *otlpmetrics.Metric_IntHistogram:
		data := &otlpmetrics.Metric_IntHistogram{IntHistogram: &otlpmetrics.IntHistogram{}}
		newIntHistogram(srcData.IntHistogram).CopyTo(newIntHistogram(data.IntHistogram))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleHistogram:
		data := &otlpmetrics.Metric_DoubleHistogram{DoubleHistogram: &otlpmetrics.DoubleHistogram{}}
		newDoubleHistogram(srcData.DoubleHistogram).CopyTo(newDoubleHistogram(data.DoubleHistogram))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleSummary:
		data := &otlpmetrics.Metric_DoubleSummary{DoubleSummary: &otlpmetrics.DoubleSummary{}}
		newDoubleSummary(srcData.DoubleSummary).CopyTo(newDoubleSummary(data.DoubleSummary))
		dest.Data = data
	}
}
