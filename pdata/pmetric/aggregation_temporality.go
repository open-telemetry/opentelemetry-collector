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
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

// AggregationTemporality defines how a metric aggregator reports aggregated values.
// It describes how those values relate to the time interval over which they are aggregated.
type AggregationTemporality int32

const (
	// AggregationTemporalityUnspecified is the default AggregationTemporality, it MUST NOT be used.
	AggregationTemporalityUnspecified = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED)
	// AggregationTemporalityDelta is a AggregationTemporality for a metric aggregator which reports changes since last report time.
	AggregationTemporalityDelta = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA)
	// AggregationTemporalityCumulative is a AggregationTemporality for a metric aggregator which reports changes since a fixed start time.
	AggregationTemporalityCumulative = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE)
)

// String returns the string representation of the AggregationTemporality.
func (at AggregationTemporality) String() string {
	switch at {
	case AggregationTemporalityUnspecified:
		return "Unspecified"
	case AggregationTemporalityDelta:
		return "Delta"
	case AggregationTemporalityCumulative:
		return "Cumulative"
	}
	return ""
}
