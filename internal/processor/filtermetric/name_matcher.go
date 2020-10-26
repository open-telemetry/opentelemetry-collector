// Copyright The OpenTelemetry Authors
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

package filtermetric

import (
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

// nameMatcher matches metrics by metric properties against prespecified values for each property.
type nameMatcher struct {
	nameFilters filterset.FilterSet
}

func newNameMatcher(config *MatchProperties) (*nameMatcher, error) {
	nameFS, err := filterset.CreateFilterSet(
		config.MetricNames,
		&filterset.Config{
			MatchType:    filterset.MatchType(config.MatchType),
			RegexpConfig: config.RegexpConfig,
		},
	)
	if err != nil {
		return nil, err
	}
	return &nameMatcher{
		nameFilters: nameFS,
	}, nil
}

// MatchMetric matches a metric using the metric properties configured on the nameMatcher.
// A metric only matches if every metric property configured on the nameMatcher is a match.
func (m *nameMatcher) MatchMetric(metric pdata.Metric) (bool, error) {
	return m.nameFilters.Matches(metric.Name()), nil
}
