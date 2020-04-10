// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	"github.com/golang/protobuf/proto"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
)

// This file defines in-memory data structures to represent metrics.
// For the proto representation see https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/metrics/v1/metrics.proto

// MetricData is the top-level struct that is propagated through the metrics pipeline.
// This is the newer version of consumerdata.MetricsData, but uses more efficient
// in-memory representation.
//
// This is a reference type (like builtin map).
//
// Must use NewMetricData functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type MetricData struct {
	orig *[]*otlpmetrics.ResourceMetrics
}

// MetricDataFromOtlp creates the internal MetricData representation from the OTLP.
func MetricDataFromOtlp(orig []*otlpmetrics.ResourceMetrics) MetricData {
	return MetricData{&orig}
}

// MetricDataToOtlp converts the internal MetricData to the OTLP.
func MetricDataToOtlp(md MetricData) []*otlpmetrics.ResourceMetrics {
	return *md.orig
}

// NewMetricData creates a new MetricData.
func NewMetricData() MetricData {
	orig := []*otlpmetrics.ResourceMetrics(nil)
	return MetricData{&orig}
}

// Clone returns a copy of MetricData.
func (md MetricData) Clone() MetricData {
	otlp := MetricDataToOtlp(md)
	resourceMetricsClones := make([]*otlpmetrics.ResourceMetrics, 0, len(otlp))
	for _, resourceMetrics := range otlp {
		resourceMetricsClones = append(resourceMetricsClones,
			proto.Clone(resourceMetrics).(*otlpmetrics.ResourceMetrics))
	}
	return MetricDataFromOtlp(resourceMetricsClones)
}

func (md MetricData) ResourceMetrics() ResourceMetricsSlice {
	return newResourceMetricsSlice(md.orig)
}

// MetricCount calculates the total number of metrics.
func (md MetricData) MetricCount() int {
	metricCount := 0
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		if rm.IsNil() {
			continue
		}
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			if ilm.IsNil() {
				continue
			}
			metricCount += ilm.Metrics().Len()
		}
	}
	return metricCount
}

// MetricAndDataPointCount calculates the total number of metrics and datapoints.
func (md MetricData) MetricAndDataPointCount() (metricCount int, dataPointCount int) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		ilms := rms.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			metrics := ilms.At(j).Metrics()
			metricCount += metrics.Len()
			for k := 0; k < metrics.Len(); k++ {
				dataPointCount += metrics.At(k).Int64DataPoints().Len()
				dataPointCount += metrics.At(k).DoubleDataPoints().Len()
				dataPointCount += metrics.At(k).HistogramDataPoints().Len()
				dataPointCount += metrics.At(k).SummaryDataPoints().Len()
			}
		}
	}
	return
}

type MetricType otlpmetrics.MetricDescriptor_Type

const (
	MetricTypeUnspecified         = MetricType(otlpmetrics.MetricDescriptor_UNSPECIFIED)
	MetricTypeGaugeInt64          = MetricType(otlpmetrics.MetricDescriptor_GAUGE_INT64)
	MetricTypeGaugeDouble         = MetricType(otlpmetrics.MetricDescriptor_GAUGE_DOUBLE)
	MetricTypeGaugeHistogram      = MetricType(otlpmetrics.MetricDescriptor_GAUGE_HISTOGRAM)
	MetricTypeCounterInt64        = MetricType(otlpmetrics.MetricDescriptor_COUNTER_INT64)
	MetricTypeCounterDouble       = MetricType(otlpmetrics.MetricDescriptor_COUNTER_DOUBLE)
	MetricTypeCumulativeHistogram = MetricType(otlpmetrics.MetricDescriptor_CUMULATIVE_HISTOGRAM)
	MetricTypeSummary             = MetricType(otlpmetrics.MetricDescriptor_SUMMARY)
)
