// Copyright The OpenTelemetry Authors
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
	"google.golang.org/protobuf/types/known/wrapperspb"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	"go.opentelemetry.io/collector/internal/data"
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

func MetricDataToOC(md data.MetricData) []consumerdata.MetricsData {
	resourceMetrics := md.ResourceMetrics()

	if resourceMetrics.Len() == 0 {
		return nil
	}

	ocResourceMetricsList := make([]consumerdata.MetricsData, 0, resourceMetrics.Len())
	for i := 0; i < resourceMetrics.Len(); i++ {
		rs := resourceMetrics.At(i)
		if rs.IsNil() {
			continue
		}
		ocResourceMetricsList = append(ocResourceMetricsList, ResourceMetricsToOC(rs))
	}

	return ocResourceMetricsList
}

func ResourceMetricsToOC(rm pdata.ResourceMetrics) consumerdata.MetricsData {
	ocMetricsData := consumerdata.MetricsData{}
	ocMetricsData.Node, ocMetricsData.Resource = internalResourceToOC(rm.Resource())
	ilms := rm.InstrumentationLibraryMetrics()
	if ilms.Len() == 0 {
		return ocMetricsData
	}
	// Approximate the number of the metrics as the number of the metrics in the first
	// instrumentation library info.
	ocMetrics := make([]*ocmetrics.Metric, 0, ilms.At(0).Metrics().Len())
	for i := 0; i < ilms.Len(); i++ {
		ilm := ilms.At(i)
		if ilm.IsNil() {
			continue
		}
		// TODO: Handle instrumentation library name and version.
		metrics := ilm.Metrics()
		for j := 0; j < metrics.Len(); j++ {
			m := metrics.At(j)
			if m.IsNil() {
				continue
			}
			ocMetrics = append(ocMetrics, metricToOC(m))
		}
	}
	if len(ocMetrics) != 0 {
		ocMetricsData.Metrics = ocMetrics
	}
	return ocMetricsData
}

func metricToOC(metric pdata.Metric) *ocmetrics.Metric {
	labelKeys := collectLabelKeys(metric)
	return &ocmetrics.Metric{
		MetricDescriptor: descriptorToOC(metric.MetricDescriptor(), labelKeys),
		Timeseries:       dataPointsToTimeseries(metric, labelKeys),
		Resource:         nil,
	}
}

func collectLabelKeys(metric pdata.Metric) *labelKeys {
	// NOTE: Internal data structure and OpenCensus have different representations of labels:
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
	ips := metric.Int64DataPoints()
	for i := 0; i < ips.Len(); i++ {
		ip := ips.At(i)
		if ip.IsNil() {
			continue
		}
		addLabelKeys(keySet, ip.LabelsMap())
	}
	dps := metric.DoubleDataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if dp.IsNil() {
			continue
		}
		addLabelKeys(keySet, dp.LabelsMap())
	}
	hps := metric.HistogramDataPoints()
	for i := 0; i < hps.Len(); i++ {
		hp := hps.At(i)
		if hp.IsNil() {
			continue
		}
		addLabelKeys(keySet, hp.LabelsMap())
	}
	sps := metric.SummaryDataPoints()
	for i := 0; i < sps.Len(); i++ {
		sp := sps.At(i)
		if sp.IsNil() {
			continue
		}
		addLabelKeys(keySet, sp.LabelsMap())
	}

	if len(keySet) == 0 {
		return &labelKeys{
			keys:       nil,
			keyIndices: nil,
		}
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

func addLabelKeys(keySet map[string]struct{}, labels pdata.StringMap) {
	labels.ForEach(func(k string, v pdata.StringValue) {
		keySet[k] = struct{}{}
	})
}

func descriptorToOC(descriptor pdata.MetricDescriptor, labelKeys *labelKeys) *ocmetrics.MetricDescriptor {
	if descriptor.IsNil() {
		return nil
	}
	return &ocmetrics.MetricDescriptor{
		Name:        descriptor.Name(),
		Description: descriptor.Description(),
		Unit:        descriptor.Unit(),
		Type:        descriptorTypeToOC(descriptor.Type()),
		LabelKeys:   labelKeys.keys,
	}
}

func descriptorTypeToOC(t pdata.MetricType) ocmetrics.MetricDescriptor_Type {
	switch t {
	case pdata.MetricTypeInvalid:
		return ocmetrics.MetricDescriptor_UNSPECIFIED
	case pdata.MetricTypeInt64:
		return ocmetrics.MetricDescriptor_GAUGE_INT64
	case pdata.MetricTypeDouble:
		return ocmetrics.MetricDescriptor_GAUGE_DOUBLE
	case pdata.MetricTypeMonotonicInt64:
		return ocmetrics.MetricDescriptor_CUMULATIVE_INT64
	case pdata.MetricTypeMonotonicDouble:
		return ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE
	case pdata.MetricTypeHistogram:
		return ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION
	case pdata.MetricTypeSummary:
		return ocmetrics.MetricDescriptor_SUMMARY
	default:
		return invalidMetricDescriptorType
	}
}

func dataPointsToTimeseries(metric pdata.Metric, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	length := metric.Int64DataPoints().Len() + metric.DoubleDataPoints().Len() +
		metric.HistogramDataPoints().Len() + metric.SummaryDataPoints().Len()
	if length == 0 {
		return nil
	}

	timeseries := make([]*ocmetrics.TimeSeries, 0, length)
	ips := metric.Int64DataPoints()
	for i := 0; i < ips.Len(); i++ {
		ip := ips.At(i)
		if ip.IsNil() {
			continue
		}
		ts := int64PointToOC(ip, labelKeys)
		timeseries = append(timeseries, ts)
	}
	dps := metric.DoubleDataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if dp.IsNil() {
			continue
		}
		ts := doublePointToOC(dp, labelKeys)
		timeseries = append(timeseries, ts)
	}
	hps := metric.HistogramDataPoints()
	for i := 0; i < hps.Len(); i++ {
		hp := hps.At(i)
		if hp.IsNil() {
			continue
		}
		ts := histogramPointToOC(hp, labelKeys)
		timeseries = append(timeseries, ts)
	}
	sps := metric.SummaryDataPoints()
	for i := 0; i < sps.Len(); i++ {
		sp := sps.At(i)
		if sp.IsNil() {
			continue
		}
		ts := summaryPointToOC(sp, labelKeys)
		timeseries = append(timeseries, ts)
	}

	return timeseries
}

