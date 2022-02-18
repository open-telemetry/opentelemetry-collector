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
	"go.opentelemetry.io/collector/model/pmetric"
)

// MetricsMarshaler is an alias for pmetric.MetricsMarshaler interface.
type MetricsMarshaler = pmetric.MetricsMarshaler

// MetricsUnmarshaler is an alias for pmetric.MetricsUnmarshaler interface.
type MetricsUnmarshaler = pmetric.MetricsUnmarshaler

// MetricsSizer is an alias for pmetric.MetricsSizer interface.
type MetricsSizer = pmetric.MetricsSizer

// Metrics is an alias for pmetric.Metrics structure.
type Metrics = pmetric.Metrics

// NewMetrics is an alias for a function to create new Metrics.
var NewMetrics = pmetric.NewMetrics

// MetricsFromInternalRep is an alias for MetricsFromInternalRep.
// TODO: Can be removed, internal pmetric.MetricsFromInternalRep should be used instead.
var MetricsFromInternalRep = pmetric.MetricsFromInternalRep

// MetricDataType is an alias for pmetric.MetricDataType type.
type MetricDataType = pmetric.MetricDataType

const (
	MetricDataTypeNone                 = pmetric.MetricDataTypeNone
	MetricDataTypeGauge                = pmetric.MetricDataTypeGauge
	MetricDataTypeSum                  = pmetric.MetricDataTypeSum
	MetricDataTypeHistogram            = pmetric.MetricDataTypeHistogram
	MetricDataTypeExponentialHistogram = pmetric.MetricDataTypeExponentialHistogram
	MetricDataTypeSummary              = pmetric.MetricDataTypeSummary
)

// MetricAggregationTemporality is an alias for pmetric.MetricAggregationTemporality type.
type MetricAggregationTemporality = pmetric.MetricAggregationTemporality

const (
	MetricAggregationTemporalityUnspecified = pmetric.MetricAggregationTemporalityUnspecified
	MetricAggregationTemporalityDelta       = pmetric.MetricAggregationTemporalityDelta
	MetricAggregationTemporalityCumulative  = pmetric.MetricAggregationTemporalityCumulative
)

// MetricDataPointFlags is an alias for pmetric.MetricDataPointFlags type.
type MetricDataPointFlags = pmetric.MetricDataPointFlags

const (
	MetricDataPointFlagsNone = pmetric.MetricDataPointFlagsNone
)

// NewMetricDataPointFlags is an alias for a function to create new MetricDataPointFlags.
var NewMetricDataPointFlags = pmetric.NewMetricDataPointFlags

// MetricDataPointFlag is an alias for pmetric.MetricDataPointFlag type.
type MetricDataPointFlag = pmetric.MetricDataPointFlag

const (
	MetricDataPointFlagNoRecordedValue = pmetric.MetricDataPointFlagNoRecordedValue
)

// MetricValueType is an alias for pmetric.MetricValueType type.
type MetricValueType = pmetric.MetricValueType

const (
	MetricValueTypeNone   = pmetric.MetricValueTypeNone
	MetricValueTypeInt    = pmetric.MetricValueTypeInt
	MetricValueTypeDouble = pmetric.MetricValueTypeDouble
)
