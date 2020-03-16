// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset/factory"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
)

// Matcher matches metrics by metric properties against prespecified values for each property.
type Matcher struct {
	nameFilters filterset.FilterSet
}

// MatchMetric matches a metric using the metric properties configured on the Matcher.
// A metric only matches if every metric property configured on the Matcher is a match.
func (m *Matcher) MatchMetric(metric *metricspb.Metric) bool {
	name := metric.GetMetricDescriptor().GetName()
	return m.nameFilters.Matches(name)
}

// NewMatcher constructs a metric Matcher that can be used to match metrics by metric properties.
// For each supported metric property, the Matcher accepts a set of prespecified values. An incoming metric
// matches on a property if the property matches at least one of the prespecified values.
// A metric only matches if every metric property configured on the Matcher is a match.
//
// The metric Matcher supports matching by the following metric properties:
// - Metric name
func NewMatcher(config *MatchProperties) (Matcher, error) {
	f := factory.Factory{}

	nameFS, err := f.CreateFilterSet(config.MetricNames, &config.MatchConfig)
	if err != nil {
		return Matcher{}, err
	}

	return Matcher{
		nameFilters: nameFS,
	}, nil
}
