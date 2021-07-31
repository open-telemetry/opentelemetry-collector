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
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestGetBoundaryEquivalence(t *testing.T) {
	cases := []struct {
		name      string
		mtype     metricspb.MetricDescriptor_Type
		pmtype    pdata.MetricDataType
		labels    labels.Labels
		wantValue float64
		wantErr   string
	}{
		{
			name:   "cumulative histogram with bucket label",
			mtype:  metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
			pmtype: pdata.MetricDataTypeHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "0.256"},
			},
			wantValue: 0.256,
		},
		{
			name:   "gauge histogram with bucket label",
			mtype:  metricspb.MetricDescriptor_GAUGE_DISTRIBUTION,
			pmtype: pdata.MetricDataTypeHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantValue: 11.71,
		},
		{
			name:   "summary with bucket label",
			mtype:  metricspb.MetricDescriptor_SUMMARY,
			pmtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:   "summary with quantile label",
			mtype:  metricspb.MetricDescriptor_SUMMARY,
			pmtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.QuantileLabel, Value: "92.88"},
			},
			wantValue: 92.88,
		},
		{
			name:   "gauge histogram mismatched with bucket label",
			mtype:  metricspb.MetricDescriptor_SUMMARY,
			pmtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:   "other data types without matches",
			mtype:  metricspb.MetricDescriptor_GAUGE_DOUBLE,
			pmtype: pdata.MetricDataTypeGauge,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "given metricType has no BucketLabel or QuantileLabel",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			oldBoundary, oerr := getBoundary(tt.mtype, tt.labels)
			pdataBoundary, perr := getBoundaryPdata(tt.pmtype, tt.labels)
			assert.Equal(t, oldBoundary, pdataBoundary, "Both boundary values MUST be equal")
			assert.Equal(t, oldBoundary, tt.wantValue, "Mismatched boundary messages")
			assert.Equal(t, oerr, perr, "The exact same error MUST be returned from both boundary helpers")

			if tt.wantErr != "" {
				require.NotEqual(t, oerr, "expected an error from old style boundary retrieval")
				require.NotEqual(t, perr, "expected an error from new style boundary retrieval")
				require.Contains(t, oerr.Error(), tt.wantErr)
				require.Contains(t, perr.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetBoundaryPdata(t *testing.T) {
	tests := []struct {
		name      string
		mtype     pdata.MetricDataType
		labels    labels.Labels
		wantValue float64
		wantErr   string
	}{
		{
			name:  "cumulative histogram with bucket label",
			mtype: pdata.MetricDataTypeHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "0.256"},
			},
			wantValue: 0.256,
		},
		{
			name:  "gauge histogram with bucket label",
			mtype: pdata.MetricDataTypeHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantValue: 11.71,
		},
		{
			name:  "summary with bucket label",
			mtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:  "summary with quantile label",
			mtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.QuantileLabel, Value: "92.88"},
			},
			wantValue: 92.88,
		},
		{
			name:  "gauge histogram mismatched with bucket label",
			mtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:  "other data types without matches",
			mtype: pdata.MetricDataTypeGauge,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "given metricType has no BucketLabel or QuantileLabel",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			value, err := getBoundaryPdata(tt.mtype, tt.labels)
			if tt.wantErr != "" {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.Nil(t, err)
			require.Equal(t, value, tt.wantValue)
		})
	}
}

