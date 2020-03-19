// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internaldata

import (
	"sort"

	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

const (
	invalidMetricDescriptorType = ocmetrics.MetricDescriptor_Type(-1)
)

type labelKeys struct {
	// ordered OC label keys
	keys []*ocmetrics.LabelKey
	// map from a label key literal
	// to its index in the slice above
	keyIndices map[string]int
}

func MetricDataToOC(td data.MetricData) []consumerdata.MetricsData {
	ocMetricsData := consumerdata.MetricsData{}

	resourceMetricsList := td.ResourceMetrics()

	ocResourceMetricsList := make([]consumerdata.MetricsData, 0, len(resourceMetricsList))

	for _, resourceMetrics := range resourceMetricsList {
		ocMetricsData.Node, ocMetricsData.Resource = internalResourceToOC(resourceMetrics.Resource())
		ocMetrics := make([]*ocmetrics.Metric, 0, resourceMetrics.MetricCount())
		for _, instrumentationLibraryMetrics := range resourceMetrics.InstrumentationLibraryMetrics() {
			// TODO: Handle instrumentation library name and version.
			metrics := instrumentationLibraryMetrics.Metrics()
			for _, metric := range metrics {
				ocMetrics = append(ocMetrics, metricToOC(metric))
			}
		}
		ocMetricsData.Metrics = ocMetrics
		ocResourceMetricsList = append(ocResourceMetricsList, ocMetricsData)
	}

	return ocResourceMetricsList
}

func metricToOC(metric data.Metric) *ocmetrics.Metric {
	labelKeys := collectLabelKeys(metric)
	return &ocmetrics.Metric{
		MetricDescriptor: descriptorToOC(metric.MetricDescriptor(), labelKeys),
		Timeseries:       dataPointsToTimeseries(metric, labelKeys),
		Resource:         nil,
	}
}

func collectLabelKeys(metric data.Metric) *labelKeys {
	// NOTE: Intrenal data structure and OpenCensus have different representations of labels:
	// - OC has a single "global" ordered list of label keys per metric in the MetricDescriptor;
	// then, every data point has an ordered list of label values matching the key index.
	// - Internally labels are stored independently as key-value storage for each point.
	//
	// So what we do in this translator:
	// - Scan all points and their labels to find all label keys used across the metric,
	// sort them and set in the MetricDescriptor.
	// - For each point we generate an ordered list of label values,
	// matching the order of label keys returned here (see `labelValuesToOC` function).
	// - If the value for particular label key is missing in the point, we set it to default
	// to preserve 1:1 matching between label keys and values.

	// First, collect a set of all labels present in the metric
	keySet := make(map[string]struct{})
	for _, point := range metric.Int64DataPoints() {
		addLabelKeys(keySet, point.LabelsMap())
	}
	for _, point := range metric.DoubleDataPoints() {
		addLabelKeys(keySet, point.LabelsMap())
	}
	for _, point := range metric.HistogramDataPoints() {
		addLabelKeys(keySet, point.LabelsMap())
	}
	for _, point := range metric.SummaryDataPoints() {
		addLabelKeys(keySet, point.LabelsMap())
	}

	// Sort keys: while not mandatory, this helps to make the
	// output OC metric deterministic and easy to test, i.e.
	// the same set of labels will always produce
	// OC labels in the alphabetically sorted order.
	sortedKeys := make([]string, 0, len(keySet))
	for key := range keySet {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Construct a resulting list of label keys
	keys := make([]*ocmetrics.LabelKey, 0, len(sortedKeys))
	// Label values will have to match keys by index
	// so this map will help with fast lookups.
	indices := make(map[string]int, len(sortedKeys))
	for i, key := range sortedKeys {
		keys = append(keys, &ocmetrics.LabelKey{
			Key: key,
		})
		indices[key] = i
	}

	return &labelKeys{
		keys:       keys,
		keyIndices: indices,
	}
}

func addLabelKeys(keySet map[string]struct{}, labels data.StringMap) {
	for i := 0; i < labels.Len(); i++ {
		keySet[labels.GetStringKeyValue(i).Key()] = struct{}{}
	}
}

func descriptorToOC(descriptor data.MetricDescriptor, labelKeys *labelKeys) *ocmetrics.MetricDescriptor {
	return &ocmetrics.MetricDescriptor{
		Name:        descriptor.Name(),
		Description: descriptor.Description(),
		Unit:        descriptor.Unit(),
		Type:        descriptorTypeToOC(descriptor.Type()),
		LabelKeys:   labelKeys.keys,
	}
}

func descriptorTypeToOC(t data.MetricType) ocmetrics.MetricDescriptor_Type {
	switch t {
	case data.MetricTypeUnspecified:
		return ocmetrics.MetricDescriptor_UNSPECIFIED
	case data.MetricTypeGaugeInt64:
		return ocmetrics.MetricDescriptor_GAUGE_INT64
	case data.MetricTypeGaugeDouble:
		return ocmetrics.MetricDescriptor_GAUGE_DOUBLE
	case data.MetricTypeGaugeHistogram:
		return ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION
	case data.MetricTypeCounterInt64:
		return ocmetrics.MetricDescriptor_CUMULATIVE_INT64
	case data.MetricTypeCounterDouble:
		return ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE
	case data.MetricTypeCumulativeHistogram:
		return ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION
	case data.MetricTypeSummary:
		return ocmetrics.MetricDescriptor_SUMMARY
	default:
		return invalidMetricDescriptorType
	}
}

func dataPointsToTimeseries(metric data.Metric, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	length := len(metric.Int64DataPoints()) + len(metric.DoubleDataPoints()) + len(metric.HistogramDataPoints()) +
		len(metric.SummaryDataPoints())
	if length == 0 {
		return nil
	}

	timeseries := make([]*ocmetrics.TimeSeries, 0, length)
	for _, point := range metric.Int64DataPoints() {
		ts := int64PointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.DoubleDataPoints() {
		ts := doublePointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.HistogramDataPoints() {
		ts := histogramPointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.SummaryDataPoints() {
		ts := summaryPointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}

	return timeseries
}

func int64PointToOC(point data.Int64DataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixnanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixnanoToTimestamp(point.Timestamp()),
				Value: &ocmetrics.Point_Int64Value{
					Int64Value: point.Value(),
				},
			},
		},
	}
}

func doublePointToOC(point data.DoubleDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixnanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixnanoToTimestamp(point.Timestamp()),
				Value: &ocmetrics.Point_DoubleValue{
					DoubleValue: point.Value(),
				},
			},
		},
	}
}

