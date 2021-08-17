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
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/model/pdata"
)

type dataPoint struct {
	value    float64
	boundary float64
}

// MetricFamilyPdata is unit which is corresponding to the metrics items which shared the same TYPE/UNIT/... metadata from
// a single scrape.
type MetricFamilyPdata interface {
	Add(metricName string, ls labels.Labels, t int64, v float64) error
	IsSameFamily(metricName string) bool
	ToMetricPdata(metrics *pdata.MetricSlice) (int, int)
}

type metricFamilyPdata struct {
	mtype               pdata.MetricDataType
	groups              map[string]*metricGroupPdata
	name                string
	mc                  MetadataCache
	droppedTimeseries   int
	labelKeys           map[string]bool
	labelKeysOrdered    []string
	metadata            *scrape.MetricMetadata
	groupOrders         map[string]int
	intervalStartTimeMs int64
}

// metricGroupPdata, represents a single metric of a metric family. for example a histogram metric is usually represent by
// a couple data complexValue (buckets and count/sum), a group of a metric family always share a same set of tags. for
// simple types like counter and gauge, each data point is a group of itself
type metricGroupPdata struct {
	family              *metricFamilyPdata
	ts                  int64
	ls                  labels.Labels
	count               float64
	hasCount            bool
	sum                 float64
	hasSum              bool
	value               float64
	complexValue        []*dataPoint
	intervalStartTimeMs int64
}

func newMetricFamilyPdata(metricName string, mc MetadataCache, logger *zap.Logger, intervalStartTimeMs int64) MetricFamilyPdata {
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
	} else if !ok && isInternalMetric(metricName) {
		metadata = defineInternalMetric(metricName, metadata, logger)
	}

	mtype := convToPdataMetricType(metadata.Type)
	if mtype == pdata.MetricDataTypeNone {
		logger.Debug(fmt.Sprintf("Invalid metric : %s %+v", metricName, metadata))
	}

	return &metricFamilyPdata{
		mtype:               mtype,
		groups:              make(map[string]*metricGroupPdata),
		name:                familyName,
		mc:                  mc,
		droppedTimeseries:   0,
		labelKeys:           make(map[string]bool),
		labelKeysOrdered:    make([]string, 0),
		metadata:            &metadata,
		groupOrders:         make(map[string]int),
		intervalStartTimeMs: intervalStartTimeMs,
	}
}

func (mf *metricFamilyPdata) IsSameFamily(metricName string) bool {
	// trim known suffix if necessary
	familyName := normalizeMetricName(metricName)
	return mf.name == familyName || familyName != metricName && mf.name == metricName
}