func TestConvToPdataMetricType(t *testing.T) {
	tests := []struct {
		name  string
		mtype textparse.MetricType
		want  pdata.MetricDataType
	}{
		{
			name:  "textparse.counter",
			mtype: textparse.MetricTypeCounter,
			want:  pdata.MetricDataTypeSum,
		},
		{
			name:  "textparse.gauge",
			mtype: textparse.MetricTypeGauge,
			want:  pdata.MetricDataTypeGauge,
		},
		{
			name:  "textparse.unknown",
			mtype: textparse.MetricTypeUnknown,
			want:  pdata.MetricDataTypeGauge,
		},
		{
			name:  "textparse.histogram",
			mtype: textparse.MetricTypeHistogram,
			want:  pdata.MetricDataTypeHistogram,
		},
		{
			name:  "textparse.summary",
			mtype: textparse.MetricTypeSummary,
			want:  pdata.MetricDataTypeSummary,
		},
		{
			name:  "textparse.metric_type_info",
			mtype: textparse.MetricTypeInfo,
			want:  pdata.MetricDataTypeNone,
		},
		{
			name:  "textparse.metric_state_set",
			mtype: textparse.MetricTypeStateset,
			want:  pdata.MetricDataTypeNone,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := convToPdataMetricType(tt.mtype)
			require.Equal(t, got.String(), tt.want.String())
		})
	}
}

func TestIsUsefulLabelPdata(t *testing.T) {
	tests := []struct {
		name      string
		mtypes    []pdata.MetricDataType
		labelKeys []string
		want      bool
	}{
		{
			name: `unuseful "metric","instance","scheme","path","job" with any kind`,
			labelKeys: []string{
				model.MetricNameLabel, model.InstanceLabel, model.SchemeLabel, model.MetricsPathLabel, model.JobLabel,
			},
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeHistogram,
				pdata.MetricDataTypeSummary,
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeSum,
			},
			want: false,
		},
		{
			name: `bucket label with non "int_histogram", "histogram":: useful`,
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeSummary,
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeSum,
			},
			labelKeys: []string{model.BucketLabel},
			want:      true,
		},
		{
			name: `quantile label with "summary": non-useful`,
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeSummary,
			},
			labelKeys: []string{model.QuantileLabel},
			want:      false,
		},
		{
			name:      `quantile label with non-"summary": useful`,
			labelKeys: []string{model.QuantileLabel},
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeHistogram,
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeSum,
			},
			want: true,
		},
		{
			name:      `any other label with any type:: useful`,
			labelKeys: []string{"any_label", "foo.bar"},
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeHistogram,
				pdata.MetricDataTypeSummary,
				pdata.MetricDataTypeSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeGauge,
				pdata.MetricDataTypeSum,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			for _, mtype := range tt.mtypes {
				for _, labelKey := range tt.labelKeys {
					got := isUsefulLabelPdata(mtype, labelKey)
					assert.Equal(t, got, tt.want)
				}
			}
		})
	}
}

type buildTestDataPdata struct {
	name   string
	inputs []*testScrapedPage
	wants  func() []*pdata.MetricSlice
}

func Test_OTLPMetricBuilder_counters(t *testing.T) {
	tests := []buildTestDataPdata{
		{
			name: "single-item",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("counter_test", 100, "foo", "bar"),
					},
				},
			},
			wants: func() []*pdata.MetricSlice {
				mL := pdata.NewMetricSlice()
				m0 := mL.AppendEmpty()
				m0.SetName("counter_test")
				m0.SetDataType(pdata.MetricDataTypeSum)
				sum := m0.Sum()
				pt0 := sum.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL}
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
			wants: func() []*pdata.MetricSlice {
				mL := pdata.NewMetricSlice()
				m0 := mL.AppendEmpty()
				m0.SetName("counter_test")
				m0.SetDataType(pdata.MetricDataTypeSum)
				sum := m0.Sum()
				pt0 := sum.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(150.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				pt1 := sum.DataPoints().AppendEmpty()
				pt1.SetDoubleVal(25.0)
				pt1.SetStartTimestamp(0)
				pt1.SetTimestamp(startTsNanos)
				pt1.LabelsMap().Insert("foo", "other")

				return []*pdata.MetricSlice{&mL}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("counter_test")
				m0.SetDataType(pdata.MetricDataTypeSum)
				sum0 := m0.Sum()
				pt0 := sum0.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(150.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				pt1 := sum0.DataPoints().AppendEmpty()
				pt1.SetDoubleVal(25.0)
				pt1.SetStartTimestamp(0)
				pt1.SetTimestamp(startTsNanos)
				pt1.LabelsMap().Insert("foo", "other")

				m1 := mL0.AppendEmpty()
				m1.SetName("counter_test2")
				m1.SetDataType(pdata.MetricDataTypeSum)
				sum1 := m1.Sum()
				pt2 := sum1.DataPoints().AppendEmpty()
				pt2.SetDoubleVal(100.0)
				pt2.SetStartTimestamp(0)
				pt2.SetTimestamp(startTsNanos)
				pt2.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL := pdata.NewMetricSlice()
				m0 := mL.AppendEmpty()
				m0.SetName("poor_name_count")
				m0.SetDataType(pdata.MetricDataTypeSum)
				sum := m0.Sum()
				pt0 := sum.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL}
			},
		},
	}

	runBuilderTestsPdata(t, tests)
}

