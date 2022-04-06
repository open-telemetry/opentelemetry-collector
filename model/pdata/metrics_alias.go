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

package pdata // import "go.opentelemetry.io/collector/model/pdata"

// This file contains aliases for metric data structures.

import "go.opentelemetry.io/collector/model/internal"

// MetricsMarshaler is an alias for internal.MetricsMarshaler interface.
type MetricsMarshaler = internal.MetricsMarshaler

// MetricsUnmarshaler is an alias for internal.MetricsUnmarshaler interface.
type MetricsUnmarshaler = internal.MetricsUnmarshaler

// MetricsSizer is an alias for internal.MetricsSizer interface.
type MetricsSizer = internal.MetricsSizer

// Metrics is an alias for internal.Metrics structure.
type Metrics = internal.Metrics

// NewMetrics is an alias for a function to create new Metrics.
var NewMetrics = internal.NewMetrics

// MetricDataType is an alias for internal.MetricDataType type.
type MetricDataType = internal.MetricDataType

const (
	MetricDataTypeNone                 = internal.MetricDataTypeNone
	MetricDataTypeGauge                = internal.MetricDataTypeGauge
	MetricDataTypeSum                  = internal.MetricDataTypeSum
	MetricDataTypeHistogram            = internal.MetricDataTypeHistogram
	MetricDataTypeExponentialHistogram = internal.MetricDataTypeExponentialHistogram
	MetricDataTypeSummary              = internal.MetricDataTypeSummary
)

// MetricAggregationTemporality is an alias for internal.MetricAggregationTemporality type.
type MetricAggregationTemporality = internal.MetricAggregationTemporality

const (
	MetricAggregationTemporalityUnspecified = internal.MetricAggregationTemporalityUnspecified
	MetricAggregationTemporalityDelta       = internal.MetricAggregationTemporalityDelta
	MetricAggregationTemporalityCumulative  = internal.MetricAggregationTemporalityCumulative
)

// MetricDataPointFlags is an alias for internal.MetricDataPointFlags type.
type MetricDataPointFlags = internal.MetricDataPointFlags

const (
	MetricDataPointFlagsNone = internal.MetricDataPointFlagsNone
)

// NewMetricDataPointFlags is an alias for a function to create new MetricDataPointFlags.
var NewMetricDataPointFlags = internal.NewMetricDataPointFlags

// MetricDataPointFlag is an alias for internal.MetricDataPointFlag type.
type MetricDataPointFlag = internal.MetricDataPointFlag

const (
	MetricDataPointFlagNoRecordedValue = internal.MetricDataPointFlagNoRecordedValue
)

// MetricValueType is an alias for internal.MetricValueType type.
type MetricValueType = internal.MetricValueType

const (
	MetricValueTypeNone   = internal.MetricValueTypeNone
	MetricValueTypeInt    = internal.MetricValueTypeInt
	MetricValueTypeDouble = internal.MetricValueTypeDouble
)

// Deprecated: [v0.48.0] Use ScopeMetricsSlice instead.
type InstrumentationLibraryMetricsSlice = internal.ScopeMetricsSlice

// Deprecated: [v0.48.0] Use NewScopeMetricsSlice instead.
var NewInstrumentationLibraryMetricsSlice = internal.NewScopeMetricsSlice

// Deprecated: [v0.48.0] Use ScopeMetrics instead.
type InstrumentationLibraryMetrics = internal.ScopeMetrics

// Deprecated: [v0.48.0] Use NewScopeMetrics instead.
var NewInstrumentationLibraryMetrics = internal.NewScopeMetrics
