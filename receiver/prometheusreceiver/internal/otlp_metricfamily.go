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
	"sort"
	"strings"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"

	"go.opentelemetry.io/collector/model/pdata"
)

// MetricFamilyPdata is unit which is corresponding to the metrics items which shared the same TYPE/UNIT/... metadata from
// a single scrape.
type MetricFamilyPdata interface {
	Add(metricName string, ls labels.Labels, t int64, v float64) error
	IsSameFamily(metricName string) bool
	ToMetricPdata() (*pdata.Metric, int, int)
}

var _ MetricFamilyPdata = (*metricFamilyPdata)(nil)

type metricFamilyPdata struct {
	// We are composing the already present metricFamily to
	// make for a scalable migration, so that we only edit target
	// fields progressively, when we are ready to make changes.
	metricFamily
	mtype  pdata.MetricDataType
	groups map[string]*metricGroupPdata
}

// metricGroupPdata, represents a single metric of a metric family. for example a histogram metric is usually represent by
// a couple data complexValue (buckets and count/sum), a group of a metric family always share a same set of tags. for
// simple types like counter and gauge, each data point is a group of itself
type metricGroupPdata struct {
	// We are composing the already present metricGroup to
	// make for a scalable migration, so that we only edit target
	// fields progressively, when we are ready to make changes.
	metricGroup
	family *metricFamilyPdata
}

func newMetricFamilyPdata(metricName string, mc MetadataCache) MetricFamily {
	familyName := normalizeMetricName(metricName)

	// lookup metadata based on familyName
	metadata, ok := mc.Metadata(familyName)
	if !ok && metricName != familyName {
		// use the original metricName as metricFamily
		familyName = metricName
		// perform a 2nd lookup with the original metric name. it can happen if there's a metric which is not histogram
		// or summary, but ends with one of those _count/_sum suffixes
		metadata, ok = mc.Metadata(metricName)
		// still not found, this can happen when metric has no TYPE HINT
		if !ok {
			metadata.Metric = familyName
			metadata.Type = textparse.MetricTypeUnknown
		}
	}

	return &metricFamilyPdata{
		mtype:  convToPdataMetricType(metadata.Type),
		groups: make(map[string]*metricGroupPdata),
		metricFamily: metricFamily{
			name:              familyName,
			mc:                mc,
			droppedTimeseries: 0,
			labelKeys:         make(map[string]bool),
			labelKeysOrdered:  make([]string, 0),
			metadata:          &metadata,
			groupOrders:       make(map[string]int),
		},
	}
}

func (mg *metricGroupPdata) toDistributionPoint(orderedLabelKeys []string, dest *pdata.HistogramDataPointSlice) bool {
	if !mg.hasCount || len(mg.complexValue) == 0 {
		return false
	}

	mg.sortPoints()

	// for OCAgent Proto, the bounds won't include +inf
	// TODO: (@odeke-em) should we also check OpenTelemetry Pdata for bucket bounds?
	bounds := make([]float64, len(mg.complexValue)-1)
	bucketCounts := make([]uint64, len(mg.complexValue))

	for i := 0; i < len(mg.complexValue); i++ {
		if i != len(mg.complexValue)-1 {
			// not need to add +inf as bound to oc proto
			bounds[i] = mg.complexValue[i].boundary
		}
		adjustedCount := mg.complexValue[i].value
		if i != 0 {
			adjustedCount -= mg.complexValue[i-1].value
		}
		bucketCounts[i] = uint64(adjustedCount)
	}

	point := dest.AppendEmpty()
	point.SetExplicitBounds(bounds)
	point.SetCount(uint64(mg.count))
	point.SetSum(mg.sum)
	point.SetBucketCounts(bucketCounts)
	point.SetStartTimestamp(pdata.Timestamp(mg.ts))
	point.SetTimestamp(pdata.Timestamp(mg.ts))

	return true
}

