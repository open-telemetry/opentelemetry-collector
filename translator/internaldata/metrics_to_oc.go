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

package internaldata

import (
	"sort"

	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/wrappers"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type labelKeys struct {
	// ordered OC label keys
	keys []*ocmetrics.LabelKey
	// map from a label key literal
	// to its index in the slice above
	keyIndices map[string]int
}

// MetricsToOC may be used only by OpenCensus receiver and exporter implementations.
// TODO: move this function to OpenCensus package.
func MetricsToOC(md pdata.Metrics) []consumerdata.MetricsData {
	resourceMetrics := md.ResourceMetrics()

	if resourceMetrics.Len() == 0 {
		return nil
	}

	ocResourceMetricsList := make([]consumerdata.MetricsData, 0, resourceMetrics.Len())
	for i := 0; i < resourceMetrics.Len(); i++ {
		ocResourceMetricsList = append(ocResourceMetricsList, resourceMetricsToOC(resourceMetrics.At(i)))
	}

	return ocResourceMetricsList
}

func resourceMetricsToOC(rm pdata.ResourceMetrics) consumerdata.MetricsData {
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
		// TODO: Handle instrumentation library name and version.
		metrics := ilm.Metrics()
		for j := 0; j < metrics.Len(); j++ {
			ocMetrics = append(ocMetrics, metricToOC(metrics.At(j)))
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
		MetricDescriptor: descriptorToOC(metric, labelKeys),
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

	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		collectLabelKeysIntDataPoints(metric.IntGauge().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleGauge:
		collectLabelKeysDoubleDataPoints(metric.DoubleGauge().DataPoints(), keySet)
	case pdata.MetricDataTypeIntSum:
		collectLabelKeysIntDataPoints(metric.IntSum().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleSum:
		collectLabelKeysDoubleDataPoints(metric.DoubleSum().DataPoints(), keySet)
	case pdata.MetricDataTypeIntHistogram:
		collectLabelKeysIntHistogramDataPoints(metric.IntHistogram().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleHistogram:
		collectLabelKeysDoubleHistogramDataPoints(metric.DoubleHistogram().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleSummary:
		collectLabelKeysDoubleSummaryDataPoints(metric.DoubleSummary().DataPoints(), keySet)
	}

	if len(keySet) == 0 {
		return &labelKeys{}
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

func collectLabelKeysIntDataPoints(ips pdata.IntDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < ips.Len(); i++ {
		addLabelKeys(keySet, ips.At(i).LabelsMap())
	}
}

func collectLabelKeysDoubleDataPoints(dps pdata.DoubleDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dps.Len(); i++ {
		addLabelKeys(keySet, dps.At(i).LabelsMap())
	}
}

func collectLabelKeysIntHistogramDataPoints(ihdp pdata.IntHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < ihdp.Len(); i++ {
		addLabelKeys(keySet, ihdp.At(i).LabelsMap())
	}
}

func collectLabelKeysDoubleHistogramDataPoints(dhdp pdata.DoubleHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		addLabelKeys(keySet, dhdp.At(i).LabelsMap())
	}
}

func collectLabelKeysDoubleSummaryDataPoints(dhdp pdata.DoubleSummaryDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		addLabelKeys(keySet, dhdp.At(i).LabelsMap())
	}
}

func addLabelKeys(keySet map[string]struct{}, labels pdata.StringMap) {
	labels.ForEach(func(k string, v string) {
		keySet[k] = struct{}{}
	})
}

func descriptorToOC(metric pdata.Metric, labelKeys *labelKeys) *ocmetrics.MetricDescriptor {
	return &ocmetrics.MetricDescriptor{
		Name:        metric.Name(),
		Description: metric.Description(),
		Unit:        metric.Unit(),
		Type:        descriptorTypeToOC(metric),
		LabelKeys:   labelKeys.keys,
	}
}

func descriptorTypeToOC(metric pdata.Metric) ocmetrics.MetricDescriptor_Type {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		return ocmetrics.MetricDescriptor_GAUGE_INT64
	case pdata.MetricDataTypeDoubleGauge:
		return ocmetrics.MetricDescriptor_GAUGE_DOUBLE
	case pdata.MetricDataTypeIntSum:
		sd := metric.IntSum()
		if sd.IsMonotonic() || sd.AggregationTemporality() == pdata.AggregationTemporalityCumulative {
			return ocmetrics.MetricDescriptor_CUMULATIVE_INT64
		}
		return ocmetrics.MetricDescriptor_GAUGE_INT64
	case pdata.MetricDataTypeDoubleSum:
		sd := metric.DoubleSum()
		if sd.IsMonotonic() || sd.AggregationTemporality() == pdata.AggregationTemporalityCumulative {
			return ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE
		}
		return ocmetrics.MetricDescriptor_GAUGE_DOUBLE
	case pdata.MetricDataTypeDoubleHistogram:
		hd := metric.DoubleHistogram()
		if hd.AggregationTemporality() == pdata.AggregationTemporalityCumulative {
			return ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION
		}
		return ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION
	case pdata.MetricDataTypeIntHistogram:
		hd := metric.IntHistogram()
		if hd.AggregationTemporality() == pdata.AggregationTemporalityCumulative {
			return ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION
		}
		return ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION
	case pdata.MetricDataTypeDoubleSummary:
		return ocmetrics.MetricDescriptor_SUMMARY
	}
	return ocmetrics.MetricDescriptor_UNSPECIFIED
}

func dataPointsToTimeseries(metric pdata.Metric, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		return intPointsToOC(metric.IntGauge().DataPoints(), labelKeys)
	case pdata.MetricDataTypeDoubleGauge:
		return doublePointToOC(metric.DoubleGauge().DataPoints(), labelKeys)
	case pdata.MetricDataTypeIntSum:
		return intPointsToOC(metric.IntSum().DataPoints(), labelKeys)
	case pdata.MetricDataTypeDoubleSum:
		return doublePointToOC(metric.DoubleSum().DataPoints(), labelKeys)
	case pdata.MetricDataTypeIntHistogram:
		return intHistogramPointToOC(metric.IntHistogram().DataPoints(), labelKeys)
	case pdata.MetricDataTypeDoubleHistogram:
		return doubleHistogramPointToOC(metric.DoubleHistogram().DataPoints(), labelKeys)
	case pdata.MetricDataTypeDoubleSummary:
		return doubleSummaryPointToOC(metric.DoubleSummary().DataPoints(), labelKeys)
	}

	return nil
}

func intPointsToOC(dps pdata.IntDataPointSlice, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)
		ts := &ocmetrics.TimeSeries{
			StartTimestamp: pdata.UnixNanoToTimestamp(ip.StartTime()),
			LabelValues:    labelValuesToOC(ip.LabelsMap(), labelKeys),
			Points: []*ocmetrics.Point{
				{
					Timestamp: pdata.UnixNanoToTimestamp(ip.Timestamp()),
					Value: &ocmetrics.Point_Int64Value{
						Int64Value: ip.Value(),
					},
				},
			},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
}

func doublePointToOC(dps pdata.DoubleDataPointSlice, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		ts := &ocmetrics.TimeSeries{
			StartTimestamp: pdata.UnixNanoToTimestamp(dp.StartTime()),
			LabelValues:    labelValuesToOC(dp.LabelsMap(), labelKeys),
			Points: []*ocmetrics.Point{
				{
					Timestamp: pdata.UnixNanoToTimestamp(dp.Timestamp()),
					Value: &ocmetrics.Point_DoubleValue{
						DoubleValue: dp.Value(),
					},
				},
			},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
}

func doubleHistogramPointToOC(dps pdata.DoubleHistogramDataPointSlice, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		buckets := histogramBucketsToOC(dp.BucketCounts())
		doubleExemplarsToOC(dp.ExplicitBounds(), buckets, dp.Exemplars())

		ts := &ocmetrics.TimeSeries{
			StartTimestamp: pdata.UnixNanoToTimestamp(dp.StartTime()),
			LabelValues:    labelValuesToOC(dp.LabelsMap(), labelKeys),
			Points: []*ocmetrics.Point{
				{
					Timestamp: pdata.UnixNanoToTimestamp(dp.Timestamp()),
					Value: &ocmetrics.Point_DistributionValue{
						DistributionValue: &ocmetrics.DistributionValue{
							Count:                 int64(dp.Count()),
							Sum:                   dp.Sum(),
							SumOfSquaredDeviation: 0,
							BucketOptions:         histogramExplicitBoundsToOC(dp.ExplicitBounds()),
							Buckets:               buckets,
						},
					},
				},
			},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
}

func intHistogramPointToOC(dps pdata.IntHistogramDataPointSlice, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		buckets := histogramBucketsToOC(dp.BucketCounts())
		intExemplarsToOC(dp.ExplicitBounds(), buckets, dp.Exemplars())

		ts := &ocmetrics.TimeSeries{
			StartTimestamp: pdata.UnixNanoToTimestamp(dp.StartTime()),
			LabelValues:    labelValuesToOC(dp.LabelsMap(), labelKeys),
			Points: []*ocmetrics.Point{
				{
					Timestamp: pdata.UnixNanoToTimestamp(dp.Timestamp()),
					Value: &ocmetrics.Point_DistributionValue{
						DistributionValue: &ocmetrics.DistributionValue{
							Count:                 int64(dp.Count()),
							Sum:                   float64(dp.Sum()),
							SumOfSquaredDeviation: 0,
							BucketOptions:         histogramExplicitBoundsToOC(dp.ExplicitBounds()),
							Buckets:               buckets,
						},
					},
				},
			},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
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

func histogramBucketsToOC(bcts []uint64) []*ocmetrics.DistributionValue_Bucket {
	if len(bcts) == 0 {
		return nil
	}

	ocBuckets := make([]*ocmetrics.DistributionValue_Bucket, 0, len(bcts))
	for _, bucket := range bcts {
		ocBuckets = append(ocBuckets, &ocmetrics.DistributionValue_Bucket{
			Count: int64(bucket),
		})
	}
	return ocBuckets
}

func doubleSummaryPointToOC(dps pdata.DoubleSummaryDataPointSlice, labelKeys *labelKeys) []*ocmetrics.TimeSeries {
	if dps.Len() == 0 {
		return nil
	}
	timeseries := make([]*ocmetrics.TimeSeries, 0, dps.Len())
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		percentileValues := summaryPercentilesToOC(dp.QuantileValues())

		ts := &ocmetrics.TimeSeries{
			StartTimestamp: pdata.UnixNanoToTimestamp(dp.StartTime()),
			LabelValues:    labelValuesToOC(dp.LabelsMap(), labelKeys),
			Points: []*ocmetrics.Point{
				{
					Timestamp: pdata.UnixNanoToTimestamp(dp.Timestamp()),
					Value: &ocmetrics.Point_SummaryValue{
						SummaryValue: &ocmetrics.SummaryValue{
							Sum:   &wrappers.DoubleValue{Value: dp.Sum()},
							Count: &wrappers.Int64Value{Value: int64(dp.Count())},
							Snapshot: &ocmetrics.SummaryValue_Snapshot{
								PercentileValues: percentileValues,
							},
						},
					},
				},
			},
		}
		timeseries = append(timeseries, ts)
	}
	return timeseries
}

func summaryPercentilesToOC(qtls pdata.ValueAtQuantileSlice) []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile {
	if qtls.Len() == 0 {
		return nil
	}

	ocPercentiles := make([]*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, 0, qtls.Len())
	for i := 0; i < qtls.Len(); i++ {
		quantile := qtls.At(i)
		ocPercentiles = append(ocPercentiles, &ocmetrics.SummaryValue_Snapshot_ValueAtPercentile{
			Percentile: quantile.Quantile() * 100,
			Value:      quantile.Value(),
		})
	}
	return ocPercentiles
}

func doubleExemplarsToOC(bounds []float64, ocBuckets []*ocmetrics.DistributionValue_Bucket, exemplars pdata.DoubleExemplarSlice) {
	if exemplars.Len() == 0 {
		return
	}

	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		val := exemplar.Value()
		pos := 0
		for ; pos < len(bounds); pos++ {
			if val > bounds[pos] {
				continue
			}
			break
		}
		ocBuckets[pos].Exemplar = exemplarToOC(exemplar.FilteredLabels(), val, exemplar.Timestamp())
	}
}

func intExemplarsToOC(bounds []float64, ocBuckets []*ocmetrics.DistributionValue_Bucket, exemplars pdata.IntExemplarSlice) {
	if exemplars.Len() == 0 {
		return
	}

	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		val := float64(exemplar.Value())
		pos := 0
		for ; pos < len(bounds); pos++ {
			if val > bounds[pos] {
				continue
			}
			break
		}
		ocBuckets[pos].Exemplar = exemplarToOC(exemplar.FilteredLabels(), val, exemplar.Timestamp())
	}
}

func exemplarToOC(filteredLabels pdata.StringMap, value float64, timestamp pdata.TimestampUnixNano) *ocmetrics.DistributionValue_Exemplar {
	var labels map[string]string
	if filteredLabels.Len() != 0 {
		labels = make(map[string]string, filteredLabels.Len())
		filteredLabels.ForEach(func(k string, v string) {
			labels[k] = v
		})
	}

	return &ocmetrics.DistributionValue_Exemplar{
		Value:       value,
		Timestamp:   pdata.UnixNanoToTimestamp(timestamp),
		Attachments: labels,
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
	labels.ForEach(func(k string, v string) {
		// Find the appropriate label value that we need to update
		keyIndex := labelKeys.keyIndices[k]
		labelValue := labelValues[keyIndex]

		// Update label value
		labelValue.Value = v
		labelValue.HasValue = true
	})

	return labelValues
}
