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

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
	"go.opentelemetry.io/collector/translator/internaldata"
)

type metricNameTest struct {
	name               string
	inc                *filtermetric.MatchProperties
	exc                *filtermetric.MatchProperties
	inMN               [][]*metricspb.Metric // input Metric batches
	outMN              [][]string            // output Metric names
	allMetricsFiltered bool
}

type metricWithResource struct {
	metricNames []string
	resource    *resourcepb.Resource
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
			resource: &resourcepb.Resource{
				Labels: map[string]string{"attr1": "attr1/val1", "attr2": "attr2/val2", "attr3": "attr3/val3"},
			},
		},
	}

	inMetricForTwoResource = []metricWithResource{
		{
			metricNames: []string{"metric1", "metric2"},
			resource: &resourcepb.Resource{
				Labels: map[string]string{"attr1": "attr1/val1"},
			},
		},
		{
			metricNames: []string{"metric3", "metric4"},
			resource: &resourcepb.Resource{
				Labels: map[string]string{"attr1": "attr1/val2"},
			},
		},
	}

	regexpMetricsFilterProperties = &filtermetric.MatchProperties{
		MatchType:   filtermetric.Regexp,
		MetricNames: validFilters,
	}

	standardTests = []metricNameTest{
		{
			name: "includeFilter",
			inc:  regexpMetricsFilterProperties,
			inMN: [][]*metricspb.Metric{metricsWithName(inMetricNames)},
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
			name: "excludeFilter",
			exc:  regexpMetricsFilterProperties,
			inMN: [][]*metricspb.Metric{metricsWithName(inMetricNames)},
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
			inMN: [][]*metricspb.Metric{metricsWithName(inMetricNames)},
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
			name: "includeAndExcludeWithEmptyAndNil",
			inc:  regexpMetricsFilterProperties,
			exc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
				MetricNames: []string{
					"prefix_test_match",
					"test_contains_match",
				},
			},
			inMN: [][]*metricspb.Metric{nil, metricsWithName(inMetricNames), {}},
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
			inMN:               [][]*metricspb.Metric{metricsWithName(inMetricNames)},
			allMetricsFiltered: true,
		},
		{
			name: "emptyFilterExclude",
			exc: &filtermetric.MatchProperties{
				MatchType: filtermetric.Strict,
			},
			inMN:  [][]*metricspb.Metric{metricsWithName(inMetricNames)},
			outMN: [][]string{inMetricNames},
		},
		{
			name: "includeWithNilResourceAttributes",
			inc:  regexpMetricsFilterProperties,
			inMN: [][]*metricspb.Metric{metricsWithNameAndResource([]metricWithResource{{metricNames: inMetricNames, resource: nil}})},
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
			inMN: [][]*metricspb.Metric{metricsWithNameAndResource(inMetricForResourceTest)},
			outMN: [][]string{
				{"metric1"},
				{"metric2"},
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
			inMN: [][]*metricspb.Metric{metricsWithNameAndResource(inMetricForResourceTest)},
			outMN: [][]string{
				{"metric1"},
				{"metric2"},
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
			inMN: [][]*metricspb.Metric{metricsWithNameAndResource(inMetricForTwoResource)},
			outMN: [][]string{
				{"metric1"},
				{"metric2"},
			},
		},
		{
			name: "excludeAllWithMissingResourceAttributes",
			exc: &filtermetric.MatchProperties{
				MatchType:          filtermetric.Strict,
				ResourceAttributes: []filterconfig.Attribute{{Key: "attr1", Value: "attr1/val1"}},
			},
			inMN: [][]*metricspb.Metric{metricsWithNameAndResource(inMetricForTwoResource)},
			outMN: [][]string{
				{"metric3"},
				{"metric4"},
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
			inMN: [][]*metricspb.Metric{metricsWithNameAndResource(inMetricForTwoResource)},
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
				ProcessorSettings: configmodels.ProcessorSettings{
					TypeVal: typeStr,
					NameVal: typeStr,
				},
				Metrics: MetricFilters{
					Include: test.inc,
					Exclude: test.exc,
				},
			}
			factory := NewFactory()
			fmp, err := factory.CreateMetricsProcessor(
				context.Background(),
				component.ProcessorCreateParams{
					Logger: zap.NewNop(),
				},
				cfg,
				next,
			)
			assert.NotNil(t, fmp)
			assert.Nil(t, err)

			caps := fmp.GetCapabilities()
			assert.False(t, caps.MutatesConsumedData)
			ctx := context.Background()
			assert.NoError(t, fmp.Start(ctx, nil))

			mds := make([]internaldata.MetricsData, len(test.inMN))
			for i, metrics := range test.inMN {
				mds[i] = internaldata.MetricsData{
					Metrics: metrics,
				}
			}
			cErr := fmp.ConsumeMetrics(context.Background(), internaldata.OCSliceToMetrics(mds))
			assert.Nil(t, cErr)
			got := next.AllMetrics()

			if test.allMetricsFiltered {
				require.Equal(t, 0, len(got))
				return
			}

			require.Equal(t, 1, len(got))
			gotMD := internaldata.MetricsToOC(got[0])
			require.Equal(t, len(test.outMN), len(gotMD))
			for i, wantOut := range test.outMN {
				assert.Equal(t, len(wantOut), len(gotMD[i].Metrics))
				for idx, out := range gotMD[i].Metrics {
					assert.Equal(t, wantOut[idx], out.MetricDescriptor.Name)
				}
			}
			assert.NoError(t, fmp.Shutdown(ctx))
		})
	}
}

