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
	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"

	"go.opentelemetry.io/collector/model/pdata"
)

// OCToMetrics converts OC data format to data.MetricData.
// Deprecated: use pdata.Metrics instead. OCToMetrics may be used only by OpenCensus
// receiver and exporter implementations.
func OCToMetrics(node *occommon.Node, resource *ocresource.Resource, metrics []*ocmetrics.Metric) pdata.Metrics {
	dest := pdata.NewMetrics()
	if node == nil && resource == nil && len(metrics) == 0 {
		return dest
	}

	rms := dest.ResourceMetrics()
	initialRmsLen := rms.Len()

	if len(metrics) == 0 {
		// At least one of the md.Node or md.Resource is not nil. Set the resource and return.
		ocNodeResourceToInternal(node, resource, rms.AppendEmpty().Resource())
		return dest
	}

	// We may need to split OC metrics into several ResourceMetrics. OC metrics can have a
	// Resource field inside them set to nil which indicates they use the Resource
	// specified in "md.Resource", or they can have the Resource field inside them set
	// to non-nil which indicates they have overridden Resource field and "md.Resource"
	// does not apply to those metrics.
	//
	// Each OC metric that has its own Resource field set to non-nil must be placed in a
	// separate ResourceMetrics instance, containing only that metric. All other OC Metrics
	// that have nil Resource field must be placed in one other ResourceMetrics instance,
	// which will gets its Resource field from "md.Resource".
	//
	// We will end up with with one or more ResourceMetrics like this:
	//
	// ResourceMetrics           ResourceMetrics  ResourceMetrics
	// +-------+-------+---+-------+ +--------------+ +--------------+
	// |Metric1|Metric2|...|MetricM| |Metric        | |Metric        | ...
	// +-------+-------+---+-------+ +--------------+ +--------------+

	// Count the number of metrics that have nil Resource and need to be combined
	// in one slice.
	combinedMetricCount := 0
	distinctResourceCount := 0
	for _, ocMetric := range metrics {
		if ocMetric == nil {
			// Skip nil metrics.
			continue
		}
		if ocMetric.Resource == nil {
			combinedMetricCount++
		} else {
			distinctResourceCount++
		}
	}
	// Total number of resources is equal to:
	// initial + numMetricsWithResource + (optional) 1
	resourceCount := initialRmsLen + distinctResourceCount
	if combinedMetricCount > 0 {
		// +1 for all metrics with nil resource
		resourceCount++
	}
	rms.EnsureCapacity(resourceCount)

	// Translate "combinedMetrics" first

	if combinedMetricCount > 0 {
		rm0 := rms.AppendEmpty()
		ocNodeResourceToInternal(node, resource, rm0.Resource())

		// Allocate a slice for metrics that need to be combined into first ResourceMetrics.
		ilms := rm0.InstrumentationLibraryMetrics()
		combinedMetrics := ilms.AppendEmpty().Metrics()
		combinedMetrics.EnsureCapacity(combinedMetricCount)

		for _, ocMetric := range metrics {
			if ocMetric == nil {
				// Skip nil metrics.
				continue
			}

			if ocMetric.Resource != nil {
				continue // Those are processed separately below.
			}

			// Add the metric to the "combinedMetrics". combinedMetrics length is equal
			// to combinedMetricCount. The loop above that calculates combinedMetricCount
			// has exact same conditions as we have here in this loop.
			ocMetricToMetrics(ocMetric, combinedMetrics.AppendEmpty())
		}
	}

	// Translate distinct metrics

	for _, ocMetric := range metrics {
		if ocMetric == nil {
			// Skip nil metrics.
			continue
		}

		if ocMetric.Resource == nil {
			continue // Already processed above.
		}

		// This metric has a different Resource and must be placed in a different
		// ResourceMetrics instance. Create a separate ResourceMetrics item just for this metric
		// and store at resourceMetricIdx.
		ocMetricToResourceMetrics(ocMetric, node, rms.AppendEmpty())
	}
	return dest
}

func ocMetricToResourceMetrics(ocMetric *ocmetrics.Metric, node *occommon.Node, out pdata.ResourceMetrics) {
	ocNodeResourceToInternal(node, ocMetric.Resource, out.Resource())
	ilms := out.InstrumentationLibraryMetrics()
	ocMetricToMetrics(ocMetric, ilms.AppendEmpty().Metrics().AppendEmpty())
}