func runBuilderTestsPdata(t *testing.T, tests []buildTestDataPdata) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wants := tt.wants()
			assert.EqualValues(t, len(wants), len(tt.inputs))
			mc := newMockMetadataCache(testMetadata)
			st := startTs
			for i, page := range tt.inputs {
				b := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())
				b.startTime = defaultBuilderStartTime // set to a non-zero value
				for _, pt := range page.pts {
					// set ts for testing
					pt.t = st
					assert.NoError(t, b.AddDataPoint(pt.lb, pt.t, pt.v))
				}
				metrics, _, _, err := b.Build()
				assert.NoError(t, err)
				assert.EqualValues(t, wants[i], metrics)
				st += interval
			}
		})
	}
}

var (
	startTsNanos             = pdata.Timestamp(startTs * 1e6)
	startTsPlusIntervalNanos = pdata.Timestamp((startTs + interval) * 1e6)
)

func Test_OTLPMetricBuilder_gauges(t *testing.T) {
	tests := []buildTestDataPdata{
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("gauge_test")
				m0.SetDataType(pdata.MetricDataTypeGauge)
				gauge0 := m0.Gauge()
				pt0 := gauge0.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				mL1 := pdata.NewMetricSlice()
				m1 := mL1.AppendEmpty()
				m1.SetName("gauge_test")
				m1.SetDataType(pdata.MetricDataTypeGauge)
				gauge1 := m1.Gauge()
				pt1 := gauge1.DataPoints().AppendEmpty()
				pt1.SetDoubleVal(90.0)
				pt1.SetStartTimestamp(0)
				pt1.SetTimestamp(startTsPlusIntervalNanos)
				pt1.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0, &mL1}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("gauge_test")
				m0.SetDataType(pdata.MetricDataTypeGauge)
				gauge0 := m0.Gauge()
				pt0 := gauge0.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("bar", "")
				pt0.LabelsMap().Insert("foo", "bar")

				pt1 := gauge0.DataPoints().AppendEmpty()
				pt1.SetDoubleVal(200.0)
				pt1.SetStartTimestamp(0)
				pt1.SetTimestamp(startTsNanos)
				pt1.LabelsMap().Insert("bar", "foo")
				pt1.LabelsMap().Insert("foo", "")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("gauge_test")
				m0.SetDataType(pdata.MetricDataTypeGauge)
				gauge0 := m0.Gauge()
				pt0 := gauge0.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("bar", "")
				pt0.LabelsMap().Insert("foo", "bar")

				pt1 := gauge0.DataPoints().AppendEmpty()
				pt1.SetDoubleVal(200.0)
				pt1.SetStartTimestamp(0)
				pt1.SetTimestamp(startTsNanos)
				pt1.LabelsMap().Insert("bar", "foo")
				pt1.LabelsMap().Insert("foo", "")

				mL1 := pdata.NewMetricSlice()
				m1 := mL1.AppendEmpty()
				m1.SetName("gauge_test")
				m1.SetDataType(pdata.MetricDataTypeGauge)
				gauge1 := m1.Gauge()
				pt2 := gauge1.DataPoints().AppendEmpty()
				pt2.SetDoubleVal(20.0)
				pt2.SetStartTimestamp(0)
				pt2.SetTimestamp(startTsPlusIntervalNanos)
				pt2.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0, &mL1}
			},
		},
	}

	runBuilderTestsPdata(t, tests)
}