// updateLabelKeys is used to store all the label keys of a same metric family in observed order. since prometheus
// receiver removes any label with empty value before feeding it to an appender, in order to figure out all the labels
// from the same metric family we will need to keep track of what labels have ever been observed.
func (mf *metricFamilyPdata) updateLabelKeys(ls labels.Labels) {
	for _, l := range ls {
		if isUsefulLabelPdata(mf.mtype, l.Name) {
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
}

func (mf *metricFamilyPdata) getGroupKey(ls labels.Labels) string {
	mf.updateLabelKeys(ls)
	return dpgSignature(mf.labelKeysOrdered, ls)
}

func (mg *metricGroupPdata) sortPoints() {
	sort.Slice(mg.complexValue, func(i, j int) bool {
		return mg.complexValue[i].boundary < mg.complexValue[j].boundary
	})
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
	// The timestamp MUST be in retrieved from milliseconds and converted to nanoseconds.
	tsNanos := timestampFromMs(mg.ts)
	// TODO (@odeke-em): investigate why the tests somehow expect StartTimestamp to be nil.
	// point.SetStartTimestamp(tsNanos)
	point.SetTimestamp(tsNanos)
	populateLabelValuesPdata(orderedLabelKeys, mg.ls, point.LabelsMap())

	return true
}

func timestampFromMs(timeAtMs int64) pdata.Timestamp {
	secs, ns := timeAtMs/1e3, (timeAtMs%1e3)*1e6
	return pdata.TimestampFromTime(time.Unix(secs, ns))
}

func (mg *metricGroupPdata) toSummaryPoint(orderedLabelKeys []string, dest *pdata.SummaryDataPointSlice) bool {
	// expecting count to be provided, however, in the following two cases, they can be missed.
	// 1. data is corrupted
	// 2. ignored by startValue evaluation
	if !mg.hasCount {
		return false
	}

	mg.sortPoints()

	point := dest.AppendEmpty()
	quantileValues := point.QuantileValues()
	for _, p := range mg.complexValue {
		quantile := quantileValues.AppendEmpty()
		quantile.SetValue(p.value)
		quantile.SetQuantile(p.boundary * 100)
	}

	// Based on the summary description from https://prometheus.io/docs/concepts/metric_types/#summary
	// the quantiles are calculated over a sliding time window, however, the count is the total count of
	// observations and the corresponding sum is a sum of all observed values, thus the sum and count used
	// at the global level of the metricspb.SummaryValue
	// The timestamp MUST be in retrieved from milliseconds and converted to nanoseconds.
	tsNanos := timestampFromMs(mg.ts)
	point.SetTimestamp(tsNanos)
	// TODO (@odeke-em): investigate why the tests somehow expect StartTimestamp to be nil.
	// point.SetStartTimestamp(tsNanos)
	if mg.family.isCumulativeTypePdata() {
		point.SetStartTimestamp(timestampFromMs(mg.intervalStartTimeMs))
	}
	point.SetSum(mg.sum)
	point.SetCount(uint64(mg.count))
	populateLabelValuesPdata(orderedLabelKeys, mg.ls, point.LabelsMap())

	return true
}

func (mg *metricGroupPdata) toNumberDataPoint(orderedLabelKeys []string, dest *pdata.NumberDataPointSlice) bool {
	var startTsNanos pdata.Timestamp
	tsNanos := timestampFromMs(mg.ts)
	// gauge/undefined types have no start time.
	if mg.family.isCumulativeTypePdata() {
		startTsNanos = timestampFromMs(mg.intervalStartTimeMs)
	}

	point := dest.AppendEmpty()
	point.SetStartTimestamp(startTsNanos)
	point.SetTimestamp(tsNanos)
	point.SetDoubleVal(mg.value)
	populateLabelValuesPdata(orderedLabelKeys, mg.ls, point.LabelsMap())

	return true
}

func populateLabelValuesPdata(orderedKeys []string, ls labels.Labels, dest pdata.StringMap) {
	src := ls.Map()
	for _, key := range orderedKeys {
		dest.Insert(key, src[key])
	}
}

// Purposefully being referenced to avoid lint warnings about being "unused".
var _ = (*metricFamilyPdata)(nil).updateLabelKeys

func (mf *metricFamilyPdata) isCumulativeTypePdata() bool {
	return mf.mtype == pdata.MetricDataTypeSum ||
		mf.mtype == pdata.MetricDataTypeHistogram ||
		mf.mtype == pdata.MetricDataTypeSummary
}

func (mf *metricFamilyPdata) loadMetricGroupOrCreate(groupKey string, ls labels.Labels, ts int64) *metricGroupPdata {
	mg, ok := mf.groups[groupKey]
	if !ok {
		mg = &metricGroupPdata{
			family:              mf,
			ts:                  ts,
			ls:                  ls,
			complexValue:        make([]*dataPoint, 0),
			intervalStartTimeMs: mf.intervalStartTimeMs,
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

func (mf *metricFamilyPdata) ToMetricPdata(metrics *pdata.MetricSlice) (int, int) {
	metric := pdata.NewMetric()
	metric.SetDataType(mf.mtype)
	metric.SetName(mf.name)

	pointCount := 0

	switch mf.mtype {
	case pdata.MetricDataTypeHistogram:
		histogram := metric.Histogram()
		hdpL := histogram.DataPoints()
		for _, mg := range mf.getGroups() {
			if !mg.toDistributionPoint(mf.labelKeysOrdered, &hdpL) {
				mf.droppedTimeseries++
			}
		}
		pointCount = hdpL.Len()

	case pdata.MetricDataTypeSummary:
		summary := metric.Summary()
		sdpL := summary.DataPoints()
		for _, mg := range mf.getGroups() {
			if !mg.toSummaryPoint(mf.labelKeysOrdered, &sdpL) {
				mf.droppedTimeseries++
			}
		}
		pointCount = sdpL.Len()

	case pdata.MetricDataTypeSum:
		sum := metric.Sum()
		sdpL := sum.DataPoints()
		for _, mg := range mf.getGroups() {
			if !mg.toNumberDataPoint(mf.labelKeysOrdered, &sdpL) {
				mf.droppedTimeseries++
			}
		}
		pointCount = sdpL.Len()

	default: // Everything else should be set to a Gauge.
		metric.SetDataType(pdata.MetricDataTypeGauge)
		gauge := metric.Gauge()
		gdpL := gauge.DataPoints()
		for _, mg := range mf.getGroups() {
			if !mg.toNumberDataPoint(mf.labelKeysOrdered, &gdpL) {
				mf.droppedTimeseries++
			}
		}
		pointCount = gdpL.Len()
	}

	if pointCount == 0 {
		return mf.droppedTimeseries, mf.droppedTimeseries
	}

	metric.CopyTo(metrics.AppendEmpty())

	// note: the total number of points is the number of points+droppedTimeseries.
	return pointCount + mf.droppedTimeseries, mf.droppedTimeseries
}

// Define manually the metadata of prometheus scrapper internal metrics
func defineInternalMetric(metricName string, metadata scrape.MetricMetadata, logger *zap.Logger) scrape.MetricMetadata {
	if metadata.Metric != "" && metadata.Type != "" && metadata.Help != "" {
		logger.Debug("Internal metric seems already fully defined")
		return metadata
	}
	metadata.Metric = metricName

	switch metricName {
	case scrapeUpMetricName:
		metadata.Type = textparse.MetricTypeGauge
		metadata.Help = "The scraping was successful"
	case "scrape_duration_seconds":
		metadata.Unit = "seconds"
		metadata.Type = textparse.MetricTypeGauge
		metadata.Help = "Duration of the scrape"
	case "scrape_samples_scraped":
		metadata.Type = textparse.MetricTypeGauge
		metadata.Help = "The number of samples the target exposed"
	case "scrape_series_added":
		metadata.Type = textparse.MetricTypeGauge
		metadata.Help = "The approximate number of new series in this scrape"
	case "scrape_samples_post_metric_relabeling":
		metadata.Type = textparse.MetricTypeGauge
		metadata.Help = "The number of samples remaining after metric relabeling was applied"
	}
	return metadata
}