func ocMetricToMetrics(ocMetric *ocmetrics.Metric, metric pdata.Metric) {
	ocDescriptor := ocMetric.GetMetricDescriptor()
	if ocDescriptor == nil {
		pdata.NewMetric().CopyTo(metric)
		return
	}

	descriptorType := descriptorTypeToMetrics(ocDescriptor.Type, metric)
	if descriptorType == pdata.MetricDataTypeNone {
		pdata.NewMetric().CopyTo(metric)
		return
	}

	metric.SetDescription(ocDescriptor.GetDescription())
	metric.SetName(ocDescriptor.GetName())
	metric.SetUnit(ocDescriptor.GetUnit())

	setDataPoints(ocMetric, metric)
}

func descriptorTypeToMetrics(t ocmetrics.MetricDescriptor_Type, metric pdata.Metric) pdata.MetricDataType {
	switch t {
	case ocmetrics.MetricDescriptor_GAUGE_INT64:
		metric.SetDataType(pdata.MetricDataTypeIntGauge)
		return pdata.MetricDataTypeIntGauge
	case ocmetrics.MetricDescriptor_GAUGE_DOUBLE:
		metric.SetDataType(pdata.MetricDataTypeGauge)
		return pdata.MetricDataTypeGauge
	case ocmetrics.MetricDescriptor_CUMULATIVE_INT64:
		metric.SetDataType(pdata.MetricDataTypeIntSum)
		sum := metric.IntSum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		return pdata.MetricDataTypeIntSum
	case ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE:
		metric.SetDataType(pdata.MetricDataTypeSum)
		sum := metric.Sum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		return pdata.MetricDataTypeSum
	case ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		metric.SetDataType(pdata.MetricDataTypeHistogram)
		histo := metric.Histogram()
		histo.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		return pdata.MetricDataTypeHistogram
	case ocmetrics.MetricDescriptor_SUMMARY:
		metric.SetDataType(pdata.MetricDataTypeSummary)
		// no temporality specified for summary metric
		return pdata.MetricDataTypeSummary
	}
	return pdata.MetricDataTypeNone
}

// setDataPoints converts OC timeseries to internal datapoints based on metric type
func setDataPoints(ocMetric *ocmetrics.Metric, metric pdata.Metric) {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		fillIntDataPoint(ocMetric, metric.IntGauge().DataPoints())
	case pdata.MetricDataTypeGauge:
		fillDoubleDataPoint(ocMetric, metric.Gauge().DataPoints())
	case pdata.MetricDataTypeIntSum:
		fillIntDataPoint(ocMetric, metric.IntSum().DataPoints())
	case pdata.MetricDataTypeSum:
		fillDoubleDataPoint(ocMetric, metric.Sum().DataPoints())
	case pdata.MetricDataTypeHistogram:
		fillDoubleHistogramDataPoint(ocMetric, metric.Histogram().DataPoints())
	case pdata.MetricDataTypeSummary:
		fillDoubleSummaryDataPoint(ocMetric, metric.Summary().DataPoints())
	}
}

func fillLabelsMap(ocLabelsKeys []*ocmetrics.LabelKey, ocLabelValues []*ocmetrics.LabelValue, labelsMap pdata.StringMap) {
	if len(ocLabelsKeys) == 0 || len(ocLabelValues) == 0 {
		return
	}

	lablesCount := len(ocLabelsKeys)

	// Handle invalid length of OC label values list
	if len(ocLabelValues) < lablesCount {
		lablesCount = len(ocLabelValues)
	}

	labelsMap.Clear()
	labelsMap.EnsureCapacity(lablesCount)
	for i := 0; i < lablesCount; i++ {
		if !ocLabelValues[i].GetHasValue() {
			continue
		}
		labelsMap.Insert(ocLabelsKeys[i].Key, ocLabelValues[i].Value)
	}
}

func fillIntDataPoint(ocMetric *ocmetrics.Metric, dps pdata.IntDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.EnsureCapacity(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampFromTime(timeseries.GetStartTimestamp().AsTime())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.AppendEmpty()
			dp.SetStartTimestamp(startTimestamp)
			dp.SetTimestamp(pdata.TimestampFromTime(point.GetTimestamp().AsTime()))
			fillLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			dp.SetValue(point.GetInt64Value())
		}
	}
}