func Test_OTLPMetricBuilder_untype(t *testing.T) {
	tests := []buildTestDataPdata{
		{
			name: "one-unknown",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("unknown_test", 100, "foo", "bar"),
					},
				},
			},
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("unknown_test")
				m0.SetDataType(pdata.MetricDataTypeGauge)
				gauge0 := m0.Gauge()
				pt0 := gauge0.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetStartTimestamp(0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("something_not_exists")
				m0.SetDataType(pdata.MetricDataTypeGauge)
				gauge0 := m0.Gauge()
				pt0 := gauge0.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				m1 := mL0.AppendEmpty()
				m1.SetName("theother_not_exists")
				m1.SetDataType(pdata.MetricDataTypeGauge)
				gauge1 := m1.Gauge()
				pt1 := gauge1.DataPoints().AppendEmpty()
				pt1.SetDoubleVal(200.0)
				pt1.SetTimestamp(startTsNanos)
				pt1.LabelsMap().Insert("bar", "")
				pt1.LabelsMap().Insert("foo", "bar")

				pt2 := gauge1.DataPoints().AppendEmpty()
				pt2.SetDoubleVal(300.0)
				pt2.SetTimestamp(startTsNanos)
				pt2.LabelsMap().Insert("bar", "foo")
				pt2.LabelsMap().Insert("foo", "")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("some_count")
				m0.SetDataType(pdata.MetricDataTypeGauge)
				gauge0 := m0.Gauge()
				pt0 := gauge0.DataPoints().AppendEmpty()
				pt0.SetDoubleVal(100.0)
				pt0.SetTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0}
			},
		},
	}

	runBuilderTestsPdata(t, tests)
}

