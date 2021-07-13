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

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/model/pdata"
)

const (
	nameStr     = "__name__"
	sumStr      = "_sum"
	countStr    = "_count"
	bucketStr   = "_bucket"
	leStr       = "le"
	quantileStr = "quantile"
	pInfStr     = "+Inf"
	keyStr      = "key"
)

// ByLabelName enables the usage of sort.Sort() with a slice of labels
type ByLabelName []prompb.Label

func (a ByLabelName) Len() int           { return len(a) }
func (a ByLabelName) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a ByLabelName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// validateMetrics returns a bool representing whether the metric has a valid type and temporality combination and a
// matching metric type and field
func validateMetrics(metric pdata.Metric) bool {
	switch metric.DataType() {
	case pdata.MetricDataTypeGauge:
		return metric.Gauge().DataPoints().Len() != 0
	case pdata.MetricDataTypeIntGauge:
		return metric.IntGauge().DataPoints().Len() != 0
	case pdata.MetricDataTypeSum:
		return metric.Sum().DataPoints().Len() != 0 && metric.Sum().AggregationTemporality() == pdata.AggregationTemporalityCumulative
	case pdata.MetricDataTypeIntSum:
		return metric.IntSum().DataPoints().Len() != 0 && metric.IntSum().AggregationTemporality() == pdata.AggregationTemporalityCumulative
	case pdata.MetricDataTypeHistogram:
		return metric.Histogram().DataPoints().Len() != 0 && metric.Histogram().AggregationTemporality() == pdata.AggregationTemporalityCumulative
	case pdata.MetricDataTypeIntHistogram:
		return metric.IntHistogram().DataPoints().Len() != 0 && metric.IntHistogram().AggregationTemporality() == pdata.AggregationTemporalityCumulative
	case pdata.MetricDataTypeSummary:
		return metric.Summary().DataPoints().Len() != 0
	}
	return false
}

