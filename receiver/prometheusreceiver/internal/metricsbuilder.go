// Copyright 2019, OpenTelemetry Authors
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

package internal

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
)

const metricsSuffixCount = "_count"
const metricsSuffixBucket = "_bucket"
const metricsSuffixSum = "_sum"

var trimmableSuffixes = []string{metricsSuffixBucket, metricsSuffixCount, metricsSuffixSum}
var errNoDataToBuild = errors.New("there's no data to build")
var errNoBoundaryLabel = errors.New("given metricType has no BucketLabel or QuantileLabel")
var errEmptyBoundaryLabel = errors.New("BucketLabel or QuantileLabel is empty")
var dummyMetric = &consumerdata.MetricsData{
	Node: &commonpb.Node{
		Identifier:  &commonpb.ProcessIdentifier{HostName: "127.0.0.1"},
		ServiceInfo: &commonpb.ServiceInfo{Name: "internal"},
	},
	Metrics: make([]*metricspb.Metric, 0),
}

type metricBuilder struct {
	node                   *commonpb.Node
	ts                     int64
	hasData                bool
	hasInternalMetric      bool
	mc                     MetadataCache
	metrics                []*metricspb.Metric
	currentMetricFamily    string
	currentMetricLabelKeys map[string]int
	currentMetadata        *scrape.MetricMetadata
	currentMetric          *metricspb.Metric
	currentDpGroups        map[string][]*dataPoint
	currentDpGOrder        map[string]int
	currentMetadataGroups  map[string]*dataPointGroupMetadata
	logger                 *zap.SugaredLogger
}

type dataPoint struct {
	value    float64
	boundary float64
	ts       int64
}

type dataPointGroupMetadata struct {
	ts       int64
	ls       *labels.Labels
	count    float64
	hasCount bool
	sum      float64
	hasSum   bool
}

// newMetricBuilder creates a MetricBuilder which is allowed to feed all the datapoints from a single prometheus
// scraped page by calling its AddDataPoint function, and turn them into an opencensus consumerdata.MetricsData object
// by calling its Build function
func newMetricBuilder(node *commonpb.Node, mc MetadataCache, logger *zap.SugaredLogger) *metricBuilder {

	return &metricBuilder{
		node:    node,
		mc:      mc,
		metrics: make([]*metricspb.Metric, 0),
		logger:  logger,
	}
}

// AddDataPoint is for feeding prometheus data points in its processing order
func (b *metricBuilder) AddDataPoint(ls labels.Labels, t int64, v float64) error {
	metricName := ls.Get(model.MetricNameLabel)
	if metricName == "" {
		return errMetricNameNotFound
	} else if shouldSkip(metricName) {
		b.hasInternalMetric = true
		lm := ls.Map()
		delete(lm, model.MetricNameLabel)
		b.logger.Infow("skip internal metric", "name", metricName, "ts", t, "value", v, "labels", lm)
		return nil
	}

	b.hasData = true
	if err := b.initNewMetricIfNeeded(metricName, ls); err != nil {
		return err
	}

	// update the labelKeys array, need to do it for every metrics as scrapeLoop can remove labels with empty values
	// only when we complete the whole MetricFamily, we can get the full set of label names
	b.updateLabelKeys(ls)
	groupKey := dpgSignature(b.currentMetricLabelKeys, ls)
	mg, ok := b.currentMetadataGroups[groupKey]
	if !ok {
		mg = &dataPointGroupMetadata{ts: t, ls: &ls}
		b.currentMetadataGroups[groupKey] = mg
	}
	switch mtype := b.currentMetric.MetricDescriptor.Type; mtype {
	case metricspb.MetricDescriptor_GAUGE_DISTRIBUTION:
		fallthrough
	case metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		fallthrough
	case metricspb.MetricDescriptor_SUMMARY:
		if strings.HasSuffix(metricName, metricsSuffixSum) {
			mg.sum = v
			mg.hasSum = true
		} else if strings.HasSuffix(metricName, metricsSuffixCount) {
			mg.count = v
			mg.hasCount = true
		} else {
			pg, ok := b.currentDpGroups[groupKey]
			if !ok {
				pg = make([]*dataPoint, 0)
				b.currentDpGOrder[groupKey] = len(b.currentDpGOrder)
			}
			boundary, err := getBoundary(mtype, ls)
			if err != nil {
				return err
			}
			dp := &dataPoint{
				value:    v,
				boundary: boundary,
			}
			b.currentDpGroups[groupKey] = append(pg, dp)
		}
	default:
		pg, ok := b.currentDpGroups[groupKey]
		if !ok {
			pg = make([]*dataPoint, 0, 4)
			b.currentDpGOrder[groupKey] = len(b.currentDpGOrder)
		}
		dp := &dataPoint{
			value: v,
			ts:    t,
		}
		b.currentDpGroups[groupKey] = append(pg, dp)
	}

	return nil
}