func (mg *metricGroupPdata) toSummaryPoint(orderedLabelKeys []string, dest *pdata.SummaryDataPointSlice) bool {
	// expecting count to be provided, however, in the following two cases, they can be missed.
	// 1. data is corrupted
	// 2. ignored by startValue evaluation
	if !mg.hasCount {
		return false
	}

	mg.sortPoints()

	quantileValues := pdata.NewValueAtQuantileSlice()
	for _, p := range mg.complexValue {
		quantile := pdata.NewValueAtQuantile()
		quantile.SetValue(p.value)
		quantile.SetQuantile(p.boundary * 100)
	}

	point := dest.AppendEmpty()
	quantileValues.CopyTo(point.QuantileValues())

	strMap := populateLabelValuesPdata(orderedLabelKeys, mg.ls)
	strMap.CopyTo(point.LabelsMap())

	// Based on the summary description from https://prometheus.io/docs/concepts/metric_types/#summary
	// the quantiles are calculated over a sliding time window, however, the count is the total count of
	// observations and the corresponding sum is a sum of all observed values, thus the sum and count used
	// at the global level of the metricspb.SummaryValue
	point.SetStartTimestamp(pdata.Timestamp(mg.ts))
	point.SetTimestamp(pdata.Timestamp(mg.ts))
	point.SetSum(mg.sum)
	point.SetCount(uint64(mg.count))

	return true
}

func populateLabelValuesPdata(orderedKeys []string, ls labels.Labels) pdata.StringMap {
	strMap := pdata.NewStringMap()
	lmap := ls.Map()
	for _, key := range orderedKeys {
		strMap.Insert(key, lmap[key])
	}
	return strMap
}

// updateLabelKeys is used to store all the label keys of a same metric family in observed order. since prometheus
// receiver removes any label with empty value before feeding it to an appender, in order to figure out all the labels
// from the same metric family we will need to keep track of what labels have ever been observed.
func (mf *metricFamilyPdata) updateLabelKeys(ls labels.Labels) {
	for _, l := range ls {
		if !isUsefulLabelPdata(mf.mtype, l.Name) {
			continue
		}
		if _, ok := mf.labelKeys[l.Name]; !ok {
			mf.labelKeys[l.Name] = true
			// use insertion sort to maintain order
			i := sort.SearchStrings(mf.labelKeysOrdered, l.Name)
			mf.labelKeysOrdered = append(mf.labelKeysOrdered, "")
			copy(mf.labelKeysOrdered[i+1:], mf.labelKeysOrdered[i:])
			mf.labelKeysOrdered[i] = l.Name
		}
	}
}

// Purposefully being referenced to avoid lint warnings about being "unused".
var _ = (*metricFamilyPdata)(nil).updateLabelKeys

func (mf *metricFamilyPdata) isCumulativeTypePdata() bool {
	return mf.mtype == pdata.MetricDataTypeSum ||
		mf.mtype == pdata.MetricDataTypeIntSum ||
		mf.mtype == pdata.MetricDataTypeHistogram ||
		mf.mtype == pdata.MetricDataTypeSummary
}

func (mf *metricFamilyPdata) loadMetricGroupOrCreate(groupKey string, ls labels.Labels, ts int64) *metricGroupPdata {
	mg, ok := mf.groups[groupKey]
	if !ok {
		mg = &metricGroupPdata{
			family: mf,
			metricGroup: metricGroup{
				ls:           ls,
				ts:           ts,
				complexValue: make([]*dataPoint, 0),
			},
		}
		mf.groups[groupKey] = mg
		// maintaining data insertion order is helpful to generate stable/reproducible metric output
		mf.groupOrders[groupKey] = len(mf.groupOrders)
	}
	return mg
}

func (mf *metricFamilyPdata) Add(metricName string, ls labels.Labels, t int64, v float64) error {
	groupKey := mf.getGroupKey(ls)
	mg := mf.loadMetricGroupOrCreate(groupKey, ls, t)
	switch mf.mtype {
	case pdata.MetricDataTypeHistogram, pdata.MetricDataTypeSummary:
		switch {
		case strings.HasSuffix(metricName, metricsSuffixSum):
			// always use the timestamp from sum (count is ok too), because the startTs from quantiles won't be reliable
			// in cases like remote server restart
			mg.ts = t
			mg.sum = v
			mg.hasSum = true
		case strings.HasSuffix(metricName, metricsSuffixCount):
			mg.count = v
			mg.hasCount = true
		default:
			boundary, err := getBoundaryPdata(mf.mtype, ls)
			if err != nil {
				mf.droppedTimeseries++
				return err
			}
			mg.complexValue = append(mg.complexValue, &dataPoint{value: v, boundary: boundary})
		}
	default:
		mg.value = v
	}

	return nil
}