func int64PointToOC(point pdata.Int64DataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixNanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixNanoToTimestamp(point.Timestamp()),
				Value: &ocmetrics.Point_Int64Value{
					Int64Value: point.Value(),
				},
			},
		},
	}
}

func doublePointToOC(point pdata.DoubleDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixNanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixNanoToTimestamp(point.Timestamp()),
				Value: &ocmetrics.Point_DoubleValue{
					DoubleValue: point.Value(),
				},
			},
		},
	}
}

func histogramPointToOC(point pdata.HistogramDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixNanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixNanoToTimestamp(point.Timestamp()),
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

func histogramBucketsToOC(buckets pdata.HistogramBucketSlice) []*ocmetrics.DistributionValue_Bucket {
	if buckets.Len() == 0 {
		return nil
	}

	ocBuckets := make([]*ocmetrics.DistributionValue_Bucket, 0, buckets.Len())
	for i := 0; i < buckets.Len(); i++ {
		bucket := buckets.At(i)
		ocBuckets = append(ocBuckets, &ocmetrics.DistributionValue_Bucket{
			Count:    int64(bucket.Count()),
			Exemplar: exemplarToOC(bucket.Exemplar()),
		})
	}
	return ocBuckets
}

func exemplarToOC(exemplar pdata.HistogramBucketExemplar) *ocmetrics.DistributionValue_Exemplar {
	if exemplar.IsNil() {
		return nil
	}
	attachments := exemplar.Attachments()
	if attachments.Len() == 0 {
		return &ocmetrics.DistributionValue_Exemplar{
			Value:       exemplar.Value(),
			Timestamp:   internal.UnixNanoToTimestamp(exemplar.Timestamp()),
			Attachments: nil,
		}
	}

	labels := make(map[string]string, attachments.Len())
	attachments.ForEach(func(k string, v pdata.StringValue) {
		labels[k] = v.Value()
	})
	return &ocmetrics.DistributionValue_Exemplar{
		Value:       exemplar.Value(),
		Timestamp:   internal.UnixNanoToTimestamp(exemplar.Timestamp()),
		Attachments: labels,
	}
}

func summaryPointToOC(point pdata.SummaryDataPoint, labelKeys *labelKeys) *ocmetrics.TimeSeries {
	return &ocmetrics.TimeSeries{
		StartTimestamp: internal.UnixNanoToTimestamp(point.StartTime()),
		LabelValues:    labelValuesToOC(point.LabelsMap(), labelKeys),
		Points: []*ocmetrics.Point{
			{
				Timestamp: internal.UnixNanoToTimestamp(point.Timestamp()),
				Value: &ocmetrics.Point_SummaryValue{
					SummaryValue: &ocmetrics.SummaryValue{
						Count:    int64Value(point.Count()),
						Sum:      doubleValue(point.Sum()),
						Snapshot: percentileToOC(point.ValueAtPercentiles()),
					},
				},
			},
		},
	}
}

func percentileToOC(percentiles pdata.SummaryValueAtPercentileSlice) *ocmetrics.SummaryValue_Snapshot {
	if percentiles.Len() == 0 {
		return nil
	}

	ocPercentiles := make([]*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, 0, percentiles.Len())
	for i := 0; i < percentiles.Len(); i++ {
		p := percentiles.At(i)
		ocPercentiles = append(ocPercentiles, &ocmetrics.SummaryValue_Snapshot_ValueAtPercentile{
			Percentile: p.Percentile(),
			Value:      p.Value(),
		})
	}
	return &ocmetrics.SummaryValue_Snapshot{
		Count:            nil,
		Sum:              nil,
		PercentileValues: ocPercentiles,
	}
}

func labelValuesToOC(labels pdata.StringMap, labelKeys *labelKeys) []*ocmetrics.LabelValue {
	if len(labelKeys.keys) == 0 {
		return nil
	}

	// Initialize label values with defaults
	// (The order matches key indices)
	labelValuesOrig := make([]ocmetrics.LabelValue, len(labelKeys.keys))
	labelValues := make([]*ocmetrics.LabelValue, len(labelKeys.keys))
	for i := 0; i < len(labelKeys.keys); i++ {
		labelValues[i] = &labelValuesOrig[i]
	}

	// Visit all defined labels in the point and override defaults with actual values
	labels.ForEach(func(k string, v pdata.StringValue) {
		// Find the appropriate label value that we need to update
		keyIndex := labelKeys.keyIndices[k]
		labelValue := labelValues[keyIndex]

		// Update label value
		labelValue.Value = v.Value()
		labelValue.HasValue = true
	})

	return labelValues
}

func int64Value(val uint64) *wrapperspb.Int64Value {
	return &wrapperspb.Int64Value{
		Value: int64(val),
	}
}

func doubleValue(val float64) *wrapperspb.DoubleValue {
	return &wrapperspb.DoubleValue{
		Value: val,
	}
}
