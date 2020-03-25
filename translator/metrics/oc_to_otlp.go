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
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	translatorcommon "github.com/open-telemetry/opentelemetry-collector/translator/common"
)

const (
	invalidOtlpMetricDescriptorType = otlpmetrics.MetricDescriptor_Type(-1)
)

// OCToOTLP converts metrics from OpenCensus to OTLP format
func OCToOTLP(md consumerdata.MetricsData) []*otlpmetrics.ResourceMetrics {
	if md.Node == nil && md.Resource == nil && len(md.Metrics) == 0 {
		return nil
	}

	resource := translatorcommon.OCNodeResourceToOtlp(md.Node, md.Resource)

	resourceMetrics := &otlpmetrics.ResourceMetrics{
		Resource: resource,
	}
	resourceMetricsList := []*otlpmetrics.ResourceMetrics{resourceMetrics}
	if len(md.Metrics) == 0 {
		return resourceMetricsList
	}

	// TODO: Add InstrumentationLibrary field support
	ilm := &otlpmetrics.InstrumentationLibraryMetrics{}
	resourceMetrics.InstrumentationLibraryMetrics = []*otlpmetrics.InstrumentationLibraryMetrics{ilm}

	ilm.Metrics = make([]*otlpmetrics.Metric, 0, len(md.Metrics))

	for _, ocMetric := range md.Metrics {
		if ocMetric == nil {
			// Skip nil metrics.
			continue
		}

		otlpMetric := metricToOtlp(ocMetric)

		if ocMetric.Resource != nil {
			// Add a separate ResourceMetrics item just for this metric since it
			// has a different Resource.
			separateRM := &otlpmetrics.ResourceMetrics{
				Resource: translatorcommon.OCNodeResourceToOtlp(md.Node, ocMetric.Resource),
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{otlpMetric},
					},
				},
			}
			resourceMetricsList = append(resourceMetricsList, separateRM)
		} else {
			// Otherwise add the metrics to the first ResourceMetrics item.
			ilm.Metrics = append(ilm.Metrics, otlpMetric)
		}
	}
	return resourceMetricsList
}

func metricToOtlp(ocMetric *ocmetrics.Metric) *otlpmetrics.Metric {
	metricDescriptor := descriptorToOtlp(ocMetric.GetMetricDescriptor())
	otlpMetric := &otlpmetrics.Metric{
		MetricDescriptor: metricDescriptor,
	}
	if metricDescriptor == nil {
		return otlpMetric
	}
	labels := getOtlpLabels(ocMetric)
	setDataPoints(otlpMetric, ocMetric, labels)
	return otlpMetric
}

// setDataPoints sets one of oltp metric datapoint slices based on metric type
func setDataPoints(
	otlpMetric *otlpmetrics.Metric,
	ocMetric *ocmetrics.Metric,
	labels [][]*otlpcommon.StringKeyValue,
) {
	switch ocMetric.MetricDescriptor.GetType() {
	case ocmetrics.MetricDescriptor_GAUGE_INT64, ocmetrics.MetricDescriptor_CUMULATIVE_INT64:
		otlpMetric.Int64DataPoints = getInt64DataPoints(ocMetric, labels)
	case ocmetrics.MetricDescriptor_GAUGE_DOUBLE, ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE:
		otlpMetric.DoubleDataPoints = getDoubleDataPoints(ocMetric, labels)
	case ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION, ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		otlpMetric.HistogramDataPoints = getHistogramDataPoints(ocMetric, labels)
	case ocmetrics.MetricDescriptor_SUMMARY:
		otlpMetric.SummaryDataPoints = getSummaryDataPoints(ocMetric, labels)
	default:
	}
}

// getOtlpLabels scans OC metric labels and returns OC timeseries idx -> OTLP labels map.
func getOtlpLabels(ocMetric *ocmetrics.Metric) [][]*otlpcommon.StringKeyValue {
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	labelsCount := len(ocLabelsKeys)
	timeseriesCount := len(ocMetric.GetTimeseries())
	if timeseriesCount == 0 {
		return nil
	}

	labelsByTimeseriesIdx := make([][]*otlpcommon.StringKeyValue, timeseriesCount)

	if labelsCount == 0 {
		return labelsByTimeseriesIdx
	}

	// Scan timeseries and fill the OC timeseries idx -> OTLP labels map
	for i := 0; i < timeseriesCount; i++ {
		ts := ocMetric.GetTimeseries()[i]
		labelsByTimeseriesIdx[i] = make([]*otlpcommon.StringKeyValue, 0, labelsCount)
		for l := 0; l < labelsCount; l++ {
			otlpLabel := ocLabelToOtlp(ocLabelsKeys[l], ts.GetLabelValues()[l])
			if otlpLabel != nil {
				labelsByTimeseriesIdx[i] = append(labelsByTimeseriesIdx[i], otlpLabel)
			}
		}
	}

	return labelsByTimeseriesIdx
}

