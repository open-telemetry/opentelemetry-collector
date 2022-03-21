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

import (
	"go.opentelemetry.io/collector/model/internal/pdata"
	"go.opentelemetry.io/collector/model/pmetric"
)

// MetricsMarshaler is an alias for pmetric.MetricsMarshaler interface.
// Deprecated: [v0.49.0] Use pmetric.MetricsMarshaler instead.
type MetricsMarshaler = pmetric.MetricsMarshaler

// MetricsUnmarshaler is an alias for pmetric.MetricsUnmarshaler interface.
// Deprecated: [v0.49.0] Use pmetric.MetricsUnmarshaler instead.
type MetricsUnmarshaler = pmetric.MetricsUnmarshaler

// MetricsSizer is an alias for pmetric.MetricsSizer interface.
// Deprecated: [v0.49.0] Use pmetric.MetricsSizer instead.
type MetricsSizer = pmetric.MetricsSizer

// Metrics is an alias for pmetric.Metrics structure.
// Deprecated: [v0.49.0] Use pmetric.Metrics instead.
type Metrics = pmetric.Metrics

// NewMetrics is an alias for a function to create new Metrics.
// Deprecated: [v0.49.0] Use pmetric.NewMetrics instead.
var NewMetrics = pdata.NewMetrics

// MetricDataType is an alias for pdata.MetricDataType type.
// Deprecated: [v0.49.0] Use pmetric.MetricDataType instead.
type MetricDataType = pdata.MetricDataType

const (

	// Deprecated: [v0.49.0] Use pmetric.MetricDataTypeNone instead.
	MetricDataTypeNone = pdata.MetricDataTypeNone

	// Deprecated: [v0.49.0] Use pmetric.MetricDataTypeGauge instead.
	MetricDataTypeGauge = pdata.MetricDataTypeGauge

	// Deprecated: [v0.49.0] Use pmetric.MetricDataTypeSum instead.
	MetricDataTypeSum = pdata.MetricDataTypeSum

	// Deprecated: [v0.49.0] Use pmetric.MetricDataTypeHistogram instead.
	MetricDataTypeHistogram = pdata.MetricDataTypeHistogram

	// Deprecated: [v0.49.0] Use pmetric.MetricDataTypeExponentialHistogram instead.
	MetricDataTypeExponentialHistogram = pdata.MetricDataTypeExponentialHistogram

	// Deprecated: [v0.49.0] Use pmetric.MetricDataTypeSummary instead.
	MetricDataTypeSummary = pdata.MetricDataTypeSummary
)

// MetricAggregationTemporality is an alias for pdata.MetricAggregationTemporality type.
// Deprecated: [v0.49.0] Use pmetric.MetricAggregationTemporality instead.
type MetricAggregationTemporality = pdata.MetricAggregationTemporality

const (

	// Deprecated: [v0.49.0] Use pmetric.MetricAggregationTemporalityUnspecified instead.
	MetricAggregationTemporalityUnspecified = pdata.MetricAggregationTemporalityUnspecified

	// Deprecated: [v0.49.0] Use pmetric.MetricAggregationTemporalityDelta instead.
	MetricAggregationTemporalityDelta = pdata.MetricAggregationTemporalityDelta

	// Deprecated: [v0.49.0] Use pmetric.MetricAggregationTemporalityCumulative instead.
	MetricAggregationTemporalityCumulative = pdata.MetricAggregationTemporalityCumulative
)

// MetricDataPointFlags is an alias for pdata.MetricDataPointFlags type.
// Deprecated: [v0.49.0] Use pmetric.MetricDataPointFlags instead.
type MetricDataPointFlags = pdata.MetricDataPointFlags

const (
	// Deprecated: [v0.49.0] Use pmetric.MetricDataPointFlagsNone instead.
	MetricDataPointFlagsNone = pdata.MetricDataPointFlagsNone
)

// NewMetricDataPointFlags is an alias for a function to create new MetricDataPointFlags.
// Deprecated: [v0.49.0] Use pmetric.NewMetricDataPointFlags instead.
var NewMetricDataPointFlags = pdata.NewMetricDataPointFlags

// MetricDataPointFlag is an alias for pdata.MetricDataPointFlag type.
// Deprecated: [v0.49.0] Use pmetric.MetricDataPointFlag instead.
type MetricDataPointFlag = pdata.MetricDataPointFlag

const (
	// Deprecated: [v0.49.0] Use pmetric.MetricDataPointFlagNoRecordedValue instead.
	MetricDataPointFlagNoRecordedValue = pdata.MetricDataPointFlagNoRecordedValue
)

// MetricValueType is an alias for pdata.MetricValueType type.
// Deprecated: [v0.49.0] Use pmetric.MetricValueType instead.
type MetricValueType = pdata.MetricValueType

const (

	// Deprecated: [v0.49.0] Use pmetric.MetricValueTypeNone instead.
	MetricValueTypeNone = pdata.MetricValueTypeNone

	// Deprecated: [v0.49.0] Use pmetric.MetricValueTypeInt instead.
	MetricValueTypeInt = pdata.MetricValueTypeInt

	// Deprecated: [v0.49.0] Use pmetric.MetricValueTypeDouble instead.
	MetricValueTypeDouble = pdata.MetricValueTypeDouble
)

// Deprecated: [v0.48.0] Use ScopeMetricsSlice instead.
type InstrumentationLibraryMetricsSlice = pdata.ScopeMetricsSlice

// Deprecated: [v0.48.0] Use NewScopeMetricsSlice instead.
var NewInstrumentationLibraryMetricsSlice = pdata.NewScopeMetricsSlice

// Deprecated: [v0.48.0] Use ScopeMetrics instead.
type InstrumentationLibraryMetrics = pdata.ScopeMetrics

// Deprecated: [v0.48.0] Use NewScopeMetrics instead.
var NewInstrumentationLibraryMetrics = pdata.NewScopeMetrics