// getGroups to return groups in insertion order
func (mf *metricFamilyPdata) getGroups() []*metricGroupPdata {
	groups := make([]*metricGroupPdata, len(mf.groupOrders))
	for k, v := range mf.groupOrders {
		groups[v] = mf.groups[k]
	}
	return groups
}

func (mf *metricFamilyPdata) ToMetricPdata() (metric *pdata.Metric, nTotalTimeseries, droppedTimeseries int) {
	numTimeseries := 0
	switch mf.mtype {
	// not supported currently
	// case metricspb.MetricDescriptor_GAUGE_DISTRIBUTION:
	//	return nil
	case pdata.MetricDataTypeHistogram:
		histogramDataPoints := pdata.NewHistogramDataPointSlice()
		for _, mg := range mf.getGroups() {
			// The API for adding points to a HistogramDataPointSlice is so awkward
			// and doesn't have a simple way of adding a point and requires
			if added := mg.toDistributionPoint(mf.labelKeysOrdered, &histogramDataPoints); !added {
				mf.droppedTimeseries++
			}
		}

		numTimeseries = histogramDataPoints.Len()
		if numTimeseries != 0 {
			pm := pdata.NewMetric()
			metric = &pm
			metric.SetDataType(mf.mtype)
			histogram := metric.Histogram()
			histogramDataPoints.CopyTo(histogram.DataPoints())
		}

	case pdata.MetricDataTypeSummary:
		summaryDataPoints := pdata.NewSummaryDataPointSlice()
		for _, mg := range mf.getGroups() {
			if added := mg.toSummaryPoint(mf.labelKeysOrdered, &summaryDataPoints); !added {
				mf.droppedTimeseries++
			}
		}

		numTimeseries = summaryDataPoints.Len()
		if numTimeseries != 0 {
			pm := pdata.NewMetric()
			metric = &pm
			metric.SetDataType(mf.mtype)
			summary := metric.Summary()
			summaryDataPoints.CopyTo(summary.DataPoints())
		}

	default:
		doubleDataPoints := pdata.NewDoubleDataPointSlice()
		for _, mg := range mf.getGroups() {
			if added := mg.toDoublePoint(mf.labelKeysOrdered, &doubleDataPoints); !added {
				mf.droppedTimeseries++
			}
		}

		numTimeseries = doubleDataPoints.Len()
		if numTimeseries != 0 {
			pm := pdata.NewMetric()
			metric = &pm
			metric.SetDataType(mf.mtype)
			panic(fmt.Sprintf("We need a specialized converter for %s", mf.mtype))
		}
	}

	// note: the total number of timeseries is the length of timeseries plus the number of dropped timeseries.
	if numTimeseries == 0 {
		return nil, mf.droppedTimeseries, mf.droppedTimeseries
	}

	metric.SetName(mf.name)
	metric.SetDescription(mf.metadata.Help)
	metric.SetUnit(heuristicalMetricAndKnownUnits(mf.name, mf.metadata.Unit))
	metric.SetDataType(mf.mtype)
	return metric, numTimeseries + mf.droppedTimeseries, mf.droppedTimeseries
}

func (mg *metricGroupPdata) toDoublePoint(orderedLabelKeys []string, dest *pdata.DoubleDataPointSlice) bool {
	var startTs int64
	// gauge/undefined types has no start time
	if mg.family.isCumulativeTypePdata() {
		startTs = mg.ts
	}

	point := pdata.NewDoubleDataPoint()
	point.SetStartTimestamp(pdata.Timestamp(startTs))
	point.SetTimestamp(pdata.Timestamp(mg.ts))
	point.SetValue(mg.value)

	return true
}
