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

package filterprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
)

// TestLoadingConfigRegexp tests loading testdata/config_strict.yaml
func TestLoadingConfigStrict(t *testing.T) {
	// list of filters used repeatedly on testdata/config_strict.yaml
	testDataFilters := []string{
		"hello_world",
		"hello/world",
	}

	testDataMetricFilter := MetricFilter{
		NameFilter: filterset.FilterConfig{
			FilterType: filterset.STRICT,
			Filters:    testDataFilters,
		},
	}

	testDataTraceFilter := TraceFilter{
		NameFilter: filterset.FilterConfig{
			FilterType: filterset.STRICT,
			Filters:    testDataFilters,
		},
	}

	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Processors[typeStr] = factory
	config, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config_strict.yaml"), factories)

	assert.Nil(t, err)
	require.NotNil(t, config)

	tests := []struct {
		filterName string
		expCfg     *Config
	}{
		{
			filterName: "filter",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: typeStr,
					TypeVal: typeStr,
				},
			},
		}, {
			filterName: "filter/include",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/include",
					TypeVal: typeStr,
				},
				Action:  INCLUDE,
				Metrics: testDataMetricFilter,
				Traces:  testDataTraceFilter,
			},
		}, {
			filterName: "filter/exclude",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/exclude",
					TypeVal: typeStr,
				},
				Action:  EXCLUDE,
				Metrics: testDataMetricFilter,
				Traces:  testDataTraceFilter,
			},
		}, {
			filterName: "filter/metricsonly",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/metricsonly",
					TypeVal: typeStr,
				},
				Action:  INCLUDE,
				Metrics: testDataMetricFilter,
			},
		}, {
			filterName: "filter/tracesonly",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/tracesonly",
					TypeVal: typeStr,
				},
				Action: EXCLUDE,
				Traces: testDataTraceFilter,
			},
		}, {
			filterName: "filter/strictmatchtypedefault",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/strictmatchtypedefault",
					TypeVal: typeStr,
				},
				Action: EXCLUDE,
				Metrics: MetricFilter{
					NameFilter: filterset.FilterConfig{
						Filters: testDataFilters,
					},
				},
				Traces: TraceFilter{
					NameFilter: filterset.FilterConfig{
						Filters: testDataFilters,
					},
				},
			},
		}, {
			filterName: "filter/strictconfig",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/strictconfig",
					TypeVal: typeStr,
				},
				Action:  INCLUDE,
				Metrics: testDataMetricFilter,
				Traces:  testDataTraceFilter,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.filterName, func(t *testing.T) {
			cfg := config.Processors[test.filterName]
			assert.Equal(t, test.expCfg, cfg)
		})
	}
}

// TestLoadingConfigRegexp tests loading testdata/config_regexp.yaml
func TestLoadingConfigRegexp(t *testing.T) {
	// list of filters used repeatedly on testdata/config.yaml
	testDataFilters := []string{
		"prefix/.*",
		"prefix_.*",
		".*/suffix",
		".*_suffix",
		".*/contains/.*",
		".*_contains_.*",
		"full/name/match",
		"full_name_match",
	}

	testDataMetricFilter := MetricFilter{
		NameFilter: filterset.FilterConfig{
			FilterType: filterset.REGEXP,
			Filters:    testDataFilters,
		},
	}

	testDataTraceFilter := TraceFilter{
		NameFilter: filterset.FilterConfig{
			FilterType: filterset.REGEXP,
			Filters:    testDataFilters,
		},
	}

	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Processors[typeStr] = factory
	config, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config_regexp.yaml"), factories)

	assert.Nil(t, err)
	require.NotNil(t, config)

	tests := []struct {
		filterName string
		expCfg     *Config
	}{
		{
			filterName: "filter",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: typeStr,
					TypeVal: typeStr,
				},
			},
		}, {
			filterName: "filter/include",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/include",
					TypeVal: typeStr,
				},
				Action:  INCLUDE,
				Metrics: testDataMetricFilter,
				Traces:  testDataTraceFilter,
			},
		}, {
			filterName: "filter/exclude",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/exclude",
					TypeVal: typeStr,
				},
				Action:  EXCLUDE,
				Metrics: testDataMetricFilter,
				Traces:  testDataTraceFilter,
			},
		}, {
			filterName: "filter/metricsonly",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/metricsonly",
					TypeVal: typeStr,
				},
				Action:  INCLUDE,
				Metrics: testDataMetricFilter,
			},
		}, {
			filterName: "filter/tracesonly",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/tracesonly",
					TypeVal: typeStr,
				},
				Action: EXCLUDE,
				Traces: testDataTraceFilter,
			},
		}, {
			filterName: "filter/unlimitedcache",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/unlimitedcache",
					TypeVal: typeStr,
				},
				Action: INCLUDE,
				Metrics: MetricFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
						Filters:    testDataFilters,
						Regexp: &filterset.RegexpConfig{
							CacheEnabled: true,
						},
					},
				},
				Traces: TraceFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
						Filters:    testDataFilters,
						Regexp: &filterset.RegexpConfig{
							CacheEnabled: true,
						},
					},
				},
			},
		}, {
			filterName: "filter/limitedcache",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/limitedcache",
					TypeVal: typeStr,
				},
				Action: EXCLUDE,
				Metrics: MetricFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
						Filters:    testDataFilters,
						Regexp: &filterset.RegexpConfig{
							CacheEnabled: true,
							CacheSize:    10,
						},
					},
				},
				Traces: TraceFilter{
					NameFilter: filterset.FilterConfig{
						FilterType: filterset.REGEXP,
						Filters:    testDataFilters,
						Regexp: &filterset.RegexpConfig{
							CacheEnabled: true,
							CacheSize:    10,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.filterName, func(t *testing.T) {
			cfg := config.Processors[test.filterName]
			assert.Equal(t, test.expCfg, cfg)
		})
	}
}
