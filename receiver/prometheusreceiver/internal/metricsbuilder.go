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
	"strconv"
	"strings"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	metricsSuffixCount  = "_count"
	metricsSuffixBucket = "_bucket"
	metricsSuffixSum    = "_sum"
	startTimeMetricName = "process_start_time_seconds"
	scrapeUpMetricName  = "up"
)

var (
	trimmableSuffixes     = []string{metricsSuffixBucket, metricsSuffixCount, metricsSuffixSum}
	errNoDataToBuild      = errors.New("there's no data to build")
	errNoBoundaryLabel    = errors.New("given metricType has no BucketLabel or QuantileLabel")
	errEmptyBoundaryLabel = errors.New("BucketLabel or QuantileLabel is empty")
)

type metricBuilder struct {
	hasData              bool
	hasInternalMetric    bool
	mc                   MetadataCache
	metrics              []*metricspb.Metric
	numTimeseries        int
	droppedTimeseries    int
	useStartTimeMetric   bool
	startTimeMetricRegex *regexp.Regexp
	startTime            float64
	logger               *zap.Logger
	currentMf            MetricFamily
}

// newMetricBuilder creates a MetricBuilder which is allowed to feed all the datapoints from a single prometheus
// scraped page by calling its AddDataPoint function, and turn them into an opencensus data.MetricsData object
// by calling its Build function
func newMetricBuilder(mc MetadataCache, useStartTimeMetric bool, startTimeMetricRegex string, logger *zap.Logger) *metricBuilder {
	var regex *regexp.Regexp
	if startTimeMetricRegex != "" {
		regex, _ = regexp.Compile(startTimeMetricRegex)
	}
	return &metricBuilder{
		mc:                   mc,
		metrics:              make([]*metricspb.Metric, 0),
		logger:               logger,
		numTimeseries:        0,
		droppedTimeseries:    0,
		useStartTimeMetric:   useStartTimeMetric,
		startTimeMetricRegex: regex,
	}
}

func (b *metricBuilder) matchStartTimeMetric(metricName string) bool {
	if b.startTimeMetricRegex != nil {
		return b.startTimeMetricRegex.MatchString(metricName)
	}

	return metricName == startTimeMetricName
}

// AddDataPoint is for feeding prometheus data complexValue in its processing order
func (b *metricBuilder) AddDataPoint(ls labels.Labels, t int64, v float64) error {
	metricName := ls.Get(model.MetricNameLabel)
	switch {
	case metricName == "":
		b.numTimeseries++
		b.droppedTimeseries++
		return errMetricNameNotFound
	case isInternalMetric(metricName):
		b.hasInternalMetric = true
		lm := ls.Map()
		delete(lm, model.MetricNameLabel)
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
		return nil
	case b.useStartTimeMetric && b.matchStartTimeMetric(metricName):
		b.startTime = v
	}

	b.hasData = true

	if b.currentMf != nil && !b.currentMf.IsSameFamily(metricName) {
		m, ts, dts := b.currentMf.ToMetric()
		b.numTimeseries += ts
		b.droppedTimeseries += dts
		if m != nil {
			b.metrics = append(b.metrics, m)
		}
		b.currentMf = newMetricFamily(metricName, b.mc)
	} else if b.currentMf == nil {
		b.currentMf = newMetricFamily(metricName, b.mc)
	}

	return b.currentMf.Add(metricName, ls, t, v)
}

// Build an opencensus data.MetricsData based on all added data complexValue.
// The only error returned by this function is errNoDataToBuild.
func (b *metricBuilder) Build() ([]*metricspb.Metric, int, int, error) {
	if !b.hasData {
		if b.hasInternalMetric {
			return make([]*metricspb.Metric, 0), 0, 0, nil
		}
		return nil, 0, 0, errNoDataToBuild
	}

	if b.currentMf != nil {
		m, ts, dts := b.currentMf.ToMetric()
		b.numTimeseries += ts
		b.droppedTimeseries += dts
		if m != nil {
			b.metrics = append(b.metrics, m)
		}
		b.currentMf = nil
	}

	return b.metrics, b.numTimeseries, b.droppedTimeseries, nil
}

// TODO: move the following helper functions to a proper place, as they are not called directly in this go file

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

// dpgSignature is used to create a key for data complexValue belong to a same group of a metric family
func dpgSignature(orderedKnownLabelKeys []string, ls labels.Labels) string {
	sign := make([]string, 0, len(orderedKnownLabelKeys))
	for _, k := range orderedKnownLabelKeys {
		v := ls.Get(k)
		if v == "" {
			continue
		}
		sign = append(sign, k+"="+v)
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
	switch metricType {
	case metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
		metricspb.MetricDescriptor_GAUGE_DISTRIBUTION:
		labelName = model.BucketLabel
	case metricspb.MetricDescriptor_SUMMARY:
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

func convToOCAMetricType(metricType textparse.MetricType) metricspb.MetricDescriptor_Type {
	switch metricType {
	case textparse.MetricTypeCounter:
		// always use float64, as it's the internal data type used in prometheus
		return metricspb.MetricDescriptor_CUMULATIVE_DOUBLE
	// textparse.MetricTypeUnknown is converted to gauge by default to fix Prometheus untyped metrics from being dropped
	case textparse.MetricTypeGauge, textparse.MetricTypeUnknown:
		return metricspb.MetricDescriptor_GAUGE_DOUBLE
	case textparse.MetricTypeHistogram:
		return metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION
	// dropping support for gaugehistogram for now until we have an official spec of its implementation
	// a draft can be found in: https://docs.google.com/document/d/1KwV0mAXwwbvvifBvDKH_LU1YjyXE_wxCkHNoCGq1GX0/edit#heading=h.1cvzqd4ksd23
	// case textparse.MetricTypeGaugeHistogram:
	//	return metricspb.MetricDescriptor_GAUGE_DISTRIBUTION
	case textparse.MetricTypeSummary:
		return metricspb.MetricDescriptor_SUMMARY
	default:
		// including: textparse.MetricTypeInfo, textparse.MetricTypeStateset
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

func timestampFromMs(timeAtMs int64) *timestamppb.Timestamp {
	secs, ns := timeAtMs/1e3, (timeAtMs%1e3)*1e6
	return &timestamppb.Timestamp{
		Seconds: secs,
		Nanos:   int32(ns),
	}
}

func isInternalMetric(metricName string) bool {
	if metricName == scrapeUpMetricName || strings.HasPrefix(metricName, "scrape_") {
		return true
	}
	return false
}
