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

package prometheusremotewriteexporter

import (
	"errors"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/consumer/pdata"
	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

const (
	nameStr     = "__name__"
	sumStr      = "_sum"
	countStr    = "_count"
	bucketStr   = "_bucket"
	leStr       = "le"
	quantileStr = "quantile"
	pInfStr     = "+Inf"
	totalStr    = "total"
	delimeter   = "_"
	keyStr      = "key"
)

// ByLabelName enables the usage of sort.Sort() with a slice of labels
type ByLabelName []prompb.Label

func (a ByLabelName) Len() int           { return len(a) }
func (a ByLabelName) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a ByLabelName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// validateMetrics returns a bool representing whether the metric has a valid type and temporality combination and a
// matching metric type and field
func validateMetrics(metric *otlp.Metric) bool {
	if metric == nil || metric.Data == nil {
		return false
	}
	switch metric.Data.(type) {
	case *otlp.Metric_DoubleGauge:
		return metric.GetDoubleGauge() != nil
	case *otlp.Metric_IntGauge:
		return metric.GetIntGauge() != nil
	case *otlp.Metric_DoubleSum:
		return metric.GetDoubleSum() != nil && metric.GetDoubleSum().GetAggregationTemporality() ==
			otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
	case *otlp.Metric_IntSum:
		return metric.GetIntSum() != nil && metric.GetIntSum().GetAggregationTemporality() ==
			otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
	case *otlp.Metric_DoubleHistogram:
		return metric.GetDoubleHistogram() != nil && metric.GetDoubleHistogram().GetAggregationTemporality() ==
			otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
	case *otlp.Metric_IntHistogram:
		return metric.GetIntHistogram() != nil && metric.GetIntHistogram().GetAggregationTemporality() ==
			otlp.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
	case *otlp.Metric_DoubleSummary:
		return metric.GetDoubleSummary() != nil
	}
	return false
}

// addSample finds a TimeSeries in tsMap that corresponds to the label set labels, and add sample to the TimeSeries; it
// creates a new TimeSeries in the map if not found. tsMap is unmodified if either of its parameters is nil.
func addSample(tsMap map[string]*prompb.TimeSeries, sample *prompb.Sample, labels []prompb.Label,
	metric *otlp.Metric) {

	if sample == nil || labels == nil || tsMap == nil {
		return
	}

	sig := timeSeriesSignature(metric, &labels)
	ts, ok := tsMap[sig]

	if ok {
		ts.Samples = append(ts.Samples, *sample)
	} else {
		newTs := &prompb.TimeSeries{
			Labels:  labels,
			Samples: []prompb.Sample{*sample},
		}
		tsMap[sig] = newTs
	}
}

// timeSeries return a string signature in the form of:
// 		TYPE-label1-value1- ...  -labelN-valueN
// the label slice should not contain duplicate label names; this method sorts the slice by label name before creating
// the signature.
func timeSeriesSignature(metric *otlp.Metric, labels *[]prompb.Label) string {
	b := strings.Builder{}
	b.WriteString(getTypeString(metric))

	sort.Sort(ByLabelName(*labels))

	for _, lb := range *labels {
		b.WriteString("-")
		b.WriteString(lb.GetName())
		b.WriteString("-")
		b.WriteString(lb.GetValue())
	}

	return b.String()
}

// createLabelSet creates a slice of Cortex Label with OTLP labels and paris of string values.
// Unpaired string value is ignored. String pairs overwrites OTLP labels if collision happens, and the overwrite is
// logged. Resultant label names are sanitized.
func createLabelSet(labels []common.StringKeyValue, externalLabels map[string]string, extras ...string) []prompb.Label {
	// map ensures no duplicate label name
	l := map[string]prompb.Label{}

	for key, value := range externalLabels {
		// External labels have already been sanitized
		l[key] = prompb.Label{
			Name:  key,
			Value: value,
		}
	}

	for _, lb := range labels {
		l[lb.Key] = prompb.Label{
			Name:  sanitize(lb.Key),
			Value: lb.Value,
		}
	}

	for i := 0; i < len(extras); i += 2 {
		if i+1 >= len(extras) {
			break
		}
		_, found := l[extras[i]]
		if found {
			log.Println("label " + extras[i] + " is overwritten. Check if Prometheus reserved labels are used.")
		}
		// internal labels should be maintained
		name := extras[i]
		if !(len(name) > 4 && name[:2] == "__" && name[len(name)-2:] == "__") {
			name = sanitize(name)
		}
		l[extras[i]] = prompb.Label{
			Name:  name,
			Value: extras[i+1],
		}
	}

	s := make([]prompb.Label, 0, len(l))
	for _, lb := range l {
		s = append(s, lb)
	}

	return s
}

// getPromMetricName creates a Prometheus metric name by attaching namespace prefix, and _total suffix for Monotonic
// metrics.
func getPromMetricName(metric *otlp.Metric, ns string) string {
	if metric == nil {
		return ""
	}

	// if the metric is counter, _total suffix should be applied
	_, isCounter1 := metric.Data.(*otlp.Metric_DoubleSum)
	_, isCounter2 := metric.Data.(*otlp.Metric_IntSum)
	isCounter := isCounter1 || isCounter2

	b := strings.Builder{}

	b.WriteString(ns)

	if b.Len() > 0 {
		b.WriteString(delimeter)
	}
	name := metric.GetName()
	b.WriteString(name)

	// do not add the total suffix if the metric name already ends in "total"
	isCounter = isCounter && name[len(name)-len(totalStr):] != totalStr

	// Including units makes two metrics with the same name and label set belong to two different TimeSeries if the
	// units are different.
	/*
		if b.Len() > 0 && len(desc.GetUnit()) > 0{
			fmt.Fprintf(&b, delimeter)
			fmt.Fprintf(&b, desc.GetUnit())
		}
	*/

	if b.Len() > 0 && isCounter {
		b.WriteString(delimeter)
		b.WriteString(totalStr)
	}
	return sanitize(b.String())
}

// batchTimeSeries splits series into multiple batch write requests.
func batchTimeSeries(tsMap map[string]*prompb.TimeSeries, maxBatchByteSize int) ([]*prompb.WriteRequest, error) {
	if len(tsMap) == 0 {
		return nil, errors.New("invalid tsMap: cannot be empty map")
	}

	var requests []*prompb.WriteRequest
	var tsArray []prompb.TimeSeries
	sizeOfCurrentBatch := 0

	for _, v := range tsMap {
		sizeOfSeries := v.Size()

		if sizeOfCurrentBatch+sizeOfSeries >= maxBatchByteSize {
			wrapped := convertTimeseriesToRequest(tsArray)
			requests = append(requests, wrapped)

			tsArray = make([]prompb.TimeSeries, 0)
			sizeOfCurrentBatch = 0
		}

		tsArray = append(tsArray, *v)
		sizeOfCurrentBatch += sizeOfSeries
	}

	if len(tsArray) != 0 {
		wrapped := convertTimeseriesToRequest(tsArray)
		requests = append(requests, wrapped)
	}

	return requests, nil
}

// convertTimeStamp converts OTLP timestamp in ns to timestamp in ms
func convertTimeStamp(timestamp uint64) int64 {
	return int64(timestamp / uint64(int64(time.Millisecond)/int64(time.Nanosecond)))
}

// copied from prometheus-go-metric-exporter
// sanitize replaces non-alphanumeric characters with underscores in s.
func sanitize(s string) string {
	if len(s) == 0 {
		return s
	}

	// Note: No length limit for label keys because Prometheus doesn't
	// define a length limit, thus we should NOT be truncating label keys.
	// See https://github.com/orijtech/prometheus-go-metrics-exporter/issues/4.
	s = strings.Map(sanitizeRune, s)
	if unicode.IsDigit(rune(s[0])) {
		s = keyStr + delimeter + s
	}
	if s[0] == '_' {
		s = keyStr + s
	}
	return s
}

// copied from prometheus-go-metric-exporter
// sanitizeRune converts anything that is not a letter or digit to an underscore
func sanitizeRune(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	// Everything else turns into an underscore
	return '_'
}

func getTypeString(metric *otlp.Metric) string {
	switch metric.Data.(type) {
	case *otlp.Metric_DoubleGauge:
		return strconv.Itoa(int(pdata.MetricDataTypeDoubleGauge))
	case *otlp.Metric_IntGauge:
		return strconv.Itoa(int(pdata.MetricDataTypeIntGauge))
	case *otlp.Metric_DoubleSum:
		return strconv.Itoa(int(pdata.MetricDataTypeDoubleSum))
	case *otlp.Metric_IntSum:
		return strconv.Itoa(int(pdata.MetricDataTypeIntSum))
	case *otlp.Metric_DoubleHistogram:
		return strconv.Itoa(int(pdata.MetricDataTypeDoubleHistogram))
	case *otlp.Metric_IntHistogram:
		return strconv.Itoa(int(pdata.MetricDataTypeIntHistogram))
	}
	return ""
}

// addSingleDoubleDataPoint converts the metric value stored in pt to a Prometheus sample, and add the sample
// to its corresponding time series in tsMap
func addSingleDoubleDataPoint(pt *otlp.DoubleDataPoint, metric *otlp.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	if pt == nil {
		return
	}
	// create parameters for addSample
	name := getPromMetricName(metric, namespace)
	labels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, name)
	sample := &prompb.Sample{
		Value: pt.Value,
		// convert ns to ms
		Timestamp: convertTimeStamp(pt.TimeUnixNano),
	}
	addSample(tsMap, sample, labels, metric)
}