func ocLabelToOtlp(ocLabelKey *ocmetrics.LabelKey, ocLabelValue *ocmetrics.LabelValue) *otlpcommon.StringKeyValue {
	if !ocLabelValue.GetHasValue() {
		return nil
	}
	return &otlpcommon.StringKeyValue{
		Key:   ocLabelKey.Key,
		Value: ocLabelValue.Value,
	}
}

func getInt64DataPoints(
	ocMetric *ocmetrics.Metric,
	labels [][]*otlpcommon.StringKeyValue,
) []*otlpmetrics.Int64DataPoint {
	int64DataPoints := make([]*otlpmetrics.Int64DataPoint, 0, getPointsCount(ocMetric))
	for i, timeseries := range ocMetric.GetTimeseries() {
		startTimestamp := timeseries.GetStartTimestamp()
		for _, point := range timeseries.GetPoints() {
			int64DataPoints = append(int64DataPoints, &otlpmetrics.Int64DataPoint{
				StartTimeUnixNano: ocTimestampToNanos(startTimestamp),
				TimeUnixNano:      ocTimestampToNanos(point.GetTimestamp()),
				Value:             point.GetInt64Value(),
				Labels:            labels[i],
			})
		}
	}
	return int64DataPoints
}

func getDoubleDataPoints(
	ocMetric *ocmetrics.Metric,
	labels [][]*otlpcommon.StringKeyValue,
) []*otlpmetrics.DoubleDataPoint {
	int64DataPoints := make([]*otlpmetrics.DoubleDataPoint, 0, getPointsCount(ocMetric))
	for i, timeseries := range ocMetric.GetTimeseries() {
		startTimestamp := timeseries.GetStartTimestamp()
		for _, point := range timeseries.GetPoints() {
			int64DataPoints = append(int64DataPoints, &otlpmetrics.DoubleDataPoint{
				StartTimeUnixNano: ocTimestampToNanos(startTimestamp),
				TimeUnixNano:      ocTimestampToNanos(point.GetTimestamp()),
				Value:             point.GetDoubleValue(),
				Labels:            labels[i],
			})
		}
	}
	return int64DataPoints
}

func getHistogramDataPoints(
	ocMetric *ocmetrics.Metric,
	labels [][]*otlpcommon.StringKeyValue,
) []*otlpmetrics.HistogramDataPoint {
	histogramDataPoints := make([]*otlpmetrics.HistogramDataPoint, 0, getPointsCount(ocMetric))
	for i, timeseries := range ocMetric.GetTimeseries() {
		startTimestamp := timeseries.GetStartTimestamp()
		for _, point := range timeseries.GetPoints() {
			distributionValue := point.GetDistributionValue()
			if distributionValue == nil {
				continue
			}
			histogramDataPoints = append(histogramDataPoints, &otlpmetrics.HistogramDataPoint{
				StartTimeUnixNano: ocTimestampToNanos(startTimestamp),
				TimeUnixNano:      ocTimestampToNanos(point.GetTimestamp()),
				Count:             uint64(distributionValue.GetCount()),
				Sum:               distributionValue.GetSum(),
				Buckets:           histogramBucketsToOtlp(distributionValue.Buckets),
				ExplicitBounds:    distributionValue.GetBucketOptions().GetExplicit().GetBounds(),
				Labels:            labels[i],
			})
		}
	}
	return histogramDataPoints
}

func getSummaryDataPoints(
	ocMetric *ocmetrics.Metric,
	labels [][]*otlpcommon.StringKeyValue,
) []*otlpmetrics.SummaryDataPoint {
	summaryDataPoints := make([]*otlpmetrics.SummaryDataPoint, 0, getPointsCount(ocMetric))
	for i, timeseries := range ocMetric.GetTimeseries() {
		startTimestamp := timeseries.GetStartTimestamp()
		for _, point := range timeseries.GetPoints() {
			ocSummaryValue := point.GetSummaryValue()
			if ocSummaryValue == nil {
				continue
			}
			summaryDataPoints = append(summaryDataPoints, &otlpmetrics.SummaryDataPoint{
				StartTimeUnixNano: ocTimestampToNanos(startTimestamp),
				TimeUnixNano:      ocTimestampToNanos(point.GetTimestamp()),
				Count:             uint64(ocSummaryValue.GetCount().GetValue()),
				Sum:               ocSummaryValue.GetSum().GetValue(),
				PercentileValues:  percentileToOtlp(ocSummaryValue.GetSnapshot().GetPercentileValues()),
				Labels:            labels[i],
			})
		}
	}
	return summaryDataPoints
}

