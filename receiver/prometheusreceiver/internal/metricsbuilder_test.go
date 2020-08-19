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

package internal

import (
	"reflect"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/scrape"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const startTs = int64(1555366610000)
const interval = int64(15 * 1000)
const defaultBuilderStartTime = float64(1.0)

var testMetadata = map[string]scrape.MetricMetadata{
	"counter_test":    {Metric: "counter_test", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"counter_test2":   {Metric: "counter_test2", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"gauge_test":      {Metric: "gauge_test", Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"gauge_test2":     {Metric: "gauge_test2", Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"hist_test":       {Metric: "hist_test", Type: textparse.MetricTypeHistogram, Help: "", Unit: ""},
	"hist_test2":      {Metric: "hist_test2", Type: textparse.MetricTypeHistogram, Help: "", Unit: ""},
	"ghist_test":      {Metric: "ghist_test", Type: textparse.MetricTypeGaugeHistogram, Help: "", Unit: ""},
	"summary_test":    {Metric: "summary_test", Type: textparse.MetricTypeSummary, Help: "", Unit: ""},
	"summary_test2":   {Metric: "summary_test2", Type: textparse.MetricTypeSummary, Help: "", Unit: ""},
	"unknown_test":    {Metric: "unknown_test", Type: textparse.MetricTypeUnknown, Help: "", Unit: ""},
	"poor_name_count": {Metric: "poor_name_count", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"up":              {Metric: "up", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"scrape_foo":      {Metric: "scrape_foo", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"example_process_start_time_seconds": {Metric: "example_process_start_time_seconds",
		Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"process_start_time_seconds": {Metric: "process_start_time_seconds",
		Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"badprocess_start_time_seconds": {Metric: "badprocess_start_time_seconds",
		Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
}

type testDataPoint struct {
	lb labels.Labels
	t  int64
	v  float64
}

type testScrapedPage struct {
	pts []*testDataPoint
}

type buildTestData struct {
	name   string
	inputs []*testScrapedPage
	wants  [][]*metricspb.Metric
}

func createLabels(mFamily string, tagPairs ...string) labels.Labels {
	lm := make(map[string]string)
	lm[model.MetricNameLabel] = mFamily
	if len(tagPairs)%2 != 0 {
		panic("tag pairs is not even")
	}

	for i := 0; i < len(tagPairs); i += 2 {
		lm[tagPairs[i]] = tagPairs[i+1]
	}

	return labels.FromMap(lm)
}

func createDataPoint(mname string, value float64, tagPairs ...string) *testDataPoint {
	return &testDataPoint{
		lb: createLabels(mname, tagPairs...),
		v:  value,
	}
}

func runBuilderTests(t *testing.T, tests []buildTestData) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, len(tt.wants), len(tt.inputs))
			mc := newMockMetadataCache(testMetadata)
			st := startTs
			for i, page := range tt.inputs {
				b := newMetricBuilder(mc, true, "", testLogger)
				b.startTime = defaultBuilderStartTime // set to a non-zero value
				for _, pt := range page.pts {
					// set ts for testing
					pt.t = st
					assert.NoError(t, b.AddDataPoint(pt.lb, pt.t, pt.v))
				}
				metrics, _, _, err := b.Build()
				assert.NoError(t, err)
				assert.EqualValues(t, tt.wants[i], metrics)
				st += interval
			}
		})
	}
}

func runBuilderStartTimeTests(t *testing.T, tests []buildTestData,
	startTimeMetricRegex string, expectedBuilderStartTime float64) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockMetadataCache(testMetadata)
			st := startTs
			for _, page := range tt.inputs {
				b := newMetricBuilder(mc, true, startTimeMetricRegex,
					testLogger)
				b.startTime = defaultBuilderStartTime // set to a non-zero value
				for _, pt := range page.pts {
					// set ts for testing
					pt.t = st
					assert.NoError(t, b.AddDataPoint(pt.lb, pt.t, pt.v))
				}
				_, _, _, err := b.Build()
				assert.NoError(t, err)
				assert.EqualValues(t, b.startTime, expectedBuilderStartTime)
				st += interval
			}
		})
	}
}

func Test_startTimeMetricMatch(t *testing.T) {
	matchBuilderStartTime := 123.456
	matchTests := []buildTestData{
		{
			name: "prefix_match",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("example_process_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
		{
			name: "match",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("process_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
	}
	nomatchTests := []buildTestData{
		{
			name: "nomatch1",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("_process_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
		{
			name: "nomatch2",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("subprocess_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
	}

	runBuilderStartTimeTests(t, matchTests, "^(.+_)*process_start_time_seconds$", matchBuilderStartTime)
	runBuilderStartTimeTests(t, nomatchTests, "^(.+_)*process_start_time_seconds$", defaultBuilderStartTime)
}

func Test_metricBuilder_counters(t *testing.T) {
	tests := []buildTestData{
		{
			name: "single-item",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("counter_test", 100, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "counter_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "two-items",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("counter_test", 150, "foo", "bar"),
						createDataPoint("counter_test", 25, "foo", "other"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "counter_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 150.0}},
								},
							},
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "other", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 25.0}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "two-metrics",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("counter_test", 150, "foo", "bar"),
						createDataPoint("counter_test", 25, "foo", "other"),
						createDataPoint("counter_test2", 100, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "counter_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 150.0}},
								},
							},
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "other", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 25.0}},
								},
							},
						},
					},
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "counter_test2",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "metrics-with-poor-names",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("poor_name_count", 100, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "poor_name_count",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
						},
					},
				},
			},
		},
	}

	runBuilderTests(t, tests)
}