func histogramPointToOC(point data.HistogramDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixnanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixnanoToTimestamp(point.Timestamp()),
				Value: &ocmetrics.Point_DistributionValue{
					DistributionValue: &ocmetrics.DistributionValue{
						Count:                 int64(point.Count()),
						Sum:                   point.Sum(),
						SumOfSquaredDeviation: 0,
						BucketOptions:         histogramExplicitBoundsToOC(point.ExplicitBounds()),
						Buckets:               histogramBucketsToOC(point.Buckets()),
					},
				},
			},
		},
	}
}

func histogramExplicitBoundsToOC(bounds []float64) *ocmetrics.DistributionValue_BucketOptions {
	if len(bounds) == 0 {
		return nil
	}

	return &ocmetrics.DistributionValue_BucketOptions{
		Type: &ocmetrics.DistributionValue_BucketOptions_Explicit_{
			Explicit: &ocmetrics.DistributionValue_BucketOptions_Explicit{
				Bounds: bounds,
			},
		},
	}
}

func histogramBucketsToOC(buckets []data.HistogramBucket) []*ocmetrics.DistributionValue_Bucket {
	if len(buckets) == 0 {
		return nil
	}

	ocBuckets := make([]*ocmetrics.DistributionValue_Bucket, 0, len(buckets))
	for _, bucket := range buckets {
		ocBuckets = append(ocBuckets, &ocmetrics.DistributionValue_Bucket{
			Count:    int64(bucket.Count()),
			Exemplar: exemplarToOC(bucket.Exemplar()),
		})
	}
	return ocBuckets
}

func exemplarToOC(exemplar data.HistogramBucketExemplar) *ocmetrics.DistributionValue_Exemplar {
	attachments := exemplar.Attachments()
	labels := make(map[string]string, attachments.Len())
	for i := 0; i < attachments.Len(); i++ {
		skv := attachments.GetStringKeyValue(i)
		labels[skv.Key()] = skv.Value()
	}
	return &ocmetrics.DistributionValue_Exemplar{
		Value:       exemplar.Value(),
		Timestamp:   internal.UnixnanoToTimestamp(exemplar.Timestamp()),
		Attachments: labels,
	}
}

func summaryPointToOC(point data.SummaryDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixnanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixnanoToTimestamp(point.Timestamp()),
				Value: &ocmetrics.Point_SummaryValue{
					SummaryValue: &ocmetrics.SummaryValue{
						Count: int64Value(point.Count()),
						Sum:   doubleValue(point.Sum()),
						Snapshot: &ocmetrics.SummaryValue_Snapshot{
							Count:            nil,
							Sum:              nil,
							PercentileValues: percentileToOC(point.ValueAtPercentiles()),
						},
					},
				},
			},
		},
	}
}

func percentileToOC(percentiles []data.SummaryValueAtPercentile) []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile {
	if len(percentiles) == 0 {
		return nil
	}

	ocPercentiles := make([]*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, 0, len(percentiles))
	for _, p := range percentiles {
		ocPercentiles = append(ocPercentiles, &ocmetrics.SummaryValue_Snapshot_ValueAtPercentile{
			Percentile: p.Percentile(),
			Value:      p.Value(),
		})
	}
	return ocPercentiles
}

func labelValuesToOC(labels data.StringMap, labelKeys *labelKeys) []*ocmetrics.LabelValue {
	if labels.Len() == 0 {
		return nil
	}

	// Initialize label values with defaults
	// (The order matches key indices)
	labelValues := make([]*ocmetrics.LabelValue, len(labelKeys.keyIndices))
	for i := 0; i < len(labelKeys.keys); i++ {
		labelValues[i] = &ocmetrics.LabelValue{
			HasValue: false,
		}
	}

	// Visit all defined label values and
	// override defaults with actual values
	for i := 0; i < labels.Len(); i++ {
		skv := labels.GetStringKeyValue(i)
		// Find the appropriate label value that we need to update
		keyIndex := labelKeys.keyIndices[skv.Key()]
		labelValue := labelValues[keyIndex]

		// Update label value
		labelValue.Value = skv.Value()
		labelValue.HasValue = true
	}

	return labelValues
}

func int64Value(val uint64) *wrappers.Int64Value {
	return &wrappers.Int64Value{
		Value: int64(val),
	}
}

func doubleValue(val float64) *wrappers.DoubleValue {
	return &wrappers.DoubleValue{
		Value: val,
	}
}
