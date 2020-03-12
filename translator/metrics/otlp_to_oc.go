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

// ResourceMetricsToMetricsData converts metrics from OTLP to internal (OpenCensus) format
func ResourceMetricsToMetricsData(resourceMetrics *otlpmetrics.ResourceMetrics) consumerdata.MetricsData {

	node, resource := translatorcommon.ResourceToOC(resourceMetrics.Resource)
	md := consumerdata.MetricsData{
		Node:     node,
		Resource: resource,
	}

	if len(resourceMetrics.Metrics) == 0 {
		return md
	}

	metrics := make([]*ocmetrics.Metric, 0, len(resourceMetrics.Metrics))
	for _, metric := range resourceMetrics.Metrics {
		metrics = append(metrics, metricToOC(metric))
	}

	md.Metrics = metrics
	return md
}

func metricToOC(metric *otlpmetrics.Metric) *ocmetrics.Metric {
	labelKeys := allLabelKeys(metric)
	return &ocmetrics.Metric{
		MetricDescriptor: descriptorToOC(metric.MetricDescriptor, labelKeys),
		Timeseries:       dataPointsToTimeseries(metric, labelKeys),
		Resource:         nil,
	}
}

func descriptorToOC(descriptor *otlpmetrics.MetricDescriptor, labelKeys []*ocmetrics.LabelKey) *ocmetrics.MetricDescriptor {
	if descriptor == nil {
		return nil
	}

	return &ocmetrics.MetricDescriptor{
		Name:        descriptor.Name,
		Description: descriptor.Description,
		Unit:        descriptor.Unit,
		Type:        descriptorTypeToOC(descriptor.Type),
		LabelKeys:   labelKeys,
	}
}

func allLabelKeys(metric *otlpmetrics.Metric) []*ocmetrics.LabelKey {
	// NOTE: OpenTelemetry and OpenCensus have different representations of labels:
	// - OC has a single "global" ordered list of label keys per metric in the MetricDescriptor;
	// then, every data point has an ordered list of label values matching the key index.
	// - In OTLP, every label is represented independently via StringKeyValue,
	// i.e. theoretically points in the same metric may have different set of labels.
	//
	// So what we do in this translator:
	// - Scan all points and their labels to generate a unique set of all label keys
	// used across the metric, sort them and set in MetricDescriptor.
	// - For each point we generate an ordered list of label values,
	// matching the order of label keys returned here (see `labelsToOC` function).

	// First, collect a set of unique keys
	uniqueKeys := make(map[string]struct{}, 0)
	for _, point := range metric.Int64Datapoints {
		addLabelKeys(uniqueKeys, point.Labels)
	}
	for _, point := range metric.DoubleDatapoints {
		addLabelKeys(uniqueKeys, point.Labels)
	}
	for _, point := range metric.HistogramDatapoints {
		addLabelKeys(uniqueKeys, point.Labels)
	}
	for _, point := range metric.SummaryDatapoints {
		addLabelKeys(uniqueKeys, point.Labels)
	}

	// Sort keys (this is actually optional, just for stable order)
	rawKeys := make([]string, 0, len(uniqueKeys))
	for key := range uniqueKeys {
		rawKeys = append(rawKeys, key)
	}
	sort.Strings(rawKeys)

	// Construct a list of label keys
	// Note: Label values will have to match keys by index
	labelKeys := make([]*ocmetrics.LabelKey, 0, len(rawKeys))
	for _, key := range rawKeys {
		labelKeys = append(labelKeys, &ocmetrics.LabelKey{
			Key: key,
		})
	}

	return labelKeys
}

func addLabelKeys(uniqueKeys map[string]struct{}, labels []*otlpcommon.StringKeyValue) {
	for _, label := range labels {
		if _, ok := uniqueKeys[label.Key]; !ok {
			uniqueKeys[label.Key] = struct{}{}
		}
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

func dataPointsToTimeseries(metric *otlpmetrics.Metric, labelKeys []*ocmetrics.LabelKey) []*ocmetrics.TimeSeries {
	length := len(metric.Int64Datapoints) + len(metric.DoubleDatapoints) + len(metric.HistogramDatapoints) +
		len(metric.SummaryDatapoints)
	if length == 0 {
		return nil
	}

	timeseries := make([]*ocmetrics.TimeSeries, 0, length)
	for _, point := range metric.Int64Datapoints {
		ts := int64PointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.DoubleDatapoints {
		ts := doublePointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.HistogramDatapoints {
		ts := histogramPointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}
	for _, point := range metric.SummaryDatapoints {
		ts := summaryPointToOC(point, labelKeys)
		timeseries = append(timeseries, ts)
	}

	return timeseries
}

func summaryPointToOC(point *otlpmetrics.SummaryDataPoint, labelKeys []*ocmetrics.LabelKey) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelsToOC(point.Labels, labelKeys),
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

func int64PointToOC(point *otlpmetrics.Int64DataPoint, labelKeys []*ocmetrics.LabelKey) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelsToOC(point.Labels, labelKeys),
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

func doublePointToOC(point *otlpmetrics.DoubleDataPoint, labelKeys []*ocmetrics.LabelKey) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelsToOC(point.Labels, labelKeys),
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

func histogramPointToOC(point *otlpmetrics.HistogramDataPoint, labelKeys []*ocmetrics.LabelKey) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: unixnanoToTimestamp(point.StartTimeUnixnano),
		LabelValues:    labelsToOC(point.Labels, labelKeys),
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

func labelsToOC(labels []*otlpcommon.StringKeyValue, labelKeys []*ocmetrics.LabelKey) []*ocmetrics.LabelValue {
	if len(labels) == 0 {
		return nil
	}

	// NOTE: We need to set label values in the same order as keys, thus
	// intermediate transformation to a map for fast lookups
	labelMap := make(map[string]string, len(labels))
	for _, label := range labels {
		labelMap[label.Key] = label.Value
	}

	labelValues := make([]*ocmetrics.LabelValue, len(labelKeys))
	// Visit all label keys in order, and set the value for each of them
	for i, key := range labelKeys {
		var labelValue *ocmetrics.LabelValue
		if val, ok := labelMap[key.Key]; ok {
			labelValue = &ocmetrics.LabelValue{
				Value:    val,
				HasValue: true,
			}
		} else {
			// Even if label value is missing, we need to set "empty" value
			// to preserve the index
			labelValue = &ocmetrics.LabelValue{
				HasValue: false,
			}
		}
		labelValues[i] = labelValue
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