// addSample finds a TimeSeries in tsMap that corresponds to the label set labels, and add sample to the TimeSeries; it
// creates a new TimeSeries in the map if not found. tsMap is unmodified if either of its parameters is nil.
func addSample(tsMap map[string]*prompb.TimeSeries, sample *prompb.Sample, labels []prompb.Label,
	metric pdata.Metric) {

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
func timeSeriesSignature(metric pdata.Metric, labels *[]prompb.Label) string {
	b := strings.Builder{}
	b.WriteString(metric.DataType().String())

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
func createLabelSet(resource pdata.Resource, labels pdata.StringMap, externalLabels map[string]string, extras ...string) []prompb.Label {
	// map ensures no duplicate label name
	l := map[string]prompb.Label{}

	for key, value := range externalLabels {
		// External labels have already been sanitized
		l[key] = prompb.Label{
			Name:  key,
			Value: value,
		}
	}

	resource.Attributes().Range(func(key string, value pdata.AttributeValue) bool {
		if isUsefulResourceAttribute(key) {
			l[key] = prompb.Label{
				Name:  sanitize(key),
				Value: value.StringVal(), // TODO(jbd): Decide what to do with non-string attributes.
			}
		}

		return true
	})

	labels.Range(func(key string, value string) bool {
		l[key] = prompb.Label{
			Name:  sanitize(key),
			Value: value,
		}

		return true
	})

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

func isUsefulResourceAttribute(key string) bool {
	// TODO(jbd): Allow users to configure what other resource
	// attributes to be included.
	// Decide what to do with non-string attributes.
	// We should always output "job" and "instance".
	switch key {
	case model.InstanceLabel:
		return true
	case model.JobLabel:
		return true
	}
	return false
}

// getPromMetricName creates a Prometheus metric name by attaching namespace prefix for Monotonic metrics.
func getPromMetricName(metric pdata.Metric, ns string) string {
	name := metric.Name()
	if len(ns) > 0 {
		name = ns + "_" + name
	}

	return sanitize(name)
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
func convertTimeStamp(timestamp pdata.Timestamp) int64 {
	return timestamp.AsTime().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
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
		s = keyStr + "_" + s
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

// addSingleDoubleDataPoint converts the metric value stored in pt to a Prometheus sample, and add the sample
// to its corresponding time series in tsMap
func addSingleDoubleDataPoint(pt pdata.DoubleDataPoint, resource pdata.Resource, metric pdata.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	// create parameters for addSample
	name := getPromMetricName(metric, namespace)
	labels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, name)
	sample := &prompb.Sample{
		Value: pt.Value(),
		// convert ns to ms
		Timestamp: convertTimeStamp(pt.Timestamp()),
	}
	addSample(tsMap, sample, labels, metric)
}

// addSingleIntDataPoint converts the metric value stored in pt to a Prometheus sample, and add the sample
// to its corresponding time series in tsMap
func addSingleIntDataPoint(pt pdata.IntDataPoint, resource pdata.Resource, metric pdata.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	// create parameters for addSample
	name := getPromMetricName(metric, namespace)
	labels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, name)
	sample := &prompb.Sample{
		Value: float64(pt.Value()),
		// convert ns to ms
		Timestamp: convertTimeStamp(pt.Timestamp()),
	}
	addSample(tsMap, sample, labels, metric)
}

// addSingleIntHistogramDataPoint converts pt to 2 + min(len(ExplicitBounds), len(BucketCount)) + 1 samples. It
// ignore extra buckets if len(ExplicitBounds) > len(BucketCounts)
func addSingleIntHistogramDataPoint(pt pdata.IntHistogramDataPoint, resource pdata.Resource, metric pdata.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	time := convertTimeStamp(pt.Timestamp())
	// sum, count, and buckets of the histogram should append suffix to baseName
	baseName := getPromMetricName(metric, namespace)
	// treat sum as a sample in an individual TimeSeries
	sum := &prompb.Sample{
		Value:     float64(pt.Sum()),
		Timestamp: time,
	}

	sumlabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+sumStr)
	addSample(tsMap, sum, sumlabels, metric)

	// treat count as a sample in an individual TimeSeries
	count := &prompb.Sample{
		Value:     float64(pt.Count()),
		Timestamp: time,
	}
	countlabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+countStr)
	addSample(tsMap, count, countlabels, metric)

	// cumulative count for conversion to cumulative histogram
	var cumulativeCount uint64

	// process each bound, ignore extra bucket values
	for index, bound := range pt.ExplicitBounds() {
		if index >= len(pt.BucketCounts()) {
			break
		}
		cumulativeCount += pt.BucketCounts()[index]
		bucket := &prompb.Sample{
			Value:     float64(cumulativeCount),
			Timestamp: time,
		}
		boundStr := strconv.FormatFloat(bound, 'f', -1, 64)
		labels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+bucketStr, leStr, boundStr)
		addSample(tsMap, bucket, labels, metric)
	}
	// add le=+Inf bucket
	cumulativeCount += pt.BucketCounts()[len(pt.BucketCounts())-1]
	infBucket := &prompb.Sample{
		Value:     float64(cumulativeCount),
		Timestamp: time,
	}
	infLabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+bucketStr, leStr, pInfStr)
	addSample(tsMap, infBucket, infLabels, metric)
}

