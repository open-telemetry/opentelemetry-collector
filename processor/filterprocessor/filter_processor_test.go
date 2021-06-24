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

package filterprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
	"go.opentelemetry.io/collector/model/pdata"
)

type metricNameTest struct {
	name               string
	inc                *filtermetric.MatchProperties
	exc                *filtermetric.MatchProperties
	inMetrics          pdata.Metrics
	outMN              [][]string // output Metric names per Resource
	allMetricsFiltered bool
}

type metricWithResource struct {
	metricNames        []string
	resourceAttributes map[string]pdata.AttributeValue
}

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

	inMetricNames = []string{
		"full_name_match",
		"not_exact_string_match",
		"prefix/test/match",
		"prefix_test_match",
		"prefixprefix/test/match",
		"test/match/suffix",
		"test_match_suffix",
		"test/match/suffixsuffix",
		"test/contains/match",
		"test_contains_match",
		"random",
		"full/name/match",
		"full_name_match", // repeats
		"not_exact_string_match",
	}

	inMetricForResourceTest = []metricWithResource{
		{
			metricNames: []string{"metric1", "metric2"},
			resourceAttributes: map[string]pdata.AttributeValue{
				"attr1": pdata.NewAttributeValueString("attr1/val1"),
				"attr2": pdata.NewAttributeValueString("attr2/val2"),
				"attr3": pdata.NewAttributeValueString("attr3/val3"),
			},
		},
	}

	inMetricForTwoResource = []metricWithResource{
		{
			metricNames: []string{"metric1", "metric2"},
			resourceAttributes: map[string]pdata.AttributeValue{
				"attr1": pdata.NewAttributeValueString("attr1/val1"),
			},
		},
		{
			metricNames: []string{"metric3", "metric4"},
			resourceAttributes: map[string]pdata.AttributeValue{
				"attr1": pdata.NewAttributeValueString("attr1/val2"),
			},
		},
	}

	regexpMetricsFilterProperties = &filtermetric.MatchProperties{
		MatchType:   filtermetric.Regexp,
		MetricNames: validFilters,
	}

	standardTests = []metricNameTest{
		{
			name:      "includeFilter",
			inc:       regexpMetricsFilterProperties,
			inMetrics: testResourceMetrics([]metricWithResource{{metricNames: inMetricNames}}),
			outMN: [][]string{{
				"full_name_match",
				"prefix/test/match",
				"prefix_test_match",
				"prefixprefix/test/match",
				"test/match/suffix",
				"test_match_suffix",
				"test/match/suffixsuffix",
				"test/contains/match",
				"test_contains_match",
				"full/name/match",
				"full_name_match",
			}},
		},
		{
			name:      "excludeFilter",
			exc:       regexpMetricsFilterProperties,
			inMetrics: testResourceMetrics([]metricWithResource{{metricNames: inMetricNames}}),
			outMN: [][]string{{
				"not_exact_string_match",
				"random",
				"not_exact_string_match",
			}},
		},
		{
			name: "includeAndExclude",
			inc:  regexpMetricsFilterProperties,
			exc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
				MetricNames: []string{
					"prefix_test_match",
					"test_contains_match",
				},
			},
			inMetrics: testResourceMetrics([]metricWithResource{{metricNames: inMetricNames}}),
			outMN: [][]string{{
				"full_name_match",
				"prefix/test/match",
				// "prefix_test_match", excluded by exclude filter
				"prefixprefix/test/match",
				"test/match/suffix",
				"test_match_suffix",
				"test/match/suffixsuffix",
				"test/contains/match",
				// "test_contains_match", excluded by exclude filter
				"full/name/match",
				"full_name_match",
			}},
		},
		{
			name: "includeAndExcludeWithEmptyResourceMetrics",
			inc:  regexpMetricsFilterProperties,
			exc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
				MetricNames: []string{
					"prefix_test_match",
					"test_contains_match",
				},
			},
			inMetrics: testResourceMetrics([]metricWithResource{{}, {metricNames: inMetricNames}}),
			outMN: [][]string{
				{
					"full_name_match",
					"prefix/test/match",
					// "prefix_test_match", excluded by exclude filter
					"prefixprefix/test/match",
					"test/match/suffix",
					"test_match_suffix",
					"test/match/suffixsuffix",
					"test/contains/match",
					// "test_contains_match", excluded by exclude filter
					"full/name/match",
					"full_name_match",
				},
			},
		},
		{
			name: "emptyFilterInclude",
			inc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
			},
			inMetrics:          testResourceMetrics([]metricWithResource{{metricNames: inMetricNames}}),
			allMetricsFiltered: true,
		},
		{
			name: "emptyFilterExclude",
			exc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
			},
			inMetrics: testResourceMetrics([]metricWithResource{{metricNames: inMetricNames}}),
			outMN:     [][]string{inMetricNames},
		},
		{
			name:      "includeWithNilResourceAttributes",
			inc:       regexpMetricsFilterProperties,
			inMetrics: testResourceMetrics([]metricWithResource{{metricNames: inMetricNames}}),
			outMN: [][]string{{
				"full_name_match",
				"prefix/test/match",
				"prefix_test_match",
				"prefixprefix/test/match",
				"test/match/suffix",
				"test_match_suffix",
				"test/match/suffixsuffix",
				"test/contains/match",
				"test_contains_match",
				"full/name/match",
				"full_name_match",
			}},
		},
		{
			name: "excludeNilWithResourceAttributes",
			exc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
			},
			inMetrics: testResourceMetrics(inMetricForResourceTest),
			outMN: [][]string{
				{"metric1", "metric2"},
			},
		},
		{
			name: "includeAllWithResourceAttributes",
			inc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
				MetricNames: []string{
					"metric1",
					"metric2",
				},
				ResourceAttributes: []filterconfig.Attribute{{Key: "attr1", Value: "attr1/val1"}},
			},
			inMetrics: testResourceMetrics(inMetricForResourceTest),
			outMN: [][]string{
				{"metric1", "metric2"},
			},
		},
		{
			name: "includeAllWithMissingResourceAttributes",
			inc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
				MetricNames: []string{
					"metric1",
					"metric2",
					"metric3",
					"metric4",
				},
				ResourceAttributes: []filterconfig.Attribute{{Key: "attr1", Value: "attr1/val1"}},
			},
			inMetrics: testResourceMetrics(inMetricForTwoResource),
			outMN: [][]string{
				{"metric1", "metric2"},
			},
		},
		{
			name: "excludeAllWithMissingResourceAttributes",
			exc: &filtermetric.MatchProperties{
				MatchType:          filtermetric.Strict,
				ResourceAttributes: []filterconfig.Attribute{{Key: "attr1", Value: "attr1/val1"}},
			},
			inMetrics: testResourceMetrics(inMetricForTwoResource),
			outMN: [][]string{
				{"metric3", "metric4"},
			},
		},
		{
			name: "includeWithRegexResourceAttributes",
			inc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Regexp,
				MetricNames: []string{
					"metric1",
					"metric3",
				},
				ResourceAttributes: []filterconfig.Attribute{{Key: "attr1", Value: "(attr1/val1|attr1/val2)"}},
			},
			inMetrics: testResourceMetrics(inMetricForTwoResource),
			outMN: [][]string{
				{"metric1"},
				{"metric3"},
			},
		},
	}
)

