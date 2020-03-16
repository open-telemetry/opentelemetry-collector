// Copyright 2020 OpenTelemetry Authors
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

package metric

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset/factory"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
)

var (
	regexpFilters = []string{
		"prefix/.*",
		"prefix_.*",
		".*/suffix",
		".*_suffix",
		".*/contains/.*",
		".*_contains_.*",
		"full/name/match",
		"full_name_match",
	}

	strictFilters = []string{
		"exact_string_match",
		".*/suffix",
		"(a|b)",
	}
)

func TestMatcherMatches(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *MatchProperties
		metric      *metricspb.Metric
		shouldMatch bool
	}{
		{
			name:        "regexpNameMatch",
			cfg:         createConfig(regexpFilters, factory.REGEXP),
			metric:      createMetric("test/match/suffix"),
			shouldMatch: true,
		}, {
			name:        "regexpNameMisatch",
			cfg:         createConfig(regexpFilters, factory.REGEXP),
			metric:      createMetric("test/match/wrongsuffix"),
			shouldMatch: false,
		}, {
			name:        "strictNameMatch",
			cfg:         createConfig(strictFilters, factory.STRICT),
			metric:      createMetric("exact_string_match"),
			shouldMatch: true,
		}, {
			name:        "strictNameMismatch",
			cfg:         createConfig(regexpFilters, factory.REGEXP),
			metric:      createMetric("wrong_string_match"),
			shouldMatch: false,
		}, {
			name:        "matcherWithNoPropertyFilters",
			cfg:         createConfig([]string{}, factory.STRICT),
			metric:      createMetric("metric"),
			shouldMatch: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matcher, err := NewMatcher(test.cfg)
			assert.NotNil(t, matcher)
			assert.Nil(t, err)

			assert.Equal(t, test.shouldMatch, matcher.MatchMetric(test.metric))
		})
	}
}