func Test_metricBuilder_gauges(t *testing.T) {
	tests := []buildTestData{
		{
			name: "one-gauge",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("gauge_test", 100, "foo", "bar"),
					},
				},
				{
					pts: []*testDataPoint{
						createDataPoint("gauge_test", 90, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "gauge_test",
							Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
						},
					},
				},
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "gauge_test",
							Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs + interval), Value: &metricspb.Point_DoubleValue{DoubleValue: 90.0}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "gauge-with-different-tags",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("gauge_test", 100, "foo", "bar"),
						createDataPoint("gauge_test", 200, "bar", "foo"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "gauge_test",
							Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "bar"}, {Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
							{
								LabelValues: []*metricspb.LabelValue{{Value: "foo", HasValue: true}, {Value: "", HasValue: false}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 200.0}},
								},
							},
						},
					},
				},
			},
		},
		{
			// TODO: A decision need to be made. If we want to have the behavior which can generate different tag key
			//  sets because metrics come and go
			name: "gauge-comes-and-go-with-different-tagset",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("gauge_test", 100, "foo", "bar"),
						createDataPoint("gauge_test", 200, "bar", "foo"),
					},
				},
				{
					pts: []*testDataPoint{
						createDataPoint("gauge_test", 20, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "gauge_test",
							Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "bar"}, {Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
							{
								LabelValues: []*metricspb.LabelValue{{Value: "foo", HasValue: true}, {Value: "", HasValue: false}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 200.0}},
								},
							},
						},
					},
				},
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "gauge_test",
							Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs + interval), Value: &metricspb.Point_DoubleValue{DoubleValue: 20.0}},
								},
							},
						},
					},
				},
			},
		},
	}

	runBuilderTests(t, tests)
}

func Test_metricBuilder_untype(t *testing.T) {
	tests := []buildTestData{
		{
			name: "one-unknown",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("unknown_test", 100, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "unknown_test",
							Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no-type-hint",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("something_not_exists", 100, "foo", "bar"),
						createDataPoint("theother_not_exists", 200, "foo", "bar"),
						createDataPoint("theother_not_exists", 300, "bar", "foo"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "something_not_exists",
							Type:      metricspb.MetricDescriptor_UNSPECIFIED,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
						},
					},
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "theother_not_exists",
							Type:      metricspb.MetricDescriptor_UNSPECIFIED,
							LabelKeys: []*metricspb.LabelKey{{Key: "bar"}, {Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 200.0}},
								},
							},
							{
								LabelValues: []*metricspb.LabelValue{{Value: "foo", HasValue: true}, {Value: "", HasValue: false}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 300.0}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "untype-metric-poor-names",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("some_count", 100, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "some_count",
							Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								LabelValues: []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DoubleValue{DoubleValue: 100.0}},
								},
							},
						},
					},
				},
			},
		},
	}

	runBuilderTests(t, tests)
}