func Test_OTLPMetricBuilder_histogram(t *testing.T) {
	tests := []buildTestDataPdata{
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("hist_test")
				m0.SetDataType(pdata.MetricDataTypeHistogram)
				hist0 := m0.Histogram()
				pt0 := hist0.DataPoints().AppendEmpty()
				pt0.SetCount(10)
				pt0.SetSum(99)
				pt0.SetExplicitBounds([]float64{10, 20})
				pt0.SetBucketCounts([]uint64{1, 1, 8})
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("hist_test")
				m0.SetDataType(pdata.MetricDataTypeHistogram)
				hist0 := m0.Histogram()
				pt0 := hist0.DataPoints().AppendEmpty()
				pt0.SetCount(10)
				pt0.SetSum(99)
				pt0.SetExplicitBounds([]float64{10, 20})
				pt0.SetBucketCounts([]uint64{1, 1, 8})
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")
				pt0.LabelsMap().Insert("key2", "")

				pt1 := hist0.DataPoints().AppendEmpty()
				pt1.SetCount(3)
				pt1.SetSum(50)
				pt1.SetExplicitBounds([]float64{10, 20})
				pt1.SetBucketCounts([]uint64{1, 1, 1})
				pt1.SetTimestamp(startTsNanos)
				pt1.SetStartTimestamp(startTsNanos)
				pt1.LabelsMap().Insert("foo", "")
				pt1.LabelsMap().Insert("key2", "v2")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("hist_test")
				m0.SetDataType(pdata.MetricDataTypeHistogram)
				hist0 := m0.Histogram()
				pt0 := hist0.DataPoints().AppendEmpty()
				pt0.SetCount(10)
				pt0.SetSum(99)
				pt0.SetExplicitBounds([]float64{10, 20})
				pt0.SetBucketCounts([]uint64{1, 1, 8})
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")
				pt0.LabelsMap().Insert("key2", "")

				pt1 := hist0.DataPoints().AppendEmpty()
				pt1.SetCount(3)
				pt1.SetSum(50)
				pt1.SetExplicitBounds([]float64{10, 20})
				pt1.SetBucketCounts([]uint64{1, 1, 1})
				pt1.SetTimestamp(startTsNanos)
				pt1.SetStartTimestamp(startTsNanos)
				pt1.LabelsMap().Insert("foo", "")
				pt1.LabelsMap().Insert("key2", "v2")

				m1 := mL0.AppendEmpty()
				m1.SetName("hist_test2")
				m1.SetDataType(pdata.MetricDataTypeHistogram)
				hist1 := m1.Histogram()
				pt2 := hist1.DataPoints().AppendEmpty()
				pt2.SetCount(3)
				pt2.SetSum(50)
				pt2.SetExplicitBounds([]float64{10, 20})
				pt2.SetBucketCounts([]uint64{1, 1, 1})
				pt2.SetTimestamp(startTsNanos)
				pt2.SetStartTimestamp(startTsNanos)

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("hist_test")
				m0.SetDataType(pdata.MetricDataTypeHistogram)
				hist0 := m0.Histogram()
				pt0 := hist0.DataPoints().AppendEmpty()
				pt0.SetCount(10)
				pt0.SetSum(99)
				pt0.SetExplicitBounds([]float64{10, 20})
				pt0.SetBucketCounts([]uint64{1, 1, 8})
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("hist_test")
				m0.SetDataType(pdata.MetricDataTypeHistogram)
				hist0 := m0.Histogram()
				pt0 := hist0.DataPoints().AppendEmpty()
				pt0.SetCount(3)
				pt0.SetSum(100)
				pt0.SetExplicitBounds([]float64{})
				pt0.SetBucketCounts([]uint64{3})
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("hist_test")
				m0.SetDataType(pdata.MetricDataTypeHistogram)
				hist0 := m0.Histogram()
				pt0 := hist0.DataPoints().AppendEmpty()
				pt0.SetCount(3)
				pt0.SetSum(100)
				pt0.SetExplicitBounds([]float64{})
				pt0.SetBucketCounts([]uint64{3})
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)

				return []*pdata.MetricSlice{&mL0}
			},
		},
		{
			name: "no-sum",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("hist_test", 1, "foo", "bar", "le", "10"),
						createDataPoint("hist_test", 2, "foo", "bar", "le", "20"),
						createDataPoint("hist_test", 3, "foo", "bar", "le", "+inf"),
						createDataPoint("hist_test_count", 3, "foo", "bar"),
					},
				},
			},
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("hist_test")
				m0.SetDataType(pdata.MetricDataTypeHistogram)
				hist0 := m0.Histogram()
				pt0 := hist0.DataPoints().AppendEmpty()
				pt0.SetCount(3)
				pt0.SetSum(0)
				pt0.SetExplicitBounds([]float64{10, 20})
				pt0.SetBucketCounts([]uint64{1, 1, 1})
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				return []*pdata.MetricSlice{&mL0}
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
						createDataPoint("hist_test_sum", 99, "foo", "bar"),
					},
				},
			},
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				return []*pdata.MetricSlice{&mL0}
			},
		},
	}

	runBuilderTestsPdata(t, tests)
}

