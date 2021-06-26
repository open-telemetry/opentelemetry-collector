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
	"go.opentelemetry.io/collector/model/pdata"
)

type Matcher interface {
	MatchMetric(metric pdata.Metric) (bool, error)
}

// NewMatcher constructs a metric Matcher. If an 'expr' match type is specified,
// returns an expr matcher, otherwise a name matcher.
func NewMatcher(config *MatchProperties) (Matcher, error) {
	if config.MatchType == Expr {
		return newExprMatcher(config.Expressions)
	}
	return newNameMatcher(config)
}