func Test_metricBuilder_histogram(t *testing.T) {
	tests := []buildTestData{
		{
			name: "single item",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 1, "foo", "bar", "le", "10"),
						createDataPoint("hist_test", 2, "foo", "bar", "le", "20"),
						createDataPoint("hist_test", 10, "foo", "bar", "le", "+inf"),
						createDataPoint("hist_test_sum", 99, "foo", "bar"),
						createDataPoint("hist_test_count", 10, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "hist_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{10, 20},
													},
												},
											},
											Count:   10,
											Sum:     99.0,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 1}, {Count: 8}},
										}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multi-groups",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 1, "foo", "bar", "le", "10"),
						createDataPoint("hist_test", 2, "foo", "bar", "le", "20"),
						createDataPoint("hist_test", 10, "foo", "bar", "le", "+inf"),
						createDataPoint("hist_test_sum", 99, "foo", "bar"),
						createDataPoint("hist_test_count", 10, "foo", "bar"),
						createDataPoint("hist_test", 1, "key2", "v2", "le", "10"),
						createDataPoint("hist_test", 2, "key2", "v2", "le", "20"),
						createDataPoint("hist_test", 3, "key2", "v2", "le", "+inf"),
						createDataPoint("hist_test_sum", 50, "key2", "v2"),
						createDataPoint("hist_test_count", 3, "key2", "v2"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "hist_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}, {Key: "key2"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}, {Value: "", HasValue: false}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{10, 20},
													},
												},
											},
											Count:   10,
											Sum:     99.0,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 1}, {Count: 8}},
										}}},
								},
							},
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "v2", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{10, 20},
													},
												},
											},
											Count:   3,
											Sum:     50.0,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 1}, {Count: 1}},
										}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multi-groups-and-families",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 1, "foo", "bar", "le", "10"),
						createDataPoint("hist_test", 2, "foo", "bar", "le", "20"),
						createDataPoint("hist_test", 10, "foo", "bar", "le", "+inf"),
						createDataPoint("hist_test_sum", 99, "foo", "bar"),
						createDataPoint("hist_test_count", 10, "foo", "bar"),
						createDataPoint("hist_test", 1, "key2", "v2", "le", "10"),
						createDataPoint("hist_test", 2, "key2", "v2", "le", "20"),
						createDataPoint("hist_test", 3, "key2", "v2", "le", "+inf"),
						createDataPoint("hist_test_sum", 50, "key2", "v2"),
						createDataPoint("hist_test_count", 3, "key2", "v2"),
						createDataPoint("hist_test2", 1, "le", "10"),
						createDataPoint("hist_test2", 2, "le", "20"),
						createDataPoint("hist_test2", 3, "le", "+inf"),
						createDataPoint("hist_test2_sum", 50),
						createDataPoint("hist_test2_count", 3),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "hist_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}, {Key: "key2"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}, {Value: "", HasValue: false}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{10, 20},
													},
												},
											},
											Count:   10,
											Sum:     99.0,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 1}, {Count: 8}},
										}}},
								},
							},
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "v2", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{10, 20},
													},
												},
											},
											Count:   3,
											Sum:     50.0,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 1}, {Count: 1}},
										}}},
								},
							},
						},
					},
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "hist_test2",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{10, 20},
													},
												},
											},
											Count:   3,
											Sum:     50.0,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 1}, {Count: 1}},
										}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "unordered-buckets",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 10, "foo", "bar", "le", "+inf"),
						createDataPoint("hist_test", 1, "foo", "bar", "le", "10"),
						createDataPoint("hist_test", 2, "foo", "bar", "le", "20"),
						createDataPoint("hist_test_sum", 99, "foo", "bar"),
						createDataPoint("hist_test_count", 10, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "hist_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{10, 20},
													},
												},
											},
											Count:   10,
											Sum:     99.0,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 1}, {Count: 8}},
										}}},
								},
							},
						},
					},
				},
			},
		},
		{
			// this won't likely happen in real env, as prometheus wont generate histogram with less than 3 buckets
			name: "only-one-bucket",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 3, "le", "+inf"),
						createDataPoint("hist_test_count", 3),
						createDataPoint("hist_test_sum", 100),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "hist_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{},
													},
												},
											},
											Count:   3,
											Sum:     100,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 3}},
										}}},
								},
							},
						},
					},
				},
			},
		},
		{
			// this won't likely happen in real env, as prometheus wont generate histogram with less than 3 buckets
			name: "only-one-bucket-noninf",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 3, "le", "20"),
						createDataPoint("hist_test_count", 3),
						createDataPoint("hist_test_sum", 100),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "hist_test",
							Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
							LabelKeys: []*metricspb.LabelKey{}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_DistributionValue{
										DistributionValue: &metricspb.DistributionValue{
											BucketOptions: &metricspb.DistributionValue_BucketOptions{
												Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
													Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
														Bounds: []float64{},
													},
												},
											},
											Count:   3,
											Sum:     100,
											Buckets: []*metricspb.DistributionValue_Bucket{{Count: 3}},
										}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "corrupted-no-buckets",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test_sum", 99),
						createDataPoint("hist_test_count", 10),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{},
			},
		},
		{
			name: "corrupted-no-sum",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 1, "foo", "bar", "le", "10"),
						createDataPoint("hist_test", 2, "foo", "bar", "le", "20"),
						createDataPoint("hist_test", 3, "foo", "bar", "le", "+inf"),
						createDataPoint("hist_test_count", 3),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{},
			},
		},
		{
			name: "corrupted-no-count",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 1, "foo", "bar", "le", "10"),
						createDataPoint("hist_test", 2, "foo", "bar", "le", "20"),
						createDataPoint("hist_test", 3, "foo", "bar", "le", "+inf"),
						createDataPoint("hist_test_sum", 99),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{},
			},
		},
	}

	runBuilderTests(t, tests)
}