// Build is to build an opencensus consumerdata.MetricsData based on all added data points
func (b *metricBuilder) Build() (*consumerdata.MetricsData, error) {
	if !b.hasData {
		if b.hasInternalMetric {
			return dummyMetric, nil
		}
		return nil, errNoDataToBuild
	}

	if err := b.completeCurrentMetric(); err != nil {
		return nil, err
	}
	return &consumerdata.MetricsData{
		Node:    b.node,
		Metrics: b.metrics,
	}, nil
}

func (b *metricBuilder) initNewMetricIfNeeded(metricName string, _ labels.Labels) error {
	metricFamily := normalizeMetricName(metricName)
	// the 2nd case is for poorly named metrics
	if b.currentMetricFamily != metricFamily && !(metricFamily != metricName && metricName == b.currentMetricFamily) {
		if b.currentMetric != nil {
			if err := b.completeCurrentMetric(); err != nil {
				return err
			}
		}

		// setup new metric group
		metadata, ok := b.mc.Metadata(metricFamily)

		// perform a 2nd lookup with the original metric name
		// this can only happen if there's a metric is not histogram or summary and ends with one of the suffixes
		// not a good name, but you can still do it that way. hence perform a 2nd lookup with original name
		if !ok && metricName != metricFamily {
			metricFamily = metricName
			metadata, ok = b.mc.Metadata(metricFamily)
			// still not found, this can happen for metrics has no TYPE HINTS
			if !ok {
				metadata.Metric = metricFamily
				metadata.Type = textparse.MetricTypeUnknown
			}
		}
		b.currentMetricFamily = metricFamily
		b.currentMetadata = &metadata
		b.currentDpGroups = make(map[string][]*dataPoint)
		b.currentDpGOrder = make(map[string]int)
		b.currentMetadataGroups = make(map[string]*dataPointGroupMetadata)
		b.currentMetric = &metricspb.Metric{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        metricFamily,
				Description: metadata.Help,
				Unit:        heuristicalMetricAndKnownUnits(metricFamily, metadata.Unit),
				Type:        convToOCAMetricType(metadata.Type),
				// Due to the fact that scrapeLoop strips any tags with empty value, we cannot get the
				// full set of labels by just looking at the first metrics, thus setting the LabelKeys of
				// this proto need to be done after finished the whole group
			},
			Timeseries: make([]*metricspb.TimeSeries, 0),
		}
		b.currentMetricLabelKeys = make(map[string]int)
	}
	return nil
}

func (b *metricBuilder) updateLabelKeys(ls labels.Labels) {
	for _, l := range ls {
		if isUsefulLabel(b.currentMetric.MetricDescriptor.Type, l.Name) {
			if _, ok := b.currentMetricLabelKeys[l.Name]; !ok {
				b.currentMetricLabelKeys[l.Name] = len(b.currentMetricLabelKeys)
			}
		}
	}
}

func (b *metricBuilder) getLabelKeys() []*metricspb.LabelKey {
	lks := make([]*metricspb.LabelKey, len(b.currentMetricLabelKeys))

	for k, v := range b.currentMetricLabelKeys {
		lks[v] = &metricspb.LabelKey{Key: k}
	}

	return lks
}

func (b *metricBuilder) getLabelValues(ls *labels.Labels) []*metricspb.LabelValue {
	lvs := make([]*metricspb.LabelValue, len(b.currentMetricLabelKeys))
	for k, v := range b.currentMetricLabelKeys {
		value := ls.Get(k)
		lvs[v] = &metricspb.LabelValue{Value: value, HasValue: value != ""}
	}

	return lvs
}

