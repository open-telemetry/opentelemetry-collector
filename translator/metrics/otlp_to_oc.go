// Copyright 2020 OpenTelemetry Authors
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

package metrics

import (
	"sort"

	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	translatorcommon "github.com/open-telemetry/opentelemetry-collector/translator/common"
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

// ResourceMetricsToMetricsData converts metrics from OTLP to internal (OpenCensus) format
func ResourceMetricsToMetricsData(resourceMetrics *otlpmetrics.ResourceMetrics) consumerdata.MetricsData {
	node, resource := translatorcommon.ResourceToOC(resourceMetrics.Resource)
	md := consumerdata.MetricsData{
		Node:     node,
		Resource: resource,
	}

	if len(resourceMetrics.InstrumentationLibraryMetrics) == 0 {
		return md
	}

	// Allocate slice with a capacity approximated for the case when there is only one
	// InstrumentationLibrary or the first InstrumentationLibrary contains most of the data.
	// This is a best guess only that reduced slice re-allocations.
	metrics := make([]*ocmetrics.Metric, 0, len(resourceMetrics.InstrumentationLibraryMetrics[0].Metrics))
	for _, il := range resourceMetrics.InstrumentationLibraryMetrics {
		for _, metric := range il.Metrics {
			metrics = append(metrics, metricToOC(metric))
		}
	}

	md.Metrics = metrics
	return md
}

func metricToOC(metric *otlpmetrics.Metric) *ocmetrics.Metric {
	labelKeys := labelKeysToOC(metric)
	return &ocmetrics.Metric{
		MetricDescriptor: descriptorToOC(metric.MetricDescriptor, labelKeys),
		Timeseries:       dataPointsToTimeseries(metric, labelKeys),
		Resource:         nil,
	}
}

func descriptorToOC(descriptor *otlpmetrics.MetricDescriptor, labelKeys *labelKeys) *ocmetrics.MetricDescriptor {
	if descriptor == nil {
		return nil
	}

	return &ocmetrics.MetricDescriptor{
		Name:        descriptor.Name,
		Description: descriptor.Description,
		Unit:        descriptor.Unit,
		Type:        descriptorTypeToOC(descriptor.Type),
		LabelKeys:   labelKeys.keys,
	}
}

func labelKeysToOC(metric *otlpmetrics.Metric) *labelKeys {
	// NOTE: OpenTelemetry and OpenCensus have different representations of labels:
	// - OC has a single "global" ordered list of label keys per metric in the MetricDescriptor;
	// then, every data point has an ordered list of label values matching the key index.
	// - In OTLP, every label is represented independently via StringKeyValue,
	// i.e. theoretically points in the same metric may have different set of labels.
	//
	// So what we do in this translator:
	// - Scan all points and their labels to find all label keys used across the metric,
	// sort them and set in the MetricDescriptor.
	// - For each point we generate an ordered list of label values,
	// matching the order of label keys returned here (see `labelValuesToOC` function).
	// - If the value for particular label key is missing in the point, we set it to default
	// to preserve 1:1 matching between label keys and values.

	// TODO: Support common labels from otlpmetrics.MetricDescriptor.Labels

	// First, collect a set of all labels present in the metric
	keySet := make(map[string]struct{}, 0)
	for _, point := range metric.Int64DataPoints {
		addLabelKeys(keySet, point.Labels)
	}
	for _, point := range metric.DoubleDataPoints {
		addLabelKeys(keySet, point.Labels)
	}
	for _, point := range metric.HistogramDataPoints {
		addLabelKeys(keySet, point.Labels)
	}
	for _, point := range metric.SummaryDataPoints {
		addLabelKeys(keySet, point.Labels)
	}

	// Sort keys: while not mandatory, this helps to make the
	// output OC metric deterministic and easy to test, i.e.
	// the same set of OTLP labels will always produce
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

func addLabelKeys(keySet map[string]struct{}, labels []*otlpcommon.StringKeyValue) {
	for _, label := range labels {
		keySet[label.Key] = struct{}{}
	}
}

func descriptorTypeToOC(t otlpmetrics.MetricDescriptor_Type) ocmetrics.MetricDescriptor_Type {
	switch t {
	case otlpmetrics.MetricDescriptor_UNSPECIFIED:
		return ocmetrics.MetricDescriptor_UNSPECIFIED
	case otlpmetrics.MetricDescriptor_GAUGE_INT64:
		return ocmetrics.MetricDescriptor_GAUGE_INT64
	case otlpmetrics.MetricDescriptor_GAUGE_DOUBLE:
		return ocmetrics.MetricDescriptor_GAUGE_DOUBLE
	case otlpmetrics.MetricDescriptor_GAUGE_HISTOGRAM:
		return ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION
	case otlpmetrics.MetricDescriptor_COUNTER_INT64:
		return ocmetrics.MetricDescriptor_CUMULATIVE_INT64
	case otlpmetrics.MetricDescriptor_COUNTER_DOUBLE:
		return ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE
	case otlpmetrics.MetricDescriptor_CUMULATIVE_HISTOGRAM:
		return ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION
	case otlpmetrics.MetricDescriptor_SUMMARY:
		return ocmetrics.MetricDescriptor_SUMMARY
	default:
		return invalidMetricDescriptorType
	}
}

func dataPointsToTimeseries(metric *otlpmetrics.Metric, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	length := len(metric.Int64DataPoints) + len(metric.DoubleDataPoints) + len(metric.HistogramDataPoints) +
		len(metric.SummaryDataPoints)
	if length == 0 {
		return nil
	}

	timeseries := make([]*ocmetrics.TimeSeries, 0, length)
	for _, point := range metric.Int64DataPoints {
		ts := int64PointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.DoubleDataPoints {
		ts := doublePointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.HistogramDataPoints {
		ts := histogramPointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.SummaryDataPoints {
		ts := summaryPointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}

	return timeseries
}

func summaryPointToOC(point *otlpmetrics.SummaryDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelValuesToOC(point.Labels, labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: unixnanoToTimestamp(point.TimestampUnixnano),
				Value: &ocmetrics.Point_SummaryValue{
					SummaryValue: &ocmetrics.SummaryValue{
						Count: int64Value(point.Count),
						Sum:   doubleValue(point.Sum),
						Snapshot: &ocmetrics.SummaryValue_Snapshot{
							Count:            nil,
							Sum:              nil,
							PercentileValues: percentileToOC(point.PercentileValues),
						},
					},
				},
			},
		},
	}
}

func percentileToOC(percentiles []*otlpmetrics.SummaryDataPoint_ValueAtPercentile) []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile {
	if len(percentiles) == 0 {
		return nil
	}

	ocPercentiles := make([]*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, 0, len(percentiles))
	for _, p := range percentiles {
		ocPercentiles = append(ocPercentiles, &ocmetrics.SummaryValue_Snapshot_ValueAtPercentile{
			Percentile: p.Percentile,
			Value:      p.Value,
		})
	}
	return ocPercentiles
}

func int64PointToOC(point *otlpmetrics.Int64DataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelValuesToOC(point.Labels, labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: unixnanoToTimestamp(point.TimestampUnixnano),
				Value: &ocmetrics.Point_Int64Value{
					Int64Value: point.Value,
				},
			},
		},
	}
}

func doublePointToOC(point *otlpmetrics.DoubleDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelValuesToOC(point.Labels, labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: unixnanoToTimestamp(point.TimestampUnixnano),
				Value: &ocmetrics.Point_DoubleValue{
					DoubleValue: point.Value,
				},
			},
		},
	}
}