func Test_metricBuilder_summary(t *testing.T) {
	tests := []buildTestData{
		{
			name: "no-sum-and-count",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{},
			},
		},
		{
			name: "empty-quantiles",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test_sum", 100, "foo", "bar"),
						createDataPoint("summary_test_count", 500, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "summary_test",
							Type:      metricspb.MetricDescriptor_SUMMARY,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{
										Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_SummaryValue{
											SummaryValue: &metricspb.SummaryValue{
												Sum:   &wrapperspb.DoubleValue{Value: 100.0},
												Count: &wrapperspb.Int64Value{Value: 500},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "regular-summary",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 1, "foo", "bar", "quantile", "0.5"),
						createDataPoint("summary_test", 2, "foo", "bar", "quantile", "0.75"),
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
						createDataPoint("summary_test_sum", 100, "foo", "bar"),
						createDataPoint("summary_test_count", 500, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "summary_test",
							Type:      metricspb.MetricDescriptor_SUMMARY,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_SummaryValue{
										SummaryValue: &metricspb.SummaryValue{
											Sum:   &wrapperspb.DoubleValue{Value: 100.0},
											Count: &wrapperspb.Int64Value{Value: 500},
											Snapshot: &metricspb.SummaryValue_Snapshot{
												PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
													{Percentile: 50.0, Value: 1},
													{Percentile: 75.0, Value: 2},
													{Percentile: 100.0, Value: 5},
												},
											}}}},
								},
							},
						},
					},
				},
			},
		},
	}

	runBuilderTests(t, tests)
}

func Test_metricBuilder_skipped(t *testing.T) {
	tests := []buildTestData{
		{
			name: "skip-internal-metrics",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("scrape_foo", 1),
						createDataPoint("up", 1.0),
					},
				},
				{
					pts: []*testDataPoint{
						createDataPoint("scrape_foo", 2),
						createDataPoint("up", 2.0),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{},
				{},
			},
		},
	}

	runBuilderTests(t, tests)
}

