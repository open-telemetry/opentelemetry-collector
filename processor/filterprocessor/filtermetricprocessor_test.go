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

package filterprocessor

import (
	"context"
	"log"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
	ptest "github.com/open-telemetry/opentelemetry-collector/processor/processortest"
)

var (
	validFilters = []string{
		"prefix/.*",
		"prefix_.*",
		".*/suffix",
		".*_suffix",
		".*/contains/.*",
		".*_contains_.*",
		"full/name/match",
		"full_name_match",
	}
)

func TestFilterMetricProcessor(t *testing.T) {
	inMetricNames := []string{
		"full_name_match",
		"not_exact_string_match",
		"prefix/test/match",
		"prefix_test_match",
		"wrongprefix/test/match",
		"test/match/suffix",
		"test_match_suffix",
		"test/match/suffixwrong",
		"test/contains/match",
		"test_contains_match",
		"random",
		"full/name/match",
		"full_name_match", // repeats
		"not_exact_string_match",
	}

	tests := []struct {
		name  string
		cfg   *Config
		inMN  []string // input Metric names
		outMN []string // output Metric names
	}{
		{
			name: "includeFilter",
			cfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Action: INCLUDE,
				Metrics: MetricFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
						Filters:    validFilters,
					},
				},
			},
			inMN: inMetricNames,
			outMN: []string{
				"full_name_match",
				"prefix/test/match",
				"prefix_test_match",
				"test/match/suffix",
				"test_match_suffix",
				"test/contains/match",
				"test_contains_match",
				"full/name/match",
				"full_name_match",
			},
		}, {
			name: "excludeFilter",
			cfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Action: EXCLUDE,
				Metrics: MetricFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
						Filters:    validFilters,
					},
				},
			},
			inMN: inMetricNames,
			outMN: []string{
				"not_exact_string_match",
				"wrongprefix/test/match",
				"test/match/suffixwrong",
				"random",
				"not_exact_string_match",
			},
		}, {
			name: "emptyFilterInclude",
			cfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Action: INCLUDE,
				Metrics: MetricFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
					},
				},
			},
			inMN:  inMetricNames,
			outMN: []string{},
		}, {
			name: "emptyFilterExclude",
			cfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Action: EXCLUDE,
				Metrics: MetricFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
					},
				},
			},
			inMN:  inMetricNames,
			outMN: inMetricNames,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// next stores the results of the filter metric processor
			next := &ptest.CachingMetricsConsumer{}
			fmp, err := newFilterMetricProcessor(next, test.cfg)
			assert.NotNil(t, fmp)
			assert.Nil(t, err)

			md := consumerdata.MetricsData{
				Metrics: make([]*metricspb.Metric, len(test.inMN)),
			}

			for idx, in := range test.inMN {
				md.Metrics[idx] = &metricspb.Metric{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name: in,
					},
				}
			}

			cErr := fmp.ConsumeMetricsData(context.Background(), md)
			assert.Nil(t, cErr)

			assert.Equal(t, len(test.outMN), len(next.Data.Metrics))
			for idx, out := range next.Data.Metrics {
				assert.Equal(t, test.outMN[idx], out.MetricDescriptor.Name)
			}
		})
	}
}
