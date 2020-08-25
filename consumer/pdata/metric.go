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
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

type MetricType otlpmetrics.MetricDescriptor_Type

const (
	MetricTypeInvalid         = MetricType(otlpmetrics.MetricDescriptor_INVALID_TYPE)
	MetricTypeInt64           = MetricType(otlpmetrics.MetricDescriptor_INT64)
	MetricTypeDouble          = MetricType(otlpmetrics.MetricDescriptor_DOUBLE)
	MetricTypeMonotonicInt64  = MetricType(otlpmetrics.MetricDescriptor_MONOTONIC_INT64)
	MetricTypeMonotonicDouble = MetricType(otlpmetrics.MetricDescriptor_MONOTONIC_DOUBLE)
	MetricTypeHistogram       = MetricType(otlpmetrics.MetricDescriptor_HISTOGRAM)
	MetricTypeSummary         = MetricType(otlpmetrics.MetricDescriptor_SUMMARY)
)

func (mt MetricType) String() string {
	return otlpmetrics.MetricDescriptor_Type(mt).String()
}

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
