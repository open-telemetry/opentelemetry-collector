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

package internaldata

import (
	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

const (
	invalidMetricType = data.MetricType(-1)
)

// OCToMetricData converts OC data format to MetricData.
func OCToMetricData(md consumerdata.MetricsData) data.MetricData {
	metricData := data.NewMetricData()
	if md.Node == nil && md.Resource == nil && len(md.Metrics) == 0 {
		return metricData
	}

	if len(md.Metrics) == 0 {
		// At least one of the md.Node or md.Resource is not nil. Set the resource and return.
		rms := metricData.ResourceMetrics()
		rms.Resize(1)
		ocNodeResourceToInternal(md.Node, md.Resource, rms.At(0).Resource())
		return metricData
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
	// 1 (for all metrics with nil resource) + numMetricsWithResource (distinctResourceCount).
	rms := metricData.ResourceMetrics()
	rms.Resize(distinctResourceCount + 1)
	rm0 := rms.At(0)
	ocNodeResourceToInternal(md.Node, md.Resource, rm0.Resource())

	// Allocate a slice for metrics that need to be combined into first ResourceMetrics.
	ilms := rm0.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	combinedMetrics := ilms.At(0).Metrics()
	combinedMetrics.Resize(combinedMetricCount)

	// Now do the metric translation and place them in appropriate ResourceMetrics
	// instances.

	// Index to next available slot in "combinedMetrics" slice.
	combinedMetricIdx := 0
	// First resourcemetric is used for the default resource, so start with 1.
	resourceMetricIdx := 1
	for _, ocMetric := range md.Metrics {
		if ocMetric == nil {
			// Skip nil metrics.
			continue
		}

		if ocMetric.Resource == nil {
			// Add the metric to the "combinedMetrics". combinedMetrics length is equal
			// to combinedMetricCount. The loop above that calculates combinedMetricCount
			// has exact same conditions as we have here in this loop.
			ocMetricToInternal(ocMetric, combinedMetrics.At(combinedMetricIdx))
			combinedMetricIdx++
		} else {
			// This metric has a different Resource and must be placed in a different
			// ResourceMetrics instance. Create a separate ResourceMetrics item just for this metric
			// and store at resourceMetricIdx.
			ocMetricToResourceMetrics(ocMetric, md.Node, rms.At(resourceMetricIdx))
			resourceMetricIdx++
		}
	}

	return metricData
}

func ocMetricToResourceMetrics(ocMetric *ocmetrics.Metric, node *occommon.Node, out data.ResourceMetrics) {
	ocNodeResourceToInternal(node, ocMetric.Resource, out.Resource())
	ilms := out.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	metrics := ilms.At(0).Metrics()
	metrics.Resize(1)
	ocMetricToInternal(ocMetric, metrics.At(0))
}

// ocMetricToInternal conversts ocMetric to internal representation and fill metric
func ocMetricToInternal(ocMetric *ocmetrics.Metric, metric data.Metric) {
	descriptorToInternal(ocMetric.GetMetricDescriptor(), metric.MetricDescriptor())
	setDataPoints(ocMetric, metric)
}

func descriptorToInternal(ocDescriptor *ocmetrics.MetricDescriptor, descriptor data.MetricDescriptor) {
	if ocDescriptor == nil {
		return
	}

	descriptorType := descriptorTypeToInternal(ocDescriptor.Type)
	if descriptorType == invalidMetricType {
		return
	}

	descriptor.InitEmpty()
	descriptor.SetType(descriptorType)
	descriptor.SetDescription(ocDescriptor.GetDescription())
	descriptor.SetName(ocDescriptor.GetName())
	descriptor.SetUnit(ocDescriptor.GetUnit())
}

func descriptorTypeToInternal(t ocmetrics.MetricDescriptor_Type) data.MetricType {
	switch t {
	case ocmetrics.MetricDescriptor_UNSPECIFIED:
		return data.MetricTypeUnspecified
	case ocmetrics.MetricDescriptor_GAUGE_INT64:
		return data.MetricTypeGaugeInt64
	case ocmetrics.MetricDescriptor_GAUGE_DOUBLE:
		return data.MetricTypeGaugeDouble
	case ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION:
		return data.MetricTypeGaugeHistogram
	case ocmetrics.MetricDescriptor_CUMULATIVE_INT64:
		return data.MetricTypeCounterInt64
	case ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE:
		return data.MetricTypeCounterDouble
	case ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		return data.MetricTypeCumulativeHistogram
	case ocmetrics.MetricDescriptor_SUMMARY:
		return data.MetricTypeSummary
	default:
		return invalidMetricType
	}
}

// setDataPoints converts OC timeseries to internal datapoints based on metric type
func setDataPoints(ocMetric *ocmetrics.Metric, metric data.Metric) {
	var int64DataPointsNum, doubleDataPointsNum, histogramDataPointsNum, summaryDataPointsNum int
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	ocPointsCount := getPointsCount(ocMetric)
	for _, timeseries := range ocMetric.GetTimeseries() {
		startTimestamp := internal.TimestampToUnixNano(timeseries.GetStartTimestamp())

		for _, point := range timeseries.GetPoints() {
			pointTimestamp := internal.TimestampToUnixNano(point.GetTimestamp())
			switch point.Value.(type) {

			case *ocmetrics.Point_Int64Value:
				dataPoints := metric.Int64DataPoints()
				if dataPoints.Len() == 0 {
					dataPoints.Resize(ocPointsCount)
				}
				dataPoint := dataPoints.At(int64DataPointsNum)
				dataPoint.SetStartTime(startTimestamp)
				dataPoint.SetTimestamp(pointTimestamp)
				setInt64DataPointValue(dataPoint, point)
				setLabelsMap(ocLabelsKeys, timeseries.GetLabelValues(), dataPoint.LabelsMap())
				int64DataPointsNum++

			case *ocmetrics.Point_DoubleValue:
				dataPoints := metric.DoubleDataPoints()
				if dataPoints.Len() == 0 {
					dataPoints.Resize(ocPointsCount)
				}
				dataPoint := dataPoints.At(doubleDataPointsNum)
				dataPoint.SetStartTime(startTimestamp)
				dataPoint.SetTimestamp(pointTimestamp)
				setDoubleDataPointValue(dataPoint, point)
				setLabelsMap(ocLabelsKeys, timeseries.GetLabelValues(), dataPoint.LabelsMap())
				doubleDataPointsNum++

			case *ocmetrics.Point_DistributionValue:
				dataPoints := metric.HistogramDataPoints()
				if dataPoints.Len() == 0 {
					dataPoints.Resize(ocPointsCount)
				}
				dataPoint := dataPoints.At(histogramDataPointsNum)
				dataPoint.SetStartTime(startTimestamp)
				dataPoint.SetTimestamp(pointTimestamp)
				setHistogramDataPointValue(dataPoint, point)
				setLabelsMap(ocLabelsKeys, timeseries.GetLabelValues(), dataPoint.LabelsMap())
				histogramDataPointsNum++

			case *ocmetrics.Point_SummaryValue:
				dataPoints := metric.SummaryDataPoints()
				if dataPoints.Len() == 0 {
					dataPoints.Resize(ocPointsCount)
				}
				dataPoint := dataPoints.At(summaryDataPointsNum)
				dataPoint.SetStartTime(startTimestamp)
				dataPoint.SetTimestamp(pointTimestamp)
				setSummaryDataPointValue(dataPoint, point)
				setLabelsMap(ocLabelsKeys, timeseries.GetLabelValues(), dataPoint.LabelsMap())
				summaryDataPointsNum++
			}
		}
	}
}

func setLabelsMap(ocLabelsKeys []*ocmetrics.LabelKey, ocLabelValues []*ocmetrics.LabelValue, labelsMap data.StringMap) {
	if len(ocLabelsKeys) == 0 || len(ocLabelValues) == 0 {
		return
	}

	lablesCount := len(ocLabelsKeys)

	// Handle invalid length of OC label values list
	if len(ocLabelValues) < lablesCount {
		lablesCount = len(ocLabelValues)
	}

	for i := 0; i < lablesCount; i++ {
		if !ocLabelValues[i].GetHasValue() {
			continue
		}
		labelsMap.Insert(ocLabelsKeys[i].Key, ocLabelValues[i].Value)
	}
}

func setInt64DataPointValue(dataPoint data.Int64DataPoint, point *ocmetrics.Point) {
	dataPoint.SetValue(point.GetInt64Value())
}

func setDoubleDataPointValue(dataPoint data.DoubleDataPoint, point *ocmetrics.Point) {
	dataPoint.SetValue(point.GetDoubleValue())
}

func setHistogramDataPointValue(dataPoint data.HistogramDataPoint, point *ocmetrics.Point) {
	distributionValue := point.GetDistributionValue()
	dataPoint.SetSum(distributionValue.GetSum())
	dataPoint.SetCount(uint64(distributionValue.GetCount()))
	histogramBucketsToInternal(distributionValue.GetBuckets(), dataPoint.Buckets())
	dataPoint.SetExplicitBounds(distributionValue.GetBucketOptions().GetExplicit().GetBounds())
}

func histogramBucketsToInternal(ocBuckets []*ocmetrics.DistributionValue_Bucket, buckets data.HistogramBucketSlice) {
	buckets.Resize(len(ocBuckets))
	for i := 0; i < buckets.Len(); i++ {
		bucket := buckets.At(i)
		bucket.SetCount(uint64(ocBuckets[i].GetCount()))
		if ocBuckets[i].GetExemplar() != nil {
			bucket.Exemplar().InitEmpty()
			exemplarToInternal(ocBuckets[i].GetExemplar(), bucket.Exemplar())
		}
	}
}

func exemplarToInternal(ocExemplar *ocmetrics.DistributionValue_Exemplar, exemplar data.HistogramBucketExemplar) {
	if ocExemplar.GetTimestamp() != nil {
		exemplar.SetTimestamp(internal.TimestampToUnixNano(ocExemplar.GetTimestamp()))
	}
	exemplar.SetValue(ocExemplar.GetValue())
	attachments := exemplar.Attachments()
	for k, v := range ocExemplar.GetAttachments() {
		attachments.Upsert(k, v)
	}
}

func setSummaryDataPointValue(dataPoint data.SummaryDataPoint, point *ocmetrics.Point) {
	summaryValue := point.GetSummaryValue()
	dataPoint.SetSum(summaryValue.GetSum().GetValue())
	dataPoint.SetCount(uint64(summaryValue.GetCount().GetValue()))
	percentileToInternal(summaryValue.GetSnapshot().GetPercentileValues(), dataPoint.ValueAtPercentiles())
}

func percentileToInternal(
	ocPercentiles []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile,
	percentiles data.SummaryValueAtPercentileSlice,
) {
	percentiles.Resize(len(ocPercentiles))
	for i := 0; i < percentiles.Len(); i++ {
		percentile := percentiles.At(i)
		percentile.SetPercentile(ocPercentiles[i].GetPercentile())
		percentile.SetValue(ocPercentiles[i].GetValue())
	}
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
