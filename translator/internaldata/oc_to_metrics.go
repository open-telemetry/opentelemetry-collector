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

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// OCSliceToMetricData converts a slice of OC data format to data.MetricData.
// Deprecated: use pdata.Metrics instead.
func OCSliceToMetrics(ocmds []consumerdata.MetricsData) pdata.Metrics {
	metricData := pdata.NewMetrics()
	if len(ocmds) == 0 {
		return metricData
	}
	for _, ocmd := range ocmds {
		appendOcToMetrics(ocmd, metricData)
	}
	return metricData
}

// OCToMetricData converts OC data format to data.MetricData.
// Deprecated: use pdata.Metrics instead. OCToMetrics may be used only by OpenCensus
// receiver and exporter implementations.
func OCToMetrics(md consumerdata.MetricsData) pdata.Metrics {
	metricData := pdata.NewMetrics()
	appendOcToMetrics(md, metricData)
	return metricData
}

func appendOcToMetrics(md consumerdata.MetricsData, dest pdata.Metrics) {
	if md.Node == nil && md.Resource == nil && len(md.Metrics) == 0 {
		return
	}

	rms := dest.ResourceMetrics()
	initialRmsLen := rms.Len()

	if len(md.Metrics) == 0 {
		// At least one of the md.Node or md.Resource is not nil. Set the resource and return.
		rms.Resize(initialRmsLen + 1)
		ocNodeResourceToInternal(md.Node, md.Resource, rms.At(initialRmsLen).Resource())
		return
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
	for _, ocMetric := range md.Metrics {
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
	rms.Resize(resourceCount)

	// Translate "combinedMetrics" first

	if combinedMetricCount > 0 {
		rm0 := rms.At(initialRmsLen)
		ocNodeResourceToInternal(md.Node, md.Resource, rm0.Resource())

		// Allocate a slice for metrics that need to be combined into first ResourceMetrics.
		ilms := rm0.InstrumentationLibraryMetrics()
		ilms.Resize(1)
		combinedMetrics := ilms.At(0).Metrics()
		combinedMetrics.Resize(combinedMetricCount)

		// Index to next available slot in "combinedMetrics" slice.
		combinedMetricIdx := 0

		for _, ocMetric := range md.Metrics {
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
			ocMetricToMetrics(ocMetric, combinedMetrics.At(combinedMetricIdx))
			combinedMetricIdx++
		}
	}

	// Translate distinct metrics

	resourceMetricIdx := 0
	if combinedMetricCount > 0 {
		// First resourcemetric is used for the default resource, so start with 1.
		resourceMetricIdx = 1
	}
	for _, ocMetric := range md.Metrics {
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
		ocMetricToResourceMetrics(ocMetric, md.Node, rms.At(initialRmsLen+resourceMetricIdx))
		resourceMetricIdx++
	}
}

func ocMetricToResourceMetrics(ocMetric *ocmetrics.Metric, node *occommon.Node, out pdata.ResourceMetrics) {
	ocNodeResourceToInternal(node, ocMetric.Resource, out.Resource())
	ilms := out.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	metrics := ilms.At(0).Metrics()
	metrics.Resize(1)
	ocMetricToMetrics(ocMetric, metrics.At(0))
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
		metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
		return pdata.MetricDataTypeDoubleGauge
	case ocmetrics.MetricDescriptor_CUMULATIVE_INT64:
		metric.SetDataType(pdata.MetricDataTypeIntSum)
		sum := metric.IntSum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		return pdata.MetricDataTypeIntSum
	case ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE:
		metric.SetDataType(pdata.MetricDataTypeDoubleSum)
		sum := metric.DoubleSum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		return pdata.MetricDataTypeDoubleSum
	case ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		metric.SetDataType(pdata.MetricDataTypeDoubleHistogram)
		histo := metric.DoubleHistogram()
		histo.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		return pdata.MetricDataTypeDoubleHistogram
	case ocmetrics.MetricDescriptor_SUMMARY:
		metric.SetDataType(pdata.MetricDataTypeDoubleSummary)
		// no temporality specified for summary metric
		return pdata.MetricDataTypeDoubleSummary
	}
	return pdata.MetricDataTypeNone
}

// setDataPoints converts OC timeseries to internal datapoints based on metric type
func setDataPoints(ocMetric *ocmetrics.Metric, metric pdata.Metric) {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		fillIntDataPoint(ocMetric, metric.IntGauge().DataPoints())
	case pdata.MetricDataTypeDoubleGauge:
		fillDoubleDataPoint(ocMetric, metric.DoubleGauge().DataPoints())
	case pdata.MetricDataTypeIntSum:
		fillIntDataPoint(ocMetric, metric.IntSum().DataPoints())
	case pdata.MetricDataTypeDoubleSum:
		fillDoubleDataPoint(ocMetric, metric.DoubleSum().DataPoints())
	case pdata.MetricDataTypeDoubleHistogram:
		fillDoubleHistogramDataPoint(ocMetric, metric.DoubleHistogram().DataPoints())
	case pdata.MetricDataTypeDoubleSummary:
		fillDoubleSummaryDataPoint(ocMetric, metric.DoubleSummary().DataPoints())
	}
}

