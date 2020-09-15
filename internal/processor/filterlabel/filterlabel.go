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

package filterlabel

import metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"

// Matcher tests a metric for a match by three levels: metric name, label key,
// and label value. Matches can be strict or regexp for any of the search
// strings and multiple searches may be supplied to test against at any level.
type Matcher struct {
	nodes []node
}

// NewMatcher creates a Matcher struct. Pass in one or more nested Config
// structs. Each top-level struct matches against a metric name, and below that
// against label keys then label values.
func NewMatcher(cfgs ...Config) (*Matcher, error) {
	m := &Matcher{}
	for _, c := range cfgs {
		n, err := newNode(c)
		if err != nil {
			return nil, err
		}
		m.nodes = append(m.nodes, n)
	}
	return m, nil
}

// MatchMetric takes a metric and returns true if there is a match along any of
// the Config graphs supplied to the constructor.
func (m *Matcher) MatchMetric(metric *metricspb.Metric) bool {
	for _, n := range m.nodes {
		if n.match(metric.MetricDescriptor.Name) && matchLblKeys(n.next(), metric) {
			return true
		}
	}
	return false
}

func matchLblKeys(nodes []node, metric *metricspb.Metric) bool {
	if nodes == nil {
		return true
	}
	for i, lblKey := range metric.MetricDescriptor.LabelKeys {
		for _, n := range nodes {
			if n.match(lblKey.Key) && matchLblVals(n.next(), metric, i) {
				return true
			}
		}
	}
	return false
}

func matchLblVals(nodes []node, metric *metricspb.Metric, lblNum int) bool {
	if nodes == nil {
		return true
	}
	for _, n := range nodes {
		for _, ts := range metric.Timeseries {
			lblVal := ts.LabelValues[lblNum]
			if lblVal.HasValue && n.match(lblVal.Value) {
				return true
			}
		}
	}
	return false
}