// addSingleIntDataPoint converts the metric value stored in pt to a Prometheus sample, and add the sample
// to its corresponding time series in tsMap
func addSingleIntDataPoint(pt *otlp.IntDataPoint, metric *otlp.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	if pt == nil {
		return
	}
	// create parameters for addSample
	name := getPromMetricName(metric, namespace)
	labels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, name)
	sample := &prompb.Sample{
		Value: float64(pt.Value),
		// convert ns to ms
		Timestamp: convertTimeStamp(pt.TimeUnixNano),
	}
	addSample(tsMap, sample, labels, metric)
}

// addSingleIntHistogramDataPoint converts pt to 2 + min(len(ExplicitBounds), len(BucketCount)) + 1 samples. It
// ignore extra buckets if len(ExplicitBounds) > len(BucketCounts)
func addSingleIntHistogramDataPoint(pt *otlp.IntHistogramDataPoint, metric *otlp.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	if pt == nil {
		return
	}
	time := convertTimeStamp(pt.TimeUnixNano)
	// sum, count, and buckets of the histogram should append suffix to baseName
	baseName := getPromMetricName(metric, namespace)
	// treat sum as a sample in an individual TimeSeries
	sum := &prompb.Sample{
		Value:     float64(pt.GetSum()),
		Timestamp: time,
	}

	sumlabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+sumStr)
	addSample(tsMap, sum, sumlabels, metric)

	// treat count as a sample in an individual TimeSeries
	count := &prompb.Sample{
		Value:     float64(pt.GetCount()),
		Timestamp: time,
	}
	countlabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+countStr)
	addSample(tsMap, count, countlabels, metric)

	// cumulative count for conversion to cumulative histogram
	var cumulativeCount uint64

	// process each bound, ignore extra bucket values
	for index, bound := range pt.GetExplicitBounds() {
		if index >= len(pt.GetBucketCounts()) {
			break
		}
		cumulativeCount += pt.GetBucketCounts()[index]
		bucket := &prompb.Sample{
			Value:     float64(cumulativeCount),
			Timestamp: time,
		}
		boundStr := strconv.FormatFloat(bound, 'f', -1, 64)
		labels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+bucketStr, leStr, boundStr)
		addSample(tsMap, bucket, labels, metric)
	}
	// add le=+Inf bucket
	cumulativeCount += pt.GetBucketCounts()[len(pt.GetBucketCounts())-1]
	infBucket := &prompb.Sample{
		Value:     float64(cumulativeCount),
		Timestamp: time,
	}
	infLabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+bucketStr, leStr, pInfStr)
	addSample(tsMap, infBucket, infLabels, metric)
}