func metricsWithName(names []string) []*metricspb.Metric {
	ret := make([]*metricspb.Metric, len(names))
	now := time.Now()
	for i, name := range names {
		ret[i] = &metricspb.Metric{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name: name,
				Type: metricspb.MetricDescriptor_GAUGE_INT64,
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					Points: []*metricspb.Point{
						{
							Timestamp: timestamppb.New(now.Add(10 * time.Second)),
							Value: &metricspb.Point_Int64Value{
								Int64Value: int64(123),
							},
						},
					},
				},
			},
		}
	}
	return ret
}

func metricsWithNameAndResource(metrics []metricWithResource) []*metricspb.Metric {
	var md []*metricspb.Metric
	now := time.Now()

	for _, m := range metrics {
		names := m.metricNames
		ret := make([]*metricspb.Metric, len(names))

		for i, name := range names {
			ret[i] = &metricspb.Metric{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name: name,
					Type: metricspb.MetricDescriptor_GAUGE_INT64,
				},
				Resource: m.resource,
				Timeseries: []*metricspb.TimeSeries{
					{
						Points: []*metricspb.Point{
							{
								Timestamp: timestamppb.New(now.Add(10 * time.Second)),
								Value: &metricspb.Point_Int64Value{
									Int64Value: int64(123),
								},
							},
						},
					},
				},
			}
			md = append(md, ret[i])
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
		component.ProcessorCreateParams{},
		cfg,
		consumertest.NewMetricsNop(),
	)
	pdms := metricSlice(128)
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
	c := goldendataset.MetricCfg{
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

func TestMetricIndexSingle(t *testing.T) {
	metrics := pdm("", 1)
	idx := newMetricIndex()
	idx.add(0, 0, 0)
	extracted := idx.extract(metrics)
	require.Equal(t, metrics, extracted)
}

func TestMetricIndexAll(t *testing.T) {
	metrics := pdm("", 2)
	idx := newMetricIndex()
	idx.add(0, 0, 0)
	idx.add(0, 0, 1)
	idx.add(0, 1, 0)
	idx.add(0, 1, 1)
	idx.add(1, 0, 0)
	idx.add(1, 0, 1)
	idx.add(1, 1, 0)
	idx.add(1, 1, 1)
	extracted := idx.extract(metrics)
	require.Equal(t, metrics, extracted)
}

func TestNilResourceMetrics(t *testing.T) {
	metrics := pdata.NewMetrics()
	rms := metrics.ResourceMetrics()
	rms.Append(pdata.NewResourceMetrics())
	requireNotPanics(t, metrics)
}

func TestNilILM(t *testing.T) {
	metrics := pdata.NewMetrics()
	rms := metrics.ResourceMetrics()
	rm := pdata.NewResourceMetrics()
	rms.Append(rm)
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Append(pdata.NewInstrumentationLibraryMetrics())
	requireNotPanics(t, metrics)
}

func TestNilMetric(t *testing.T) {
	metrics := pdata.NewMetrics()
	rms := metrics.ResourceMetrics()
	rm := pdata.NewResourceMetrics()
	rms.Append(rm)
	ilms := rm.InstrumentationLibraryMetrics()
	ilm := pdata.NewInstrumentationLibraryMetrics()
	ilms.Append(ilm)
	ms := ilm.Metrics()
	ms.Append(pdata.NewMetric())
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
		component.ProcessorCreateParams{
			Logger: zap.NewNop(),
		},
		cfg,
		consumertest.NewMetricsNop(),
	)
	require.NotPanics(t, func() {
		_ = proc.ConsumeMetrics(ctx, metrics)
	})
}