func TestFilterMetricProcessor(t *testing.T) {
	for _, test := range standardTests {
		t.Run(test.name, func(t *testing.T) {
			// next stores the results of the filter metric processor
			next := new(consumertest.MetricsSink)
			cfg := &Config{
				ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
				Metrics: MetricFilters{
					Include: test.inc,
					Exclude: test.exc,
				},
			}
			factory := NewFactory()
			fmp, err := factory.CreateMetricsProcessor(
				context.Background(),
				componenttest.NewNopProcessorCreateSettings(),
				cfg,
				next,
			)
			assert.NotNil(t, fmp)
			assert.Nil(t, err)

			caps := fmp.Capabilities()
			assert.True(t, caps.MutatesData)
			ctx := context.Background()
			assert.NoError(t, fmp.Start(ctx, nil))

			cErr := fmp.ConsumeMetrics(context.Background(), test.inMetrics)
			assert.Nil(t, cErr)
			got := next.AllMetrics()

			if test.allMetricsFiltered {
				require.Equal(t, 0, len(got))
				return
			}

			require.Equal(t, 1, len(got))
			require.Equal(t, len(test.outMN), got[0].ResourceMetrics().Len())
			for i, wantOut := range test.outMN {
				gotMetrics := got[0].ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics()
				assert.Equal(t, len(wantOut), gotMetrics.Len())
				for idx := range wantOut {
					assert.Equal(t, wantOut[idx], gotMetrics.At(idx).Name())
				}
			}
			assert.NoError(t, fmp.Shutdown(ctx))
		})
	}
}