func Test_metricBuilder_baddata(t *testing.T) {
	t.Run("empty-metric-name", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilder(mc, true, "", testLogger)
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(labels.FromStrings("a", "b"), startTs, 123); err != errMetricNameNotFound {
			t.Error("expecting errMetricNameNotFound error, but get nil")
			return
		}

		if _, _, _, err := b.Build(); err != errNoDataToBuild {
			t.Error("expecting errNoDataToBuild error, but get nil")
		}
	})

	t.Run("histogram-datapoint-no-bucket-label", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilder(mc, true, "", testLogger)
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(createLabels("hist_test", "k", "v"), startTs, 123); err != errEmptyBoundaryLabel {
			t.Error("expecting errEmptyBoundaryLabel error, but get nil")
		}
	})

	t.Run("summary-datapoint-no-quantile-label", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilder(mc, true, "", testLogger)
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(createLabels("summary_test", "k", "v"), startTs, 123); err != errEmptyBoundaryLabel {
			t.Error("expecting errEmptyBoundaryLabel error, but get nil")
		}
	})

}

func Test_isUsefulLabel(t *testing.T) {
	type args struct {
		mType    metricspb.MetricDescriptor_Type
		labelKey string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"metricName", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.MetricNameLabel}, false},
		{"instance", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.InstanceLabel}, false},
		{"scheme", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.SchemeLabel}, false},
		{"metricPath", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.MetricsPathLabel}, false},
		{"job", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.JobLabel}, false},
		{"bucket", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.BucketLabel}, true},
		{"bucketForGaugeDistribution", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, model.BucketLabel}, false},
		{"bucketForCumulativeDistribution", args{metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION, model.BucketLabel}, false},
		{"Quantile", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.QuantileLabel}, true},
		{"QuantileForSummay", args{metricspb.MetricDescriptor_SUMMARY, model.QuantileLabel}, false},
		{"other", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, "other"}, true},
		{"empty", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUsefulLabel(tt.args.mType, tt.args.labelKey); got != tt.want {
				t.Errorf("isUsefulLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dpgSignature(t *testing.T) {
	knownLabelKeys := []string{"a", "b"}

	tests := []struct {
		name string
		ls   labels.Labels
		want string
	}{
		{"1st label", labels.FromStrings("a", "va"), `[]string{"a=va"}`},
		{"2nd label", labels.FromStrings("b", "vb"), `[]string{"b=vb"}`},
		{"two labels", labels.FromStrings("a", "va", "b", "vb"), `[]string{"a=va", "b=vb"}`},
		{"extra label", labels.FromStrings("a", "va", "b", "vb", "x", "xa"), `[]string{"a=va", "b=vb"}`},
		{"different order", labels.FromStrings("b", "vb", "a", "va"), `[]string{"a=va", "b=vb"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dpgSignature(knownLabelKeys, tt.ls); got != tt.want {
				t.Errorf("dpgSignature() = %v, want %v", got, tt.want)
			}
		})
	}

	// this is important for caching start values, as new metrics with new tag of a same group can come up in a 2nd run,
	// however, its order within the group is not predictable. we need to have a way to generate a stable key even if
	// the total number of keys changes in between different scrape runs
	t.Run("knownLabelKeys updated", func(t *testing.T) {
		ls := labels.FromStrings("a", "va")
		want := dpgSignature(knownLabelKeys, ls)
		got := dpgSignature(append(knownLabelKeys, "c"), ls)
		if got != want {
			t.Errorf("dpgSignature() = %v, want %v", got, want)
		}
	})
}

func Test_normalizeMetricName(t *testing.T) {
	tests := []struct {
		name  string
		mname string
		want  string
	}{
		{"normal", "normal", "normal"},
		{"count", "foo_count", "foo"},
		{"bucket", "foo_bucket", "foo"},
		{"sum", "foo_sum", "foo"},
		{"no_prefix", "_sum", "_sum"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeMetricName(tt.mname); got != tt.want {
				t.Errorf("normalizeMetricName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBoundary(t *testing.T) {
	ls := labels.FromStrings("le", "100.0", "foo", "bar", "quantile", "0.5")
	ls2 := labels.FromStrings("foo", "bar")
	ls3 := labels.FromStrings("le", "xyz", "foo", "bar", "quantile", "0.5")
	type args struct {
		metricType metricspb.MetricDescriptor_Type
		labels     labels.Labels
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{"histogram", args{metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION, ls}, 100.0, false},
		{"gaugehistogram", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, ls}, 100.0, false},
		{"gaugehistogram_no_label", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, ls2}, 0, true},
		{"gaugehistogram_bad_value", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, ls3}, 0, true},
		{"summary", args{metricspb.MetricDescriptor_SUMMARY, ls}, 0.5, false},
		{"otherType", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, ls}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBoundary(tt.args.metricType, tt.args.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("getBoundary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getBoundary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convToOCAMetricType(t *testing.T) {
	tests := []struct {
		name       string
		metricType textparse.MetricType
		want       metricspb.MetricDescriptor_Type
	}{
		{"counter", textparse.MetricTypeCounter, metricspb.MetricDescriptor_CUMULATIVE_DOUBLE},
		{"gauge", textparse.MetricTypeGauge, metricspb.MetricDescriptor_GAUGE_DOUBLE},
		{"histogram", textparse.MetricTypeHistogram, metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION},
		{"guageHistogram", textparse.MetricTypeGaugeHistogram, metricspb.MetricDescriptor_UNSPECIFIED},
		{"summary", textparse.MetricTypeSummary, metricspb.MetricDescriptor_SUMMARY},
		{"info", textparse.MetricTypeInfo, metricspb.MetricDescriptor_UNSPECIFIED},
		{"stateset", textparse.MetricTypeStateset, metricspb.MetricDescriptor_UNSPECIFIED},
		{"unknown", textparse.MetricTypeUnknown, metricspb.MetricDescriptor_GAUGE_DOUBLE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convToOCAMetricType(tt.metricType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convToOCAMetricType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_heuristicalMetricAndKnownUnits(t *testing.T) {
	tests := []struct {
		metricName string
		parsedUnit string
		want       string
	}{
		{"test", "ms", "ms"},
		{"millisecond", "", ""},
		{"test_millisecond", "", "ms"},
		{"test_milliseconds", "", "ms"},
		{"test_ms", "", "ms"},
		{"test_second", "", "s"},
		{"test_seconds", "", "s"},
		{"test_s", "", "s"},
		{"test_microsecond", "", "us"},
		{"test_microseconds", "", "us"},
		{"test_us", "", "us"},
		{"test_nanosecond", "", "ns"},
		{"test_nanoseconds", "", "ns"},
		{"test_ns", "", "ns"},
		{"test_byte", "", "By"},
		{"test_bytes", "", "By"},
		{"test_by", "", "By"},
		{"test_bit", "", "Bi"},
		{"test_bits", "", "Bi"},
		{"test_kilogram", "", "kg"},
		{"test_kilograms", "", "kg"},
		{"test_kg", "", "kg"},
		{"test_gram", "", "g"},
		{"test_grams", "", "g"},
		{"test_g", "", "g"},
		{"test_nanogram", "", "ng"},
		{"test_nanograms", "", "ng"},
		{"test_ng", "", "ng"},
		{"test_meter", "", "m"},
		{"test_meters", "", "m"},
		{"test_metre", "", "m"},
		{"test_metres", "", "m"},
		{"test_m", "", "m"},
		{"test_kilometer", "", "km"},
		{"test_kilometers", "", "km"},
		{"test_kilometre", "", "km"},
		{"test_kilometres", "", "km"},
		{"test_km", "", "km"},
		{"test_milimeter", "", "mm"},
		{"test_milimeters", "", "mm"},
		{"test_milimetre", "", "mm"},
		{"test_milimetres", "", "mm"},
		{"test_mm", "", "mm"},
	}
	for _, tt := range tests {
		t.Run(tt.metricName, func(t *testing.T) {
			if got := heuristicalMetricAndKnownUnits(tt.metricName, tt.parsedUnit); got != tt.want {
				t.Errorf("heuristicalMetricAndKnownUnits() = %v, want %v", got, tt.want)
			}
		})
	}
}