func histogramPointToOC(point *otlpmetrics.HistogramDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelValuesToOC(point.Labels, labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: unixnanoToTimestamp(point.TimestampUnixnano),
				Value: &ocmetrics.Point_DistributionValue{
					DistributionValue: &ocmetrics.DistributionValue{
						Count:                 int64(point.Count),
						Sum:                   point.Sum,
						SumOfSquaredDeviation: 0,
						BucketOptions:         histogramExplicitBoundsToOC(point.ExplicitBounds),
						Buckets:               histogramBucketsToOC(point.Buckets),
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

func histogramBucketsToOC(buckets []*otlpmetrics.HistogramDataPoint_Bucket) []*ocmetrics.DistributionValue_Bucket {
	if len(buckets) == 0 {
		return nil
	}

	ocBuckets := make([]*ocmetrics.DistributionValue_Bucket, 0, len(buckets))
	for _, bucket := range buckets {
		ocBuckets = append(ocBuckets, &ocmetrics.DistributionValue_Bucket{
			Count:    int64(bucket.Count),
			Exemplar: exemplarToOC(bucket.Exemplar),
		})
	}
	return ocBuckets
}

func exemplarToOC(exemplar *otlpmetrics.HistogramDataPoint_Bucket_Exemplar) *ocmetrics.DistributionValue_Exemplar {
	if exemplar == nil {
		return nil
	}

	return &ocmetrics.DistributionValue_Exemplar{
		Value:       exemplar.Value,
		Timestamp:   unixnanoToTimestamp(exemplar.TimestampUnixnano),
		Attachments: exemplarAttachmentsToOC(exemplar.Attachments),
	}
}

func exemplarAttachmentsToOC(attachments []*otlpcommon.StringKeyValue) map[string]string {
	if len(attachments) == 0 {
		return nil
	}

	ocAttachments := make(map[string]string, len(attachments))
	for _, att := range attachments {
		ocAttachments[att.Key] = att.Value
	}
	return ocAttachments
}

func labelValuesToOC(labels []*otlpcommon.StringKeyValue, labelKeys *labelKeys) []*ocmetrics.LabelValue {
	if len(labels) == 0 {
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
	for _, label := range labels {
		// Find the appropriate label value that we need to update
		keyIndex := labelKeys.keyIndices[label.Key]
		labelValue := labelValues[keyIndex]

		// Update label value
		labelValue.Value = label.Value
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

func unixnanoToTimestamp(u uint64) *timestamp.Timestamp {
	return internal.UnixnanoToTimestamp(data.TimestampUnixNano(u))
}
