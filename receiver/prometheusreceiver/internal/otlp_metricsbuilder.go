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
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/value"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/model/pdata"
)

const (
	metricsSuffixCount  = "_count"
	metricsSuffixBucket = "_bucket"
	metricsSuffixSum    = "_sum"
	metricSuffixTotal   = "_total"
	startTimeMetricName = "process_start_time_seconds"
	scrapeUpMetricName  = "up"
)

var (
	trimmableSuffixes     = []string{metricsSuffixBucket, metricsSuffixCount, metricsSuffixSum, metricSuffixTotal}
	errNoDataToBuild      = errors.New("there's no data to build")
	errNoBoundaryLabel    = errors.New("given metricType has no BucketLabel or QuantileLabel")
	errEmptyBoundaryLabel = errors.New("BucketLabel or QuantileLabel is empty")
)

// dpgSignature is used to create a key for data complexValue belong to a same group of a metric family
func dpgSignature(orderedKnownLabelKeys []string, ls labels.Labels) string {
	size := 0
	for _, k := range orderedKnownLabelKeys {
		v := ls.Get(k)
		if v == "" {
			continue
		}
		// 2 enclosing quotes + 1 equality sign = 3 extra chars.
		// Note: if any character in the label value requires escaping,
		// we'll need more space than that, which will lead to some
		// extra allocation.
		size += 3 + len(k) + len(v)
	}
	sign := make([]byte, 0, size)
	for _, k := range orderedKnownLabelKeys {
		v := ls.Get(k)
		if v == "" {
			continue
		}
		sign = strconv.AppendQuote(sign, k+"="+v)
	}
	return string(sign)
}

