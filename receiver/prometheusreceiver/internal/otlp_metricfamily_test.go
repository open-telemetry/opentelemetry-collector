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
	"fmt"
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/scrape"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/model/pdata"
)

type byLookupMetadataCache map[string]scrape.MetricMetadata

func (bmc byLookupMetadataCache) Metadata(familyName string) (scrape.MetricMetadata, bool) {
	lookup, ok := bmc[familyName]
	return lookup, ok
}

func (bmc byLookupMetadataCache) SharedLabels() labels.Labels {
	return nil
}

var mc = byLookupMetadataCache{
	"counter": scrape.MetricMetadata{
		Metric: "cr",
		Type:   textparse.MetricTypeCounter,
		Help:   "This is some help",
		Unit:   "By",
	},
	"gauge": scrape.MetricMetadata{
		Metric: "ge",
		Type:   textparse.MetricTypeGauge,
		Help:   "This is some help",
		Unit:   "1",
	},
	"gaugehistogram": scrape.MetricMetadata{
		Metric: "gh",
		Type:   textparse.MetricTypeGaugeHistogram,
		Help:   "This is some help",
		Unit:   "?",
	},
	"histogram": scrape.MetricMetadata{
		Metric: "hg",
		Type:   textparse.MetricTypeHistogram,
		Help:   "This is some help",
		Unit:   "ms",
	},
	"summary": scrape.MetricMetadata{
		Metric: "s",
		Type:   textparse.MetricTypeSummary,
		Help:   "This is some help",
		Unit:   "ms",
	},
	"unknown": scrape.MetricMetadata{
		Metric: "u",
		Type:   textparse.MetricTypeUnknown,
		Help:   "This is some help",
		Unit:   "?",
	},
}

func TestIsCumulativeEquivalence(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "counter", want: true},
		{name: "gauge", want: false},
		{name: "histogram", want: true},
		{name: "gaugehistogram", want: false},
		{name: "does not exist", want: false},
		{name: "unknown", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mf := newMetricFamily(tt.name, mc, zap.NewNop(), 1).(*metricFamily)
			mfp := newMetricFamilyPdata(tt.name, mc, 1).(*metricFamilyPdata)
			assert.Equal(t, mf.isCumulativeType(), mfp.isCumulativeTypePdata(), "mismatch in isCumulative")
			assert.Equal(t, mf.isCumulativeType(), tt.want, "isCumulative does not match for regular metricFamily")
			assert.Equal(t, mfp.isCumulativeTypePdata(), tt.want, "isCumulative does not match for pdata metricFamily")
		})
	}
}