func fillDoubleDataPoint(ocMetric *ocmetrics.Metric, dps pdata.DoubleDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.EnsureCapacity(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampFromTime(timeseries.GetStartTimestamp().AsTime())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.AppendEmpty()
			dp.SetStartTimestamp(startTimestamp)
			dp.SetTimestamp(pdata.TimestampFromTime(point.GetTimestamp().AsTime()))
			fillLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			dp.SetValue(point.GetDoubleValue())
		}
	}
}

func fillDoubleHistogramDataPoint(ocMetric *ocmetrics.Metric, dps pdata.HistogramDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.EnsureCapacity(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampFromTime(timeseries.GetStartTimestamp().AsTime())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.AppendEmpty()
			dp.SetStartTimestamp(startTimestamp)
			dp.SetTimestamp(pdata.TimestampFromTime(point.GetTimestamp().AsTime()))
			fillLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			distributionValue := point.GetDistributionValue()
			dp.SetSum(distributionValue.GetSum())
			dp.SetCount(uint64(distributionValue.GetCount()))
			ocHistogramBucketsToMetrics(distributionValue.GetBuckets(), dp)
			dp.SetExplicitBounds(distributionValue.GetBucketOptions().GetExplicit().GetBounds())
		}
	}
}

func fillDoubleSummaryDataPoint(ocMetric *ocmetrics.Metric, dps pdata.SummaryDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.EnsureCapacity(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampFromTime(timeseries.GetStartTimestamp().AsTime())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.AppendEmpty()
			dp.SetStartTimestamp(startTimestamp)
			dp.SetTimestamp(pdata.TimestampFromTime(point.GetTimestamp().AsTime()))
			fillLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			summaryValue := point.GetSummaryValue()
			dp.SetSum(summaryValue.GetSum().GetValue())
			dp.SetCount(uint64(summaryValue.GetCount().GetValue()))
			ocSummaryPercentilesToMetrics(summaryValue.GetSnapshot().GetPercentileValues(), dp)
		}
	}
}

func ocHistogramBucketsToMetrics(ocBuckets []*ocmetrics.DistributionValue_Bucket, dp pdata.HistogramDataPoint) {
	if len(ocBuckets) == 0 {
		return
	}
	buckets := make([]uint64, len(ocBuckets))
	for i := range buckets {
		buckets[i] = uint64(ocBuckets[i].GetCount())
		if ocBuckets[i].GetExemplar() != nil {
			exemplar := dp.Exemplars().AppendEmpty()
			exemplarToMetrics(ocBuckets[i].GetExemplar(), exemplar)
		}
	}
	dp.SetBucketCounts(buckets)
}

func ocSummaryPercentilesToMetrics(ocPercentiles []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, dp pdata.SummaryDataPoint) {
	if len(ocPercentiles) == 0 {
		return
	}

	quantiles := pdata.NewValueAtQuantileSlice()
	quantiles.EnsureCapacity(len(ocPercentiles))

	for _, percentile := range ocPercentiles {
		quantile := quantiles.AppendEmpty()
		quantile.SetQuantile(percentile.GetPercentile() / 100)
		quantile.SetValue(percentile.GetValue())
	}

	quantiles.CopyTo(dp.QuantileValues())
}

func exemplarToMetrics(ocExemplar *ocmetrics.DistributionValue_Exemplar, exemplar pdata.Exemplar) {
	if ocExemplar.GetTimestamp() != nil {
		exemplar.SetTimestamp(pdata.TimestampFromTime(ocExemplar.GetTimestamp().AsTime()))
	}
	ocAttachments := ocExemplar.GetAttachments()
	exemplar.SetValue(ocExemplar.GetValue())
	filteredLabels := exemplar.FilteredLabels()
	filteredLabels.Clear()
	filteredLabels.EnsureCapacity(len(ocAttachments))
	for k, v := range ocAttachments {
		filteredLabels.Upsert(k, v)
	}
}

func getPointsCount(ocMetric *ocmetrics.Metric) int {
	timeseriesSlice := ocMetric.GetTimeseries()
	var count int
	for _, timeseries := range timeseriesSlice {
		count += len(timeseries.GetPoints())
	}
	return count
}