func normalizeMetricName(name string) string {
	for _, s := range trimmableSuffixes {
		if strings.HasSuffix(name, s) && name != s {
			return strings.TrimSuffix(name, s)
		}
	}
	return name
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

func isInternalMetric(metricName string) bool {
	if metricName == scrapeUpMetricName || strings.HasPrefix(metricName, "scrape_") {
		return true
	}
	return false
}
func isUsefulLabelPdata(mType pdata.MetricDataType, labelKey string) bool {
	switch labelKey {
	case model.MetricNameLabel, model.InstanceLabel, model.SchemeLabel, model.MetricsPathLabel, model.JobLabel:
		return false
	case model.BucketLabel:
		return mType != pdata.MetricDataTypeHistogram
	case model.QuantileLabel:
		return mType != pdata.MetricDataTypeSummary
	}
	return true
}

func getBoundaryPdata(metricType pdata.MetricDataType, labels labels.Labels) (float64, error) {
	labelName := ""
	switch metricType {
	case pdata.MetricDataTypeHistogram:
		labelName = model.BucketLabel
	case pdata.MetricDataTypeSummary:
		labelName = model.QuantileLabel
	default:
		return 0, errNoBoundaryLabel
	}

	v := labels.Get(labelName)
	if v == "" {
		return 0, errEmptyBoundaryLabel
	}

	return strconv.ParseFloat(v, 64)
}

func convToPdataMetricType(metricType textparse.MetricType) pdata.MetricDataType {
	switch metricType {
	case textparse.MetricTypeCounter: // metricspb.MetricDescriptor_CUMULATIVE_DOUBLE
		// always use float64, as it's the internal data type used in prometheus
		return pdata.MetricDataTypeSum
	// textparse.MetricTypeUnknown is converted to gauge by default to fix Prometheus untyped metrics from being dropped
	case textparse.MetricTypeGauge, textparse.MetricTypeUnknown: // metricspb.MetricDescriptor_GAUGE_DOUBLE
		return pdata.MetricDataTypeGauge
	case textparse.MetricTypeHistogram: // metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION
		return pdata.MetricDataTypeHistogram
	// dropping support for gaugehistogram for now until we have an official spec of its implementation
	// a draft can be found in: https://docs.google.com/document/d/1KwV0mAXwwbvvifBvDKH_LU1YjyXE_wxCkHNoCGq1GX0/edit#heading=h.1cvzqd4ksd23
	// case textparse.MetricTypeGaugeHistogram:
	//	return metricspb.MetricDescriptor_GAUGE_DISTRIBUTION
	case textparse.MetricTypeSummary:
		return pdata.MetricDataTypeSummary
	default:
		// including: textparse.MetricTypeInfo, textparse.MetricTypeStateset
		return pdata.MetricDataTypeNone
	}
}

type metricBuilderPdata struct {
	metrics              pdata.MetricSlice
	currentMf            MetricFamilyPdata
	hasData              bool
	hasInternalMetric    bool
	mc                   MetadataCache
	numTimeseries        int
	droppedTimeseries    int
	useStartTimeMetric   bool
	startTimeMetricRegex *regexp.Regexp
	startTime            float64
	intervalStartTimeMs  int64
	logger               *zap.Logger
	stalenessStore       *stalenessStore
}

// newMetricBuilder creates a MetricBuilder which is allowed to feed all the datapoints from a single prometheus
// scraped page by calling its AddDataPoint function, and turn them into a pdata.Metrics object.
// by calling its Build function
func newMetricBuilderPdata(mc MetadataCache, useStartTimeMetric bool, startTimeMetricRegex string, logger *zap.Logger, stalenessStore *stalenessStore) *metricBuilderPdata {
	var regex *regexp.Regexp
	if startTimeMetricRegex != "" {
		regex, _ = regexp.Compile(startTimeMetricRegex)
	}
	return &metricBuilderPdata{
		metrics:              pdata.NewMetricSlice(),
		mc:                   mc,
		logger:               logger,
		numTimeseries:        0,
		droppedTimeseries:    0,
		useStartTimeMetric:   useStartTimeMetric,
		startTimeMetricRegex: regex,
		stalenessStore:       stalenessStore,
	}
}

func (b *metricBuilderPdata) matchStartTimeMetric(metricName string) bool {
	if b.startTimeMetricRegex != nil {
		return b.startTimeMetricRegex.MatchString(metricName)
	}

	return metricName == startTimeMetricName
}

// AddDataPoint is for feeding prometheus data complexValue in its processing order
func (b *metricBuilderPdata) AddDataPoint(ls labels.Labels, t int64, v float64) (rerr error) {
	// Any datapoint with duplicate labels MUST be rejected per:
	// * https://github.com/open-telemetry/wg-prometheus/issues/44
	// * https://github.com/open-telemetry/opentelemetry-collector/issues/3407
	// as Prometheus rejects such too as of version 2.16.0, released on 2020-02-13.
	seen := make(map[string]bool)
	dupLabels := make([]string, 0, len(ls))
	for _, label := range ls {
		if _, ok := seen[label.Name]; ok {
			dupLabels = append(dupLabels, label.Name)
		}
		seen[label.Name] = true
	}
	if len(dupLabels) != 0 {
		sort.Strings(dupLabels)
		return fmt.Errorf("invalid sample: non-unique label names: %q", dupLabels)
	}

	defer func() {
		// Only mark this data point as in the current scrape
		// iff it isn't a stale metric.
		if rerr == nil && !value.IsStaleNaN(v) {
			b.stalenessStore.markAsCurrentlySeen(ls, t)
		}
	}()

	metricName := ls.Get(model.MetricNameLabel)
	switch {
	case metricName == "":
		b.numTimeseries++
		b.droppedTimeseries++
		return errMetricNameNotFound
	case isInternalMetric(metricName):
		b.hasInternalMetric = true
		lm := ls.Map()
		// See https://www.prometheus.io/docs/concepts/jobs_instances/#automatically-generated-labels-and-time-series
		// up: 1 if the instance is healthy, i.e. reachable, or 0 if the scrape failed.
		if metricName == scrapeUpMetricName && v != 1.0 {
			if v == 0.0 {
				b.logger.Warn("Failed to scrape Prometheus endpoint",
					zap.Int64("scrape_timestamp", t),
					zap.String("target_labels", fmt.Sprintf("%v", lm)))
			} else {
				b.logger.Warn("The 'up' metric contains invalid value",
					zap.Float64("value", v),
					zap.Int64("scrape_timestamp", t),
					zap.String("target_labels", fmt.Sprintf("%v", lm)))
			}
		}
	case b.useStartTimeMetric && b.matchStartTimeMetric(metricName):
		b.startTime = v
	}

	b.hasData = true

	if b.currentMf != nil && !b.currentMf.IsSameFamily(metricName) {
		nTs, nDts := b.currentMf.ToMetricPdata(&b.metrics)
		b.numTimeseries += nTs
		b.droppedTimeseries += nDts
		b.currentMf = newMetricFamilyPdata(metricName, b.mc, b.logger, b.intervalStartTimeMs)
	} else if b.currentMf == nil {
		b.currentMf = newMetricFamilyPdata(metricName, b.mc, b.logger, b.intervalStartTimeMs)
	}

	return b.currentMf.Add(metricName, ls, t, v)
}

// Build an pdata.MetricSlice based on all added data complexValue.
// The only error returned by this function is errNoDataToBuild.
func (b *metricBuilderPdata) Build() (*pdata.MetricSlice, int, int, error) {
	if !b.hasData {
		if b.hasInternalMetric {
			metricsL := pdata.NewMetricSlice()
			return &metricsL, 0, 0, nil
		}
		return nil, 0, 0, errNoDataToBuild
	}

	if b.currentMf != nil {
		ts, dts := b.currentMf.ToMetricPdata(&b.metrics)
		b.numTimeseries += ts
		b.droppedTimeseries += dts
		b.currentMf = nil
	}

	return &b.metrics, b.numTimeseries, b.droppedTimeseries, nil
}
