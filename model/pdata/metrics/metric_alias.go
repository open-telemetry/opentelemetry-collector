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

package metrics // import "go.opentelemetry.io/collector/model/pdata/metrics"

// This file contains aliases for metrics data structures.

import "go.opentelemetry.io/collector/model/internal/pdata"

// Marshaler is an alias for pdata.MetricsMarshaler interface.
type Marshaler = pdata.MetricsMarshaler

// Unmarshaler is an alias for pdata.MetricsUnmarshaler interface.
type Unmarshaler = pdata.MetricsUnmarshaler

// Sizer is an alias for pdata.MetricsSizer interface.
type Sizer = pdata.MetricsSizer

// Metrics is an alias for pdata.Metrics structure.
type Metrics = pdata.Metrics

// New is an alias for a function to create new Metrics.
var New = pdata.NewMetrics

// MetricDataType is an alias for pdata.MetricDataType type.
type MetricDataType = pdata.MetricDataType

const (
	MetricDataTypeNone                 = pdata.MetricDataTypeNone
	MetricDataTypeGauge                = pdata.MetricDataTypeGauge
	MetricDataTypeSum                  = pdata.MetricDataTypeSum
	MetricDataTypeHistogram            = pdata.MetricDataTypeHistogram
	MetricDataTypeExponentialHistogram = pdata.MetricDataTypeExponentialHistogram
	MetricDataTypeSummary              = pdata.MetricDataTypeSummary
)

// MetricAggregationTemporality is an alias for pdata.MetricAggregationTemporality type.
type MetricAggregationTemporality = pdata.MetricAggregationTemporality

const (
	MetricAggregationTemporalityUnspecified = pdata.MetricAggregationTemporalityUnspecified
	MetricAggregationTemporalityDelta       = pdata.MetricAggregationTemporalityDelta
	MetricAggregationTemporalityCumulative  = pdata.MetricAggregationTemporalityCumulative
)

// MetricDataPointFlags is an alias for pdata.MetricDataPointFlags type.
type MetricDataPointFlags = pdata.MetricDataPointFlags

const (
	MetricDataPointFlagsNone = pdata.MetricDataPointFlagsNone
)

// NewMetricDataPointFlags is an alias for a function to create new MetricDataPointFlags.
var NewMetricDataPointFlags = pdata.NewMetricDataPointFlags

// MetricDataPointFlag is an alias for pdata.MetricDataPointFlag type.
type MetricDataPointFlag = pdata.MetricDataPointFlag

const (
	MetricDataPointFlagNoRecordedValue = pdata.MetricDataPointFlagNoRecordedValue
)

// MetricValueType is an alias for pdata.MetricValueType type.
type MetricValueType = pdata.MetricValueType

const (
	MetricValueTypeNone   = pdata.MetricValueTypeNone
	MetricValueTypeInt    = pdata.MetricValueTypeInt
	MetricValueTypeDouble = pdata.MetricValueTypeDouble
)