// addSingleDoubleHistogramDataPoint converts pt to 2 + min(len(ExplicitBounds), len(BucketCount)) + 1 samples. It
// ignore extra buckets if len(ExplicitBounds) > len(BucketCounts)
func addSingleDoubleHistogramDataPoint(pt *otlp.DoubleHistogramDataPoint, metric *otlp.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	if pt == nil {
		return
	}
	time := convertTimeStamp(pt.TimeUnixNano)
	// sum, count, and buckets of the histogram should append suffix to baseName
	baseName := getPromMetricName(metric, namespace)
	// treat sum as a sample in an individual TimeSeries
	sum := &prompb.Sample{
		Value:     pt.GetSum(),
		Timestamp: time,
	}

	sumlabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+sumStr)
	addSample(tsMap, sum, sumlabels, metric)

	// treat count as a sample in an individual TimeSeries
	count := &prompb.Sample{
		Value:     float64(pt.GetCount()),
		Timestamp: time,
	}
	countlabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+countStr)
	addSample(tsMap, count, countlabels, metric)

	// cumulative count for conversion to cumulative histogram
	var cumulativeCount uint64

	// process each bound, based on histograms proto definition, # of buckets = # of explicit bounds + 1
	for index, bound := range pt.GetExplicitBounds() {
		if index >= len(pt.GetBucketCounts()) {
			break
		}
		cumulativeCount += pt.GetBucketCounts()[index]
		bucket := &prompb.Sample{
			Value:     float64(cumulativeCount),
			Timestamp: time,
		}
		boundStr := strconv.FormatFloat(bound, 'f', -1, 64)
		labels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+bucketStr, leStr, boundStr)
		addSample(tsMap, bucket, labels, metric)
	}
	// add le=+Inf bucket
	cumulativeCount += pt.GetBucketCounts()[len(pt.GetBucketCounts())-1]
	infBucket := &prompb.Sample{
		Value:     float64(cumulativeCount),
		Timestamp: time,
	}
	infLabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+bucketStr, leStr, pInfStr)
	addSample(tsMap, infBucket, infLabels, metric)
}