func setLabelsMap(ocLabelsKeys []*ocmetrics.LabelKey, ocLabelValues []*ocmetrics.LabelValue, labelsMap pdata.StringMap) {
	if len(ocLabelsKeys) == 0 || len(ocLabelValues) == 0 {
		return
	}

	lablesCount := len(ocLabelsKeys)

	// Handle invalid length of OC label values list
	if len(ocLabelValues) < lablesCount {
		lablesCount = len(ocLabelValues)
	}

	labelsMap.InitEmptyWithCapacity(lablesCount)
	for i := 0; i < lablesCount; i++ {
		if !ocLabelValues[i].GetHasValue() {
			continue
		}
		labelsMap.Insert(ocLabelsKeys[i].Key, ocLabelValues[i].Value)
	}
}

func fillIntDataPoint(ocMetric *ocmetrics.Metric, dps pdata.IntDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.Resize(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	pos := 0
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampToUnixNano(timeseries.GetStartTimestamp())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.At(pos)
			pos++

			dp.SetStartTime(startTimestamp)
			dp.SetTimestamp(pdata.TimestampToUnixNano(point.GetTimestamp()))
			setLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			dp.SetValue(point.GetInt64Value())
		}
	}
	dps.Resize(pos)
}

func fillDoubleDataPoint(ocMetric *ocmetrics.Metric, dps pdata.DoubleDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.Resize(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	pos := 0
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampToUnixNano(timeseries.GetStartTimestamp())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.At(pos)
			pos++

			dp.SetStartTime(startTimestamp)
			dp.SetTimestamp(pdata.TimestampToUnixNano(point.GetTimestamp()))
			setLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			dp.SetValue(point.GetDoubleValue())
		}
	}
	dps.Resize(pos)
}

func fillDoubleHistogramDataPoint(ocMetric *ocmetrics.Metric, dps pdata.DoubleHistogramDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.Resize(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	pos := 0
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampToUnixNano(timeseries.GetStartTimestamp())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.At(pos)
			pos++

			dp.SetStartTime(startTimestamp)
			dp.SetTimestamp(pdata.TimestampToUnixNano(point.GetTimestamp()))
			setLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			distributionValue := point.GetDistributionValue()
			dp.SetSum(distributionValue.GetSum())
			dp.SetCount(uint64(distributionValue.GetCount()))
			ocHistogramBucketsToMetrics(distributionValue.GetBuckets(), dp)
			dp.SetExplicitBounds(distributionValue.GetBucketOptions().GetExplicit().GetBounds())
		}
	}
	dps.Resize(pos)
}

func fillDoubleSummaryDataPoint(ocMetric *ocmetrics.Metric, dps pdata.DoubleSummaryDataPointSlice) {
	ocPointsCount := getPointsCount(ocMetric)
	dps.Resize(ocPointsCount)
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	pos := 0
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := pdata.TimestampToUnixNano(timeseries.GetStartTimestamp())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}

			dp := dps.At(pos)
			pos++

			dp.SetStartTime(startTimestamp)
			dp.SetTimestamp(pdata.TimestampToUnixNano(point.GetTimestamp()))
			setLabelsMap(ocLabelsKeys, timeseries.LabelValues, dp.LabelsMap())
			summaryValue := point.GetSummaryValue()
			dp.SetSum(summaryValue.GetSum().GetValue())
			dp.SetCount(uint64(summaryValue.GetCount().GetValue()))
			ocSummaryPercentilesToMetrics(summaryValue.GetSnapshot().GetPercentileValues(), dp)
		}
	}
	dps.Resize(pos)
}

func ocHistogramBucketsToMetrics(ocBuckets []*ocmetrics.DistributionValue_Bucket, dp pdata.DoubleHistogramDataPoint) {
	if len(ocBuckets) == 0 {
		return
	}
	buckets := make([]uint64, len(ocBuckets))
	for i := range buckets {
		buckets[i] = uint64(ocBuckets[i].GetCount())
		if ocBuckets[i].GetExemplar() != nil {
			exemplar := pdata.NewDoubleExemplar()
			exemplarToMetrics(ocBuckets[i].GetExemplar(), exemplar)
			dp.Exemplars().Append(exemplar)
		}
	}
	dp.SetBucketCounts(buckets)
}

func ocSummaryPercentilesToMetrics(ocPercentiles []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile, dp pdata.DoubleSummaryDataPoint) {
	if len(ocPercentiles) == 0 {
		return
	}

	quantiles := pdata.NewValueAtQuantileSlice()
	quantiles.Resize(len(ocPercentiles))

	for i, percentile := range ocPercentiles {
		quantiles.At(i).SetQuantile(percentile.GetPercentile() / 100)
		quantiles.At(i).SetValue(percentile.GetValue())
	}

	quantiles.CopyTo(dp.QuantileValues())
}

func exemplarToMetrics(ocExemplar *ocmetrics.DistributionValue_Exemplar, exemplar pdata.DoubleExemplar) {
	if ocExemplar.GetTimestamp() != nil {
		exemplar.SetTimestamp(pdata.TimestampToUnixNano(ocExemplar.GetTimestamp()))
	}
	exemplar.SetValue(ocExemplar.GetValue())
	attachments := exemplar.FilteredLabels()
	ocAttachments := ocExemplar.GetAttachments()
	attachments.InitEmptyWithCapacity(len(ocAttachments))
	for k, v := range ocAttachments {
		attachments.Upsert(k, v)
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