func (b *metricBuilder) completeCurrentMetric() error {
	switch mtype := b.currentMetric.MetricDescriptor.Type; mtype {
	case metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		groups := b.currentDpGroupOrdered()
		if len(groups) == 0 {
			// this can happen if only sum or count is added, but not following data points
			return errors.New("no data point added to summary")
		}
		for _, gk := range groups {
			pts := b.currentDpGroups[gk]
			mg, ok := b.currentMetadataGroups[gk]
			if !ok || !(mg.hasSum && mg.hasCount) {
				return errors.New("count or sum is missing: " + gk)
			}

			// always use the first timestamp for the whole metric group
			ts := timestampFromMs(mg.ts)
			// sort it by boundary
			sort.Slice(pts, func(i, j int) bool {
				return pts[i].boundary < pts[j].boundary
			})

			// for OCAgent Proto, the bounds won't include +inf
			bounds := make([]float64, len(pts)-1)
			buckets := make([]*metricspb.DistributionValue_Bucket, len(pts))

			for i := 0; i < len(pts); i++ {
				if i != len(pts)-1 {
					// not need to add +inf as bound to oc proto
					bounds[i] = pts[i].boundary
				}
				adjustedCount := pts[i].value
				if i != 0 {
					adjustedCount -= pts[i-1].value
				}
				buckets[i] = &metricspb.DistributionValue_Bucket{Count: int64(adjustedCount)}
			}

			distrValue := &metricspb.DistributionValue{
				BucketOptions: &metricspb.DistributionValue_BucketOptions{
					Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
						Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
							Bounds: bounds,
						},
					},
				},
				Count:   int64(mg.count),
				Sum:     mg.sum,
				Buckets: buckets,
				// SumOfSquaredDeviation:  // there's no way to compute this value from prometheus data
			}

			timeseries := &metricspb.TimeSeries{
				StartTimestamp: ts,
				LabelValues:    b.getLabelValues(mg.ls),
				Points: []*metricspb.Point{
					{Timestamp: ts, Value: &metricspb.Point_DistributionValue{DistributionValue: distrValue}},
				},
			}

			b.currentMetric.MetricDescriptor.LabelKeys = b.getLabelKeys()
			b.currentMetric.Timeseries = append(b.currentMetric.Timeseries, timeseries)
		}
	case metricspb.MetricDescriptor_SUMMARY:
		groups := b.currentDpGroupOrdered()
		if len(groups) == 0 {
			// this can happen if only sum or count is added, but not following data points
			return errors.New("no data point added to summary")
		}
		for _, gk := range groups {
			pts := b.currentDpGroups[gk]
			mg, ok := b.currentMetadataGroups[gk]
			if !ok || !(mg.hasSum && mg.hasCount) {
				return errors.New("count or sum is missing: " + gk)
			}

			// always use the first timestamp for the whole metric group
			ts := timestampFromMs(mg.ts)
			// sort it by boundary
			sort.Slice(pts, func(i, j int) bool {
				return pts[i].boundary < pts[j].boundary
			})

			percentiles := make([]*metricspb.SummaryValue_Snapshot_ValueAtPercentile, len(pts))

			for i, p := range pts {
				percentiles[i] =
					&metricspb.SummaryValue_Snapshot_ValueAtPercentile{Percentile: p.boundary * 100, Value: p.value}
			}

			// Based on the summary description from https://prometheus.io/docs/concepts/metric_types/#summary
			// the quantiles are calculated over a sliding time window, however, the count is the total count of
			// observations and the corresponding sum is a sum of all observed values, thus the sum and count used
			// at the global level of the metricspb.SummaryValue

			summaryValue := &metricspb.SummaryValue{
				Sum:   &wrappers.DoubleValue{Value: mg.sum},
				Count: &wrappers.Int64Value{Value: int64(mg.count)},
				Snapshot: &metricspb.SummaryValue_Snapshot{
					PercentileValues: percentiles,
				},
			}

			timeseries := &metricspb.TimeSeries{
				StartTimestamp: ts,
				LabelValues:    b.getLabelValues(mg.ls),
				Points: []*metricspb.Point{
					{Timestamp: ts, Value: &metricspb.Point_SummaryValue{SummaryValue: summaryValue}},
				},
			}

			b.currentMetric.MetricDescriptor.LabelKeys = b.getLabelKeys()
			b.currentMetric.Timeseries = append(b.currentMetric.Timeseries, timeseries)
		}
	default:
		for _, gk := range b.currentDpGroupOrdered() {
			pts := b.currentDpGroups[gk]
			mg := b.currentMetadataGroups[gk]

			var startTs *timestamp.Timestamp
			// do not set startTs if metric type is Gauge as per comment
			if mtype != metricspb.MetricDescriptor_GAUGE_DOUBLE {
				startTs = timestampFromMs(mg.ts)
			}

			for _, p := range pts {
				timeseries := &metricspb.TimeSeries{
					StartTimestamp: startTs,
					Points:         []*metricspb.Point{{Timestamp: timestampFromMs(p.ts), Value: &metricspb.Point_DoubleValue{DoubleValue: p.value}}},
					LabelValues:    b.getLabelValues(mg.ls),
				}
				b.currentMetric.Timeseries = append(b.currentMetric.Timeseries, timeseries)
			}

			b.currentMetric.MetricDescriptor.LabelKeys = b.getLabelKeys()
		}
	}

	b.metrics = append(b.metrics, b.currentMetric)
	b.currentMetric = nil
	return nil
}