func TestMetricGroupData_toDistributionUnitTest(t *testing.T) {
	type scrape struct {
		at     int64
		value  float64
		metric string
	}
	tests := []struct {
		name                string
		metricName          string
		labels              labels.Labels
		scrapes             []*scrape
		want                func() pdata.HistogramDataPoint
		intervalStartTimeMs int64
	}{
		{
			name:                "histogram with startTimestamp of 11",
			metricName:          "histogram",
			intervalStartTimeMs: 1717,
			labels:              labels.Labels{{Name: "a", Value: "A"}, {Name: "le", Value: "0.75"}, {Name: "b", Value: "B"}},
			scrapes: []*scrape{
				{at: 11, value: 10, metric: "histogram_count"},
				{at: 11, value: 1004.78, metric: "histogram_sum"},
				{at: 13, value: 33.7, metric: "value"},
			},
			want: func() pdata.HistogramDataPoint {
				point := pdata.NewHistogramDataPoint()
				point.SetCount(10)
				point.SetSum(1004.78)
				point.SetTimestamp(11 * 1e6) // the time in milliseconds -> nanoseconds.
				point.SetBucketCounts([]uint64{33})
				point.SetExplicitBounds([]float64{})
				point.SetStartTimestamp(11 * 1e6)
				labelsMap := point.LabelsMap()
				labelsMap.Insert("a", "A")
				labelsMap.Insert("b", "B")
				return point
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mp := newMetricFamilyPdata(tt.metricName, mc, tt.intervalStartTimeMs).(*metricFamilyPdata)
			for _, tv := range tt.scrapes {
				require.NoError(t, mp.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
			}

			require.Equal(t, 1, len(mp.groups), "Expecting exactly 1 groupKey")
			groupKey := mp.getGroupKey(tt.labels.Copy())
			require.NotNil(t, mp.groups[groupKey], "Expecting the groupKey to have a value given key:: "+groupKey)

			hdpL := pdata.NewHistogramDataPointSlice()
			require.True(t, mp.groups[groupKey].toDistributionPoint(mp.labelKeysOrdered, &hdpL))
			require.Equal(t, 1, hdpL.Len(), "Exactly one point expected")
			got := hdpL.At(0)
			want := tt.want()
			require.Equal(t, want, got, "Expected the points to be equal")
		})
	}
}

func TestMetricGroupData_toDistributionPointEquivalence(t *testing.T) {
	type scrape struct {
		at     int64
		value  float64
		metric string
	}
	tests := []struct {
		name    string
		labels  labels.Labels
		scrapes []*scrape
	}{
		{
			name:   "histogram",
			labels: labels.Labels{{Name: "a", Value: "A"}, {Name: "le", Value: "0.75"}, {Name: "b", Value: "B"}},
			scrapes: []*scrape{
				{at: 11, value: 10, metric: "histogram_count"},
				{at: 11, value: 1004.78, metric: "histogram_sum"},
				{at: 13, value: 33.7, metric: "value"},
			},
		},
	}

	for i, tt := range tests {
		tt := tt
		intervalStartTimeMs := int64(i + 1)
		t.Run(tt.name, func(t *testing.T) {
			mf := newMetricFamily(tt.name, mc, zap.NewNop(), intervalStartTimeMs).(*metricFamily)
			mp := newMetricFamilyPdata(tt.name, mc, intervalStartTimeMs).(*metricFamilyPdata)
			for _, tv := range tt.scrapes {
				require.NoError(t, mp.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
				require.NoError(t, mf.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
			}
			groupKey := mf.getGroupKey(tt.labels.Copy())
			ocTimeseries := mf.groups[groupKey].toDistributionTimeSeries(mf.labelKeysOrdered)
			hdpL := pdata.NewHistogramDataPointSlice()
			require.True(t, mp.groups[groupKey].toDistributionPoint(mp.labelKeysOrdered, &hdpL))
			require.Equal(t, len(ocTimeseries.Points), hdpL.Len(), "They should have the exact same number of points")
			require.Equal(t, 1, hdpL.Len(), "Exactly one point expected")
			ocPoint := ocTimeseries.Points[0]
			pdataPoint := hdpL.At(0)
			// 1. Ensure that the startTimestamps are equal.
			require.Equal(t, ocTimeseries.GetStartTimestamp().AsTime(), pdataPoint.Timestamp().AsTime(), "The timestamp must be equal")
			// 2. Ensure that the count is equal.
			ocHistogram := ocPoint.GetDistributionValue()
			require.Equal(t, ocHistogram.GetCount(), int64(pdataPoint.Count()), "Count must be equal")
			// 3. Ensure that the sum is equal.
			require.Equal(t, ocHistogram.GetSum(), pdataPoint.Sum(), "Sum must be equal")
			// 4. Ensure that the point's timestamp is equal to that from the OpenCensusProto data point.
			require.Equal(t, ocPoint.GetTimestamp().AsTime(), pdataPoint.Timestamp().AsTime(), "Point timestamps must be equal")
			// 5. Ensure that bucket bounds are the same.
			require.Equal(t, len(ocHistogram.GetBuckets()), len(pdataPoint.BucketCounts()), "Bucket counts must have the same length")
			var ocBucketCounts []uint64
			for i, bucket := range ocHistogram.GetBuckets() {
				ocBucketCounts = append(ocBucketCounts, uint64(bucket.GetCount()))

				// 6. Ensure that the exemplars match.
				ocExemplar := bucket.Exemplar
				if ocExemplar == nil {
					if i >= pdataPoint.Exemplars().Len() { // Both have the exact same number of exemplars.
						continue
					}
					// Otherwise an exemplar is present for the pdata data point but not for the OpenCensus Proto histogram.
					t.Fatalf("Exemplar #%d is ONLY present in the pdata point but not in the OpenCensus Proto histogram", i)
				}
				pdataExemplar := pdataPoint.Exemplars().At(i)
				msgPrefix := fmt.Sprintf("Exemplar #%d:: ", i)
				require.Equal(t, ocExemplar.Timestamp.AsTime(), pdataExemplar.Timestamp().AsTime(), msgPrefix+"timestamp mismatch")
				require.Equal(t, ocExemplar.Value, pdataExemplar.DoubleVal(), msgPrefix+"value mismatch")
				pdataExemplarAttachments := make(map[string]string)
				pdataExemplar.FilteredLabels().Range(func(key, value string) bool {
					pdataExemplarAttachments[key] = value
					return true
				})
				require.Equal(t, ocExemplar.Attachments, pdataExemplarAttachments, msgPrefix+"attachments mismatch")
			}
			// 7. Ensure that bucket bounds are the same.
			require.Equal(t, ocBucketCounts, pdataPoint.BucketCounts(), "Bucket counts must be equal")
			// 8. Ensure that the labels all match up.
			ocStringMap := pdata.NewStringMap()
			for i, labelValue := range ocTimeseries.LabelValues {
				ocStringMap.Insert(mf.labelKeysOrdered[i], labelValue.Value)
			}
			require.Equal(t, ocStringMap.Sort(), pdataPoint.LabelsMap().Sort())
		})
	}
}

func TestMetricGroupData_toSummaryUnitTest(t *testing.T) {
	type scrape struct {
		at     int64
		value  float64
		metric string
	}

	type labelsScrapes struct {
		labels  labels.Labels
		scrapes []*scrape
	}
	tests := []struct {
		name          string
		labelsScrapes []*labelsScrapes
		want          func() pdata.SummaryDataPoint
	}{
		{
			name: "summary",
			labelsScrapes: []*labelsScrapes{
				{
					labels: labels.Labels{
						{Name: "a", Value: "A"}, {Name: "quantile", Value: "0.0"}, {Name: "b", Value: "B"},
					},
					scrapes: []*scrape{
						{at: 10, value: 10, metric: "histogram_count"},
						{at: 10, value: 12, metric: "histogram_sum"},
						{at: 10, value: 8, metric: "value"},
					},
				},
				{
					labels: labels.Labels{
						{Name: "a", Value: "A"}, {Name: "quantile", Value: "0.75"}, {Name: "b", Value: "B"},
					},
					scrapes: []*scrape{
						{at: 11, value: 10, metric: "histogram_count"},
						{at: 11, value: 1004.78, metric: "histogram_sum"},
						{at: 11, value: 33.7, metric: "value"},
					},
				},
				{
					labels: labels.Labels{
						{Name: "a", Value: "A"}, {Name: "quantile", Value: "0.50"}, {Name: "b", Value: "B"},
					},
					scrapes: []*scrape{
						{at: 12, value: 10, metric: "histogram_count"},
						{at: 12, value: 13, metric: "histogram_sum"},
						{at: 12, value: 27, metric: "value"},
					},
				},
				{
					labels: labels.Labels{
						{Name: "a", Value: "A"}, {Name: "quantile", Value: "0.90"}, {Name: "b", Value: "B"},
					},
					scrapes: []*scrape{
						{at: 13, value: 10, metric: "histogram_count"},
						{at: 13, value: 14, metric: "histogram_sum"},
						{at: 13, value: 56, metric: "value"},
					},
				},
				{
					labels: labels.Labels{
						{Name: "a", Value: "A"}, {Name: "quantile", Value: "0.99"}, {Name: "b", Value: "B"},
					},
					scrapes: []*scrape{
						{at: 14, value: 10, metric: "histogram_count"},
						{at: 14, value: 15, metric: "histogram_sum"},
						{at: 14, value: 82, metric: "value"},
					},
				},
			},
			want: func() pdata.SummaryDataPoint {
				point := pdata.NewSummaryDataPoint()
				point.SetCount(10)
				point.SetSum(15)
				qtL := point.QuantileValues()
				qn0 := qtL.AppendEmpty()
				qn0.SetQuantile(0)
				qn0.SetValue(8)
				qn50 := qtL.AppendEmpty()
				qn50.SetQuantile(50)
				qn50.SetValue(27)
				qn75 := qtL.AppendEmpty()
				qn75.SetQuantile(75)
				qn75.SetValue(33.7)
				qn90 := qtL.AppendEmpty()
				qn90.SetQuantile(90)
				qn90.SetValue(56)
				qn99 := qtL.AppendEmpty()
				qn99.SetQuantile(99)
				qn99.SetValue(82)
				point.SetTimestamp(14 * 1e6) // the time in milliseconds -> nanoseconds.
				point.SetStartTimestamp(14 * 1e6)
				labelsMap := point.LabelsMap()
				labelsMap.Insert("a", "A")
				labelsMap.Insert("b", "B")
				return point
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mp := newMetricFamilyPdata(tt.name, mc, 1).(*metricFamilyPdata)
			for _, lbs := range tt.labelsScrapes {
				for _, scrape := range lbs.scrapes {
					require.NoError(t, mp.Add(scrape.metric, lbs.labels.Copy(), scrape.at, scrape.value))
				}
			}

			require.Equal(t, 1, len(mp.groups), "Expecting exactly 1 groupKey")
			// Get the lone group key.
			groupKey := ""
			for key := range mp.groups {
				groupKey = key
			}
			require.NotNil(t, mp.groups[groupKey], "Expecting the groupKey to have a value given key:: "+groupKey)

			sdpL := pdata.NewSummaryDataPointSlice()
			require.True(t, mp.groups[groupKey].toSummaryPoint(mp.labelKeysOrdered, &sdpL))
			require.Equal(t, 1, sdpL.Len(), "Exactly one point expected")
			got := sdpL.At(0)
			want := tt.want()
			require.Equal(t, want, got, "Expected the points to be equal")
		})
	}
}

func TestMetricGroupData_toSummaryPointEquivalence(t *testing.T) {
	type scrape struct {
		at     int64
		value  float64
		metric string
	}
	tests := []struct {
		name    string
		labels  labels.Labels
		scrapes []*scrape
	}{
		{
			name:   "summary",
			labels: labels.Labels{{Name: "a", Value: "A"}, {Name: "quantile", Value: "0.75"}, {Name: "b", Value: "B"}},
			scrapes: []*scrape{
				{at: 11, value: 10, metric: "summary_count"},
				{at: 11, value: 1004.78, metric: "summary_sum"},
				{at: 13, value: 33.7, metric: "value"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mf := newMetricFamily(tt.name, mc, zap.NewNop(), 1).(*metricFamily)
			mp := newMetricFamilyPdata(tt.name, mc, 1).(*metricFamilyPdata)
			for _, tv := range tt.scrapes {
				require.NoError(t, mp.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
				require.NoError(t, mf.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
			}
			groupKey := mf.getGroupKey(tt.labels.Copy())
			ocTimeseries := mf.groups[groupKey].toSummaryTimeSeries(mf.labelKeysOrdered)
			sdpL := pdata.NewSummaryDataPointSlice()
			require.True(t, mp.groups[groupKey].toSummaryPoint(mp.labelKeysOrdered, &sdpL))
			require.Equal(t, len(ocTimeseries.Points), sdpL.Len(), "They should have the exact same number of points")
			require.Equal(t, 1, sdpL.Len(), "Exactly one point expected")
			ocPoint := ocTimeseries.Points[0]
			pdataPoint := sdpL.At(0)
			// 1. Ensure that the startTimestamps are equal.
			require.Equal(t, ocTimeseries.GetStartTimestamp().AsTime(), pdataPoint.Timestamp().AsTime(), "The timestamp must be equal")
			// 2. Ensure that the count is equal.
			ocSummary := ocPoint.GetSummaryValue()
			if false {
				t.Logf("\nOcSummary: %#v\nPdSummary: %#v\n\nocPoint: %#v\n", ocSummary, pdataPoint, ocPoint.GetSummaryValue())
				return
			}
			require.Equal(t, ocSummary.GetCount().GetValue(), int64(pdataPoint.Count()), "Count must be equal")
			// 3. Ensure that the sum is equal.
			require.Equal(t, ocSummary.GetSum().GetValue(), pdataPoint.Sum(), "Sum must be equal")
			// 4. Ensure that the point's timestamp is equal to that from the OpenCensusProto data point.
			require.Equal(t, ocPoint.GetTimestamp().AsTime(), pdataPoint.Timestamp().AsTime(), "Point timestamps must be equal")
			// 5. Ensure that the labels all match up.
			ocStringMap := pdata.NewStringMap()
			for i, labelValue := range ocTimeseries.LabelValues {
				ocStringMap.Insert(mf.labelKeysOrdered[i], labelValue.Value)
			}
			require.Equal(t, ocStringMap.Sort(), pdataPoint.LabelsMap().Sort())
			// 6. Ensure that the quantile values all match up.
			ocQuantiles := ocSummary.GetSnapshot().GetPercentileValues()
			pdataQuantiles := pdataPoint.QuantileValues()
			require.Equal(t, len(ocQuantiles), pdataQuantiles.Len())
			for i, ocQuantile := range ocQuantiles {
				pdataQuantile := pdataQuantiles.At(i)
				require.Equal(t, ocQuantile.Percentile, pdataQuantile.Quantile(), "The quantile percentiles must match")
				require.Equal(t, ocQuantile.Value, pdataQuantile.Value(), "The quantile values must match")
			}
		})
	}
}

func TestMetricGroupData_toNumberDataUnitTest(t *testing.T) {
	type scrape struct {
		at     int64
		value  float64
		metric string
	}
	tests := []struct {
		name                     string
		metricKind               string
		labels                   labels.Labels
		scrapes                  []*scrape
		intervalStartTimestampMs int64
		want                     func() pdata.NumberDataPoint
	}{
		{
			metricKind:               "counter",
			name:                     "counter:: startTimestampMs of 11",
			intervalStartTimestampMs: 11,
			labels:                   labels.Labels{{Name: "a", Value: "A"}, {Name: "b", Value: "B"}},
			scrapes: []*scrape{
				{at: 13, value: 33.7, metric: "value"},
			},
			want: func() pdata.NumberDataPoint {
				point := pdata.NewNumberDataPoint()
				point.SetDoubleVal(33.7)
				point.SetTimestamp(13 * 1e6) // the time in milliseconds -> nanoseconds.
				point.SetStartTimestamp(11 * 1e6)
				labelsMap := point.LabelsMap()
				labelsMap.Insert("a", "A")
				labelsMap.Insert("b", "B")
				return point
			},
		},
		{
			name:                     "counter:: startTimestampMs of 0",
			metricKind:               "counter",
			intervalStartTimestampMs: 0,
			labels:                   labels.Labels{{Name: "a", Value: "A"}, {Name: "b", Value: "B"}},
			scrapes: []*scrape{
				{at: 28, value: 99.9, metric: "value"},
			},
			want: func() pdata.NumberDataPoint {
				point := pdata.NewNumberDataPoint()
				point.SetDoubleVal(99.9)
				point.SetTimestamp(28 * 1e6) // the time in milliseconds -> nanoseconds.
				point.SetStartTimestamp(0)
				labelsMap := point.LabelsMap()
				labelsMap.Insert("a", "A")
				labelsMap.Insert("b", "B")
				return point
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mp := newMetricFamilyPdata(tt.metricKind, mc, tt.intervalStartTimestampMs).(*metricFamilyPdata)
			for _, tv := range tt.scrapes {
				require.NoError(t, mp.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
			}

			require.Equal(t, 1, len(mp.groups), "Expecting exactly 1 groupKey")
			groupKey := mp.getGroupKey(tt.labels.Copy())
			require.NotNil(t, mp.groups[groupKey], "Expecting the groupKey to have a value given key:: "+groupKey)

			ndpL := pdata.NewNumberDataPointSlice()
			require.True(t, mp.groups[groupKey].toNumberDataPoint(mp.labelKeysOrdered, &ndpL))
			require.Equal(t, 1, ndpL.Len(), "Exactly one point expected")
			got := ndpL.At(0)
			want := tt.want()
			require.Equal(t, want, got, "Expected the points to be equal")
		})
	}
}

func TestMetricGroupData_toNumberDataPointEquivalence(t *testing.T) {
	type scrape struct {
		at     int64
		value  float64
		metric string
	}
	tests := []struct {
		name      string
		labels    labels.Labels
		scrapes   []*scrape
		wantValue float64
	}{
		{
			name:   "counter",
			labels: labels.Labels{{Name: "a", Value: "A"}, {Name: "b", Value: "B"}},
			scrapes: []*scrape{
				{at: 13, value: 33.7, metric: "value"},
			},
			wantValue: 33.7,
		},
	}

	for i, tt := range tests {
		tt := tt
		intervalStartTimeMs := int64(11 + i)
		t.Run(tt.name, func(t *testing.T) {
			mf := newMetricFamily(tt.name, mc, zap.NewNop(), intervalStartTimeMs).(*metricFamily)
			mp := newMetricFamilyPdata(tt.name, mc, intervalStartTimeMs).(*metricFamilyPdata)
			for _, tv := range tt.scrapes {
				require.NoError(t, mp.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
				require.NoError(t, mf.Add(tv.metric, tt.labels.Copy(), tv.at, tv.value))
			}
			groupKey := mf.getGroupKey(tt.labels.Copy())
			ocTimeseries := mf.groups[groupKey].toDoubleValueTimeSeries(mf.labelKeysOrdered)
			ddpL := pdata.NewNumberDataPointSlice()
			require.True(t, mp.groups[groupKey].toNumberDataPoint(mp.labelKeysOrdered, &ddpL))
			require.Equal(t, len(ocTimeseries.Points), ddpL.Len(), "They should have the exact same number of points")
			require.Equal(t, 1, ddpL.Len(), "Exactly one point expected")
			ocPoint := ocTimeseries.Points[0]
			pdataPoint := ddpL.At(0)
			// 1. Ensure that the startTimestamps are equal.
			require.Equal(t, ocTimeseries.GetStartTimestamp().AsTime(), pdataPoint.StartTimestamp().AsTime(), "The timestamp must be equal")
			require.Equal(t, intervalStartTimeMs*1e6, pdataPoint.StartTimestamp().AsTime().UnixNano(), "intervalStartTimeMs must be the same")
			// 2. Ensure that the value is equal.
			require.Equal(t, ocPoint.GetDoubleValue(), pdataPoint.DoubleVal(), "Values must be equal")
			require.Equal(t, tt.wantValue, pdataPoint.DoubleVal(), "Values must be equal")
			// 4. Ensure that the point's timestamp is equal to that from the OpenCensusProto data point.
			require.Equal(t, ocPoint.GetTimestamp().AsTime(), pdataPoint.Timestamp().AsTime(), "Point timestamps must be equal")
			// 5. Ensure that the labels all match up.
			ocStringMap := pdata.NewStringMap()
			for i, labelValue := range ocTimeseries.LabelValues {
				ocStringMap.Insert(mf.labelKeysOrdered[i], labelValue.Value)
			}
			require.Equal(t, ocStringMap.Sort(), pdataPoint.LabelsMap().Sort())
		})
	}
}