// addSingleHistogramDataPoint converts pt to 2 + min(len(ExplicitBounds), len(BucketCount)) + 1 samples. It
// ignore extra buckets if len(ExplicitBounds) > len(BucketCounts)
func addSingleHistogramDataPoint(pt pdata.HistogramDataPoint, resource pdata.Resource, metric pdata.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	time := convertTimeStamp(pt.Timestamp())
	// sum, count, and buckets of the histogram should append suffix to baseName
	baseName := getPromMetricName(metric, namespace)
	// treat sum as a sample in an individual TimeSeries
	sum := &prompb.Sample{
		Value:     pt.Sum(),
		Timestamp: time,
	}

	sumlabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+sumStr)
	addSample(tsMap, sum, sumlabels, metric)

	// treat count as a sample in an individual TimeSeries
	count := &prompb.Sample{
		Value:     float64(pt.Count()),
		Timestamp: time,
	}
	countlabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+countStr)
	addSample(tsMap, count, countlabels, metric)

	// cumulative count for conversion to cumulative histogram
	var cumulativeCount uint64

	// process each bound, based on histograms proto definition, # of buckets = # of explicit bounds + 1
	for index, bound := range pt.ExplicitBounds() {
		if index >= len(pt.BucketCounts()) {
			break
		}
		cumulativeCount += pt.BucketCounts()[index]
		bucket := &prompb.Sample{
			Value:     float64(cumulativeCount),
			Timestamp: time,
		}
		boundStr := strconv.FormatFloat(bound, 'f', -1, 64)
		labels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+bucketStr, leStr, boundStr)
		addSample(tsMap, bucket, labels, metric)
	}
	// add le=+Inf bucket
	cumulativeCount += pt.BucketCounts()[len(pt.BucketCounts())-1]
	infBucket := &prompb.Sample{
		Value:     float64(cumulativeCount),
		Timestamp: time,
	}
	infLabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+bucketStr, leStr, pInfStr)
	addSample(tsMap, infBucket, infLabels, metric)
}

// addSingleSummaryDataPoint converts pt to len(QuantileValues) + 2 samples.
func addSingleSummaryDataPoint(pt pdata.SummaryDataPoint, resource pdata.Resource, metric pdata.Metric, namespace string,
	tsMap map[string]*prompb.TimeSeries, externalLabels map[string]string) {
	time := convertTimeStamp(pt.Timestamp())
	// sum and count of the summary should append suffix to baseName
	baseName := getPromMetricName(metric, namespace)
	// treat sum as a sample in an individual TimeSeries
	sum := &prompb.Sample{
		Value:     pt.Sum(),
		Timestamp: time,
	}

	sumlabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+sumStr)
	addSample(tsMap, sum, sumlabels, metric)

	// treat count as a sample in an individual TimeSeries
	count := &prompb.Sample{
		Value:     float64(pt.Count()),
		Timestamp: time,
	}
	countlabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName+countStr)
	addSample(tsMap, count, countlabels, metric)

	// process each percentile/quantile
	for i := 0; i < pt.QuantileValues().Len(); i++ {
		qt := pt.QuantileValues().At(i)
		quantile := &prompb.Sample{
			Value:     qt.Value(),
			Timestamp: time,
		}
		percentileStr := strconv.FormatFloat(qt.Quantile(), 'f', -1, 64)
		qtlabels := createLabelSet(resource, pt.LabelsMap(), externalLabels, nameStr, baseName, quantileStr, percentileStr)
		addSample(tsMap, quantile, qtlabels, metric)
	}
}

func orderBySampleTimestamp(tsArray []prompb.TimeSeries) []prompb.TimeSeries {
	for i := range tsArray {
		sL := tsArray[i].Samples
		sort.Slice(sL, func(i, j int) bool {
			return sL[i].Timestamp < sL[j].Timestamp
		})
	}
	return tsArray
}

func convertTimeseriesToRequest(tsArray []prompb.TimeSeries) *prompb.WriteRequest {
	// the remote_write endpoint only requires the timeseries.
	// otlp defines it's own way to handle metric metadata
	return &prompb.WriteRequest{
		// Prometheus requires time series to be sorted by Timestamp to avoid out of order problems.
		// See:
		// * https://github.com/open-telemetry/wg-prometheus/issues/10
		// * https://github.com/open-telemetry/opentelemetry-collector/issues/2315
		Timeseries: orderBySampleTimestamp(tsArray),
	}
}
