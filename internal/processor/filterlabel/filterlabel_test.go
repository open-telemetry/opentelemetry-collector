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

import (
	"fmt"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/crossdock/crossdock-go/require"
)

func TestEmptyMatchType(t *testing.T) {
	matcher, err := NewMatcher(Config{next: []Config{{}}})
	require.Nil(t, matcher)
	require.Error(t, err, "matchType cannot be empty")
}

func TestInvalidMatchType(t *testing.T) {
	matcher, err := NewMatcher(Config{matchType: "foo"})
	require.Nil(t, matcher)
	require.Error(t, err, "invalid matchType %q", "foo")
}

func TestMatchMetric_Strict(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-0-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Strict_MetricOnly(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Strict_MetricLblOnly(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Strict_LblValWrongPos(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-2",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-0-3", // lbl val exists but in wrong position
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(4, 4)
	matches := matcher.MatchMetric(m)
	require.False(t, matches)
}

func TestMatchMetric_Strict_MismatchedKey(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "foo",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-1-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(4, 4)
	matches := matcher.MatchMetric(m)
	require.False(t, matches)
}

func TestMatchMetric_Strict_MismatchedMetric(t *testing.T) {
	cfg := Config{
		str:       "foo",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-1-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.False(t, matches)
}

func TestMatchMetric_Strict_MultiMetricCfg(t *testing.T) {
	cfg := []Config{{
		str:       "foo",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-1-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}, {
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-1-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}}
	matcher := testMatcher(t, cfg...)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Strict_MultiKeyCfg(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "foo",
				matchType: MatchTypeStrict,
			}},
		}, {
			str:       "k-2",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-2-2",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(4, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Strict_MultiLblCfg(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "foo",
				matchType: MatchTypeStrict,
			}, {
				str:       "v-1-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(4, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Regexp_Metric(t *testing.T) {
	cfg := Config{
		str:       ".*",
		matchType: MatchTypeRegexp,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-0-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Regexp_LblKey(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-[0-9]",
			matchType: MatchTypeRegexp,
			next: []Config{{
				str:       "v-0-1",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func TestMatchMetric_Regexp_LblVal(t *testing.T) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-1",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-[0-9]-1",
				matchType: MatchTypeRegexp,
			}},
		}},
	}
	matcher := testMatcher(t, cfg)
	m := testMetrics(2, 4)
	matches := matcher.MatchMetric(m)
	require.True(t, matches)
}

func BenchmarkStrictMatch32MetricsBestCase(b *testing.B) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-0",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-0-0",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher, _ := NewMatcher(cfg)
	m := testMetrics(32, 32)
	for i := 0; i < b.N; i++ {
		_ = matcher.MatchMetric(m)
	}
}

func BenchmarkStrictMatch32MetricsWorstCase(b *testing.B) {
	cfg := Config{
		str:       "metric.name",
		matchType: MatchTypeStrict,
		next: []Config{{
			str:       "k-31",
			matchType: MatchTypeStrict,
			next: []Config{{
				str:       "v-31-31",
				matchType: MatchTypeStrict,
			}},
		}},
	}
	matcher, _ := NewMatcher(cfg)
	m := testMetrics(32, 32)
	for i := 0; i < b.N; i++ {
		matched := matcher.MatchMetric(m)
		require.True(b, matched)
	}
}

func BenchmarkRegexpMatch32Metrics(b *testing.B) {
	cfg := Config{
		str:       ".*\\w$",
		matchType: MatchTypeRegexp,
		next: []Config{{
			str:       ".*\\w.*\\D.*3.*1.*",
			matchType: MatchTypeRegexp,
			next: []Config{{
				str:       ".*\\w.*\\D.*3.*1.*\\D.*3.*1.*",
				matchType: MatchTypeRegexp,
			}},
		}},
	}
	matcher, _ := NewMatcher(cfg)
	m := testMetrics(32, 32)
	for i := 0; i < b.N; i++ {
		matched := matcher.MatchMetric(m)
		require.True(b, matched)
	}
}

func testMatcher(t *testing.T, cfg ...Config) *Matcher {
	matcher, err := NewMatcher(cfg...)
	require.NoError(t, err)
	return matcher
}

func testMetrics(numTS int, numLbls int) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name:      "metric.name",
			LabelKeys: testLblk(numLbls),
		},
		Timeseries: testTs(numTS, numLbls),
	}
}

func testTs(numTs int, numValues int) []*metricspb.TimeSeries {
	var out []*metricspb.TimeSeries
	for i := 0; i < numTs; i++ {
		out = append(out, &metricspb.TimeSeries{LabelValues: testLblv(i, numValues)})
	}
	return out
}

func testLblk(numLbls int) []*metricspb.LabelKey {
	var out []*metricspb.LabelKey
	for i := 0; i < numLbls; i++ {
		out = append(out, &metricspb.LabelKey{Key: fmt.Sprintf("k-%d", i)})
	}
	return out
}

func testLblv(index int, numValues int) []*metricspb.LabelValue {
	var out []*metricspb.LabelValue
	for i := 0; i < numValues; i++ {
		out = append(out, &metricspb.LabelValue{
			Value:    fmt.Sprintf("v-%d-%d", index, i),
			HasValue: true,
		})
	}
	return out
}