func percentileToOtlp(
	ocPercentiles []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile,
) []*otlpmetrics.SummaryDataPoint_ValueAtPercentile {
	if ocPercentiles == nil {
		return nil
	}
	otlpPercentiles := make([]*otlpmetrics.SummaryDataPoint_ValueAtPercentile, 0, len(ocPercentiles))
	for _, p := range ocPercentiles {
		otlpPercentiles = append(otlpPercentiles, &otlpmetrics.SummaryDataPoint_ValueAtPercentile{
			Percentile: p.Percentile,
			Value:      p.Value,
		})
	}
	return otlpPercentiles
}

func histogramBucketsToOtlp(ocBuckets []*ocmetrics.DistributionValue_Bucket) []*otlpmetrics.HistogramDataPoint_Bucket {
	if ocBuckets == nil {
		return nil
	}
	otlpBuckets := make([]*otlpmetrics.HistogramDataPoint_Bucket, 0, len(ocBuckets))
	for _, bucket := range ocBuckets {
		otlpBuckets = append(otlpBuckets, &otlpmetrics.HistogramDataPoint_Bucket{
			Count:    uint64(bucket.Count),
			Exemplar: exemplarToOtlp(bucket.Exemplar),
		})
	}
	return otlpBuckets
}

func exemplarToOtlp(ocExemplar *ocmetrics.DistributionValue_Exemplar) *otlpmetrics.HistogramDataPoint_Bucket_Exemplar {
	if ocExemplar == nil {
		return nil
	}

	return &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
		Value:        ocExemplar.Value,
		TimeUnixNano: ocTimestampToNanos(ocExemplar.Timestamp),
		Attachments:  exemplarAttachmentsToOtlp(ocExemplar.Attachments),
	}
}

func exemplarAttachmentsToOtlp(ocAttachments map[string]string) []*otlpcommon.StringKeyValue {
	if len(ocAttachments) == 0 {
		return nil
	}

	otlpAttachments := make([]*otlpcommon.StringKeyValue, 0, len(ocAttachments))
	for key, value := range ocAttachments {
		otlpAttachments = append(otlpAttachments, &otlpcommon.StringKeyValue{
			Key:   key,
			Value: value,
		})
	}
	return otlpAttachments
}

func ocTimestampToNanos(ts *timestamp.Timestamp) uint64 {
	return uint64(internal.TimestampToUnixNano(ts))
}

func getPointsCount(ocMetric *ocmetrics.Metric) int {
	timeseriesSlice := ocMetric.GetTimeseries()
	var count int
	for _, timeseries := range timeseriesSlice {
		points := timeseries.GetPoints()
		count += len(points)
	}
	return count
}

func descriptorToOtlp(descriptor *ocmetrics.MetricDescriptor) *otlpmetrics.MetricDescriptor {
	descriptorType := descriptorTypeToOtlp(descriptor.Type)
	if descriptor == nil || descriptorType == invalidOtlpMetricDescriptorType {
		return nil
	}

	return &otlpmetrics.MetricDescriptor{
		Name:        descriptor.Name,
		Description: descriptor.Description,
		Unit:        descriptor.Unit,
		Type:        descriptorType,
	}
}

func descriptorTypeToOtlp(t ocmetrics.MetricDescriptor_Type) otlpmetrics.MetricDescriptor_Type {
	switch t {
	case ocmetrics.MetricDescriptor_UNSPECIFIED:
		return otlpmetrics.MetricDescriptor_UNSPECIFIED
	case ocmetrics.MetricDescriptor_GAUGE_INT64:
		return otlpmetrics.MetricDescriptor_GAUGE_INT64
	case ocmetrics.MetricDescriptor_GAUGE_DOUBLE:
		return otlpmetrics.MetricDescriptor_GAUGE_DOUBLE
	case ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION:
		return otlpmetrics.MetricDescriptor_GAUGE_HISTOGRAM
	case ocmetrics.MetricDescriptor_CUMULATIVE_INT64:
		return otlpmetrics.MetricDescriptor_COUNTER_INT64
	case ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE:
		return otlpmetrics.MetricDescriptor_COUNTER_DOUBLE
	case ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		return otlpmetrics.MetricDescriptor_CUMULATIVE_HISTOGRAM
	case ocmetrics.MetricDescriptor_SUMMARY:
		return otlpmetrics.MetricDescriptor_SUMMARY
	default:
		return invalidOtlpMetricDescriptorType
	}
}