func testResourceMetrics(mwrs []metricWithResource) pdata.Metrics {
	md := pdata.NewMetrics()
	now := time.Now()

	for _, mwr := range mwrs {
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().InitFromMap(mwr.resourceAttributes)
		ms := rm.InstrumentationLibraryMetrics().AppendEmpty().Metrics()
		for _, name := range mwr.metricNames {
			m := ms.AppendEmpty()
			m.SetName(name)
			m.SetDataType(pdata.MetricDataTypeDoubleGauge)
			dp := m.DoubleGauge().DataPoints().AppendEmpty()
			dp.SetTimestamp(pdata.TimestampFromTime(now.Add(10 * time.Second)))
			dp.SetValue(123)
		}
	}
	return md
}

func BenchmarkStrictFilter(b *testing.B) {
	mp := &filtermetric.MatchProperties{
		MatchType:   "strict",
		MetricNames: []string{"p10_metric_0"},
	}
	benchmarkFilter(b, mp)
}

func BenchmarkRegexpFilter(b *testing.B) {
	mp := &filtermetric.MatchProperties{
		MatchType:   "regexp",
		MetricNames: []string{"p10_metric_0"},
	}
	benchmarkFilter(b, mp)
}

func BenchmarkExprFilter(b *testing.B) {
	mp := &filtermetric.MatchProperties{
		MatchType:   "expr",
		Expressions: []string{`MetricName == "p10_metric_0"`},
	}
	benchmarkFilter(b, mp)
}

func benchmarkFilter(b *testing.B, mp *filtermetric.MatchProperties) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	pcfg := cfg.(*Config)
	pcfg.Metrics = MetricFilters{
		Exclude: mp,
	}
	ctx := context.Background()
	proc, _ := factory.CreateMetricsProcessor(
		ctx,
		componenttest.NewNopProcessorCreateSettings(),
		cfg,
		consumertest.NewNop(),
	)
	pdms := metricSlice(128)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pdm := range pdms {
			_ = proc.ConsumeMetrics(ctx, pdm)
		}
	}
}

func metricSlice(numMetrics int) []pdata.Metrics {
	var out []pdata.Metrics
	for i := 0; i < numMetrics; i++ {
		const size = 2
		out = append(out, pdm(fmt.Sprintf("p%d_", i), size))
	}
	return out
}

func pdm(prefix string, size int) pdata.Metrics {
	c := goldendataset.MetricsCfg{
		MetricDescriptorType: pdata.MetricDataTypeIntGauge,
		MetricNamePrefix:     prefix,
		NumILMPerResource:    size,
		NumMetricsPerILM:     size,
		NumPtLabels:          size,
		NumPtsPerMetric:      size,
		NumResourceAttrs:     size,
		NumResourceMetrics:   size,
	}
	return goldendataset.MetricsFromCfg(c)
}

func TestNilResourceMetrics(t *testing.T) {
	metrics := pdata.NewMetrics()
	rms := metrics.ResourceMetrics()
	rms.AppendEmpty()
	requireNotPanics(t, metrics)
}

func TestNilILM(t *testing.T) {
	metrics := pdata.NewMetrics()
	rms := metrics.ResourceMetrics()
	rm := rms.AppendEmpty()
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.AppendEmpty()
	requireNotPanics(t, metrics)
}

func TestNilMetric(t *testing.T) {
	metrics := pdata.NewMetrics()
	rms := metrics.ResourceMetrics()
	rm := rms.AppendEmpty()
	ilms := rm.InstrumentationLibraryMetrics()
	ilm := ilms.AppendEmpty()
	ms := ilm.Metrics()
	ms.AppendEmpty()
	requireNotPanics(t, metrics)
}

func requireNotPanics(t *testing.T, metrics pdata.Metrics) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	pcfg := cfg.(*Config)
	pcfg.Metrics = MetricFilters{
		Exclude: &filtermetric.MatchProperties{
			MatchType:   "strict",
			MetricNames: []string{"foo"},
		},
	}
	ctx := context.Background()
	proc, _ := factory.CreateMetricsProcessor(
		ctx,
		componenttest.NewNopProcessorCreateSettings(),
		cfg,
		consumertest.NewNop(),
	)
	require.NotPanics(t, func() {
		_ = proc.ConsumeMetrics(ctx, metrics)
	})
}
