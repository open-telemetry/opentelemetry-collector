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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricData(t *testing.T) {
	for _, arg := range []struct {
		metricData    MetricData
		typ           string
		hasAggregated bool
		hasMonotonic  bool
	}{
		{&gauge{}, "Gauge", false, false},
		{&sum{}, "Sum", true, true},
		{&histogram{}, "Histogram", true, false},
	} {
		assert.Equal(t, arg.typ, arg.metricData.Type())
		assert.Equal(t, arg.hasAggregated, arg.metricData.HasAggregated())
		assert.Equal(t, arg.hasMonotonic, arg.metricData.HasMonotonic())
	}
}

func TestAggregation(t *testing.T) {
	delta := Aggregated{Aggregation: "delta"}
	assert.Equal(t, "pdata.AggregationTemporalityDelta", delta.Type())

	cumulative := Aggregated{Aggregation: "cumulative"}
	assert.Equal(t, "pdata.AggregationTemporalityCumulative", cumulative.Type())

	unknown := Aggregated{Aggregation: ""}
	assert.Equal(t, "pdata.AggregationTemporalityUnknown", unknown.Type())
}