func Test_metricBuilder_summary(t *testing.T) {
	tests := []buildTestDataPdata{
		{
			name: "no-sum-and-count",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
					},
				},
			},
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				return []*pdata.MetricSlice{&mL0}
			},
		},
		{
			name: "no-count",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 1, "foo", "bar", "quantile", "0.5"),
						createDataPoint("summary_test", 2, "foo", "bar", "quantile", "0.75"),
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
						createDataPoint("summary_test_sum", 500, "foo", "bar"),
					},
				},
			},
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				return []*pdata.MetricSlice{&mL0}
			},
		},
		{
			name: "no-sum",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 1, "foo", "bar", "quantile", "0.5"),
						createDataPoint("summary_test", 2, "foo", "bar", "quantile", "0.75"),
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
						createDataPoint("summary_test_count", 500, "foo", "bar"),
					},
				},
			},
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("summary_test")
				m0.SetDataType(pdata.MetricDataTypeSummary)
				sum0 := m0.Summary()
				pt0 := sum0.DataPoints().AppendEmpty()
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.SetCount(500)
				pt0.SetSum(0.0)
				pt0.LabelsMap().Insert("foo", "bar")
				qvL := pt0.QuantileValues()
				q50 := qvL.AppendEmpty()
				q50.SetQuantile(50)
				q50.SetValue(1.0)
				q75 := qvL.AppendEmpty()
				q75.SetQuantile(75)
				q75.SetValue(2.0)
				q100 := qvL.AppendEmpty()
				q100.SetQuantile(100)
				q100.SetValue(5.0)

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("summary_test")
				m0.SetDataType(pdata.MetricDataTypeSummary)
				sum0 := m0.Summary()
				pt0 := sum0.DataPoints().AppendEmpty()
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.SetCount(500)
				pt0.SetSum(100.0)
				pt0.LabelsMap().Insert("foo", "bar")

				return []*pdata.MetricSlice{&mL0}
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
			wants: func() []*pdata.MetricSlice {
				mL0 := pdata.NewMetricSlice()
				m0 := mL0.AppendEmpty()
				m0.SetName("summary_test")
				m0.SetDataType(pdata.MetricDataTypeSummary)
				sum0 := m0.Summary()
				pt0 := sum0.DataPoints().AppendEmpty()
				pt0.SetTimestamp(startTsNanos)
				pt0.SetStartTimestamp(startTsNanos)
				pt0.SetCount(500)
				pt0.SetSum(100.0)
				pt0.LabelsMap().Insert("foo", "bar")
				qvL := pt0.QuantileValues()
				q50 := qvL.AppendEmpty()
				q50.SetQuantile(50)
				q50.SetValue(1.0)
				q75 := qvL.AppendEmpty()
				q75.SetQuantile(75)
				q75.SetValue(2.0)
				q100 := qvL.AppendEmpty()
				q100.SetQuantile(100)
				q100.SetValue(5.0)

				return []*pdata.MetricSlice{&mL0}
			},
		},
	}

	runBuilderTestsPdata(t, tests)
}

// Ensure that we reject duplicate label keys. See https://github.com/open-telemetry/wg-prometheus/issues/44.
func TestMetricBuilderDuplicateLabelKeysAreRejected(t *testing.T) {
	mc := newMockMetadataCache(testMetadata)
	mb := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())

	dupLabels := labels.Labels{
		{Name: "__name__", Value: "test"},
		{Name: "a", Value: "1"},
		{Name: "a", Value: "1"},
		{Name: "z", Value: "9"},
		{Name: "z", Value: "1"},
		{Name: "instance", Value: "0.0.0.0:8855"},
		{Name: "job", Value: "test"},
	}

	err := mb.AddDataPoint(dupLabels, 1917, 1.0)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), `invalid sample: non-unique label names: ["a" "z"]`)
}

func Test_metricBuilder_baddata(t *testing.T) {
	t.Run("empty-metric-name", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())
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
		b := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(createLabels("hist_test", "k", "v"), startTs, 123); err != errEmptyBoundaryLabel {
			t.Error("expecting errEmptyBoundaryLabel error, but get nil")
		}
	})

	t.Run("summary-datapoint-no-quantile-label", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(createLabels("summary_test", "k", "v"), startTs, 123); err != errEmptyBoundaryLabel {
			t.Error("expecting errEmptyBoundaryLabel error, but get nil")
		}
	})
}