func (b *metricBuilder) currentDpGroupOrdered() []string {
	groupKeys := make([]string, len(b.currentDpGOrder))
	for k, v := range b.currentDpGOrder {
		groupKeys[v] = k
	}

	return groupKeys
}

func isUsefulLabel(mType metricspb.MetricDescriptor_Type, labelKey string) bool {
	result := false
	switch labelKey {
	case model.MetricNameLabel:
	case model.InstanceLabel:
	case model.SchemeLabel:
	case model.MetricsPathLabel:
	case model.JobLabel:
	case model.BucketLabel:
		result = mType != metricspb.MetricDescriptor_GAUGE_DISTRIBUTION &&
			mType != metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION
	case model.QuantileLabel:
		result = mType != metricspb.MetricDescriptor_SUMMARY
	default:
		result = true
	}
	return result
}

func dpgSignature(knownLabelKeys map[string]int, ls labels.Labels) string {
	sign := make([]string, len(knownLabelKeys))
	for k, v := range knownLabelKeys {
		sign[v] = ls.Get(k)
	}

	return fmt.Sprintf("%#v", sign)
}

func normalizeMetricName(name string) string {
	for _, s := range trimmableSuffixes {
		if strings.HasSuffix(name, s) && name != s {
			return strings.TrimSuffix(name, s)
		}
	}
	return name
}

func getBoundary(metricType metricspb.MetricDescriptor_Type, labels labels.Labels) (float64, error) {
	labelName := ""
	if metricType == metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION ||
		metricType == metricspb.MetricDescriptor_GAUGE_DISTRIBUTION {
		labelName = model.BucketLabel
	} else if metricType == metricspb.MetricDescriptor_SUMMARY {
		labelName = model.QuantileLabel
	} else {
		return 0, errNoBoundaryLabel
	}

	v := labels.Get(labelName)
	if v == "" {
		return 0, errEmptyBoundaryLabel
	}

	return strconv.ParseFloat(v, 64)
}

func convToOCAMetricType(metricType textparse.MetricType) metricspb.MetricDescriptor_Type {
	switch metricType {
	case textparse.MetricTypeCounter:
		// always use float64, as it's the internal data type used in prometheus
		return metricspb.MetricDescriptor_CUMULATIVE_DOUBLE
	case textparse.MetricTypeGauge:
		return metricspb.MetricDescriptor_GAUGE_DOUBLE
	case textparse.MetricTypeHistogram:
		return metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION
	case textparse.MetricTypeGaugeHistogram:
		return metricspb.MetricDescriptor_GAUGE_DISTRIBUTION
	case textparse.MetricTypeSummary:
		return metricspb.MetricDescriptor_SUMMARY
	default:
		// including: textparse.MetricTypeUnknown, textparse.MetricTypeInfo, textparse.MetricTypeStateset
		return metricspb.MetricDescriptor_UNSPECIFIED
	}
}

/*
   code borrowed from the original promreceiver
*/

func heuristicalMetricAndKnownUnits(metricName, parsedUnit string) string {
	if parsedUnit != "" {
		return parsedUnit
	}
	lastUnderscoreIndex := strings.LastIndex(metricName, "_")
	if lastUnderscoreIndex <= 0 || lastUnderscoreIndex >= len(metricName)-1 {
		return ""
	}

	unit := ""

	supposedUnit := metricName[lastUnderscoreIndex+1:]
	switch strings.ToLower(supposedUnit) {
	case "millisecond", "milliseconds", "ms":
		unit = "ms"
	case "second", "seconds", "s":
		unit = "s"
	case "microsecond", "microseconds", "us":
		unit = "us"
	case "nanosecond", "nanoseconds", "ns":
		unit = "ns"
	case "byte", "bytes", "by":
		unit = "By"
	case "bit", "bits":
		unit = "Bi"
	case "kilogram", "kilograms", "kg":
		unit = "kg"
	case "gram", "grams", "g":
		unit = "g"
	case "meter", "meters", "metre", "metres", "m":
		unit = "m"
	case "kilometer", "kilometers", "kilometre", "kilometres", "km":
		unit = "km"
	case "milimeter", "milimeters", "milimetre", "milimetres", "mm":
		unit = "mm"
	case "nanogram", "ng", "nanograms":
		unit = "ng"
	}

	return unit
}

func timestampFromMs(timeAtMs int64) *timestamp.Timestamp {
	secs, ns := timeAtMs/1e3, (timeAtMs%1e3)*1e6
	return &timestamp.Timestamp{
		Seconds: secs,
		Nanos:   int32(ns),
	}
}

func shouldSkip(metricName string) bool {
	if metricName == "up" || strings.HasPrefix(metricName, "scrape_") {
		return true
	}
	return false
}
