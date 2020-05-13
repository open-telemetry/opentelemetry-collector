// Copyright The OpenTelemetry Authors
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

package pdata

import (
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
)

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

// InternalNewMetricsResourceSlice is a helper
func InternalNewMetricsResourceSlice(orig *[]*otlpmetrics.ResourceMetrics) ResourceMetricsSlice {
	return newResourceMetricsSlice(orig)
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
