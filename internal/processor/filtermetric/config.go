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

package filtermetric

import (
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/internal/processor/filterset/regexp"
)

// MatchType specifies the strategy for matching against `pdata.Metric`s. This
// is distinct from filterset.MatchType which matches against metric (and
// tracing) names only. To support matching against metric names and
// `pdata.Metric`s, filtermetric.MatchType is effectively a superset of
// filterset.MatchType.
type MatchType string

// These are the MatchTypes that users can specify for filtering
// `pdata.Metric`s.
const (
	Regexp           = MatchType(filterset.Regexp)
	Strict           = MatchType(filterset.Strict)
	Expr   MatchType = "expr"
)

// MatchProperties specifies the set of properties in a metric to match against and the
// type of string pattern matching to use.
type MatchProperties struct {
	// MatchType specifies the type of matching desired
	MatchType MatchType `mapstructure:"match_type"`
	// RegexpConfig specifies options for the Regexp match type
	RegexpConfig *regexp.Config `mapstructure:"regexp"`

	// MetricNames specifies the list of string patterns to match metric names against.
	// A match occurs if the metric name matches at least one string pattern in this list.
	MetricNames []string `mapstructure:"metric_names"`

	// Expressions specifies the list of expr expressions to match metrics against.
	// A match occurs if any datapoint in a metric matches at least one expression in this list.
	Expressions []string `mapstructure:"expressions"`
}