// addSingleDoubleSummaryDataPoint converts pt to len(QuantileValues) + 2 samples.
func addSingleDoubleSummaryDataPoint(pt *otlp.DoubleSummaryDataPoint, metric *otlp.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	if pt == nil {
		return
	}
	time := convertTimeStamp(pt.TimeUnixNano)
	// sum and count of the summary should append suffix to baseName
	baseName := getPromMetricName(metric, namespace)
	// treat sum as a sample in an individual TimeSeries
	sum := &prompb.Sample{
		Value:     pt.GetSum(),
		Timestamp: time,
	}

	sumlabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+sumStr)
	addSample(tsMap, sum, sumlabels, metric)

	// treat count as a sample in an individual TimeSeries
	count := &prompb.Sample{
		Value:     float64(pt.GetCount()),
		Timestamp: time,
	}
	countlabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName+countStr)
	addSample(tsMap, count, countlabels, metric)

	// process each percentile/quantile
	for _, qt := range pt.GetQuantileValues() {
		quantile := &prompb.Sample{
			Value:     qt.Value,
			Timestamp: time,
		}
		percentileStr := strconv.FormatFloat(qt.GetQuantile(), 'f', -1, 64)
		qtlabels := createLabelSet(pt.GetLabels(), externalLabels, nameStr, baseName, quantileStr, percentileStr)
		addSample(tsMap, quantile, qtlabels, metric)
	}
}

func convertTimeseriesToRequest(tsArray []prompb.TimeSeries) *prompb.WriteRequest {
	// the remote_write endpoint only requires the timeseries.
	// otlp defines it's own way to handle metric metadata
	return &prompb.WriteRequest{
		Timeseries: tsArray,
	}
}
