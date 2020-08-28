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
	"go.opentelemetry.io/collector/internal"
	"go.opentelemetry.io/collector/internal/dataold"
)

const (
	invalidMetricType = dataold.MetricType(-1)
)

// OCSliceToMetricData converts a slice of OC data format to MetricData.
func OCSliceToMetricData(ocmds []consumerdata.MetricsData) dataold.MetricData {
	metricData := dataold.NewMetricData()
	if len(ocmds) == 0 {
		return metricData
	}
	for _, ocmd := range ocmds {
		appendOcToMetricData(ocmd, metricData)
	}
	return metricData
}

// OCToMetricData converts OC data format to MetricData.
func OCToMetricData(md consumerdata.MetricsData) dataold.MetricData {
	metricData := dataold.NewMetricData()
	appendOcToMetricData(md, metricData)
	return metricData
}

func appendOcToMetricData(md consumerdata.MetricsData, dest dataold.MetricData) {
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
	// 1 (for all metrics with nil resource) + numMetricsWithResource (distinctResourceCount).
	rms.Resize(initialRmsLen + distinctResourceCount + 1)
	rm0 := rms.At(initialRmsLen)
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
			ocMetricToOldMetrics(ocMetric, combinedMetrics.At(combinedMetricIdx))
			combinedMetricIdx++
		} else {
			// This metric has a different Resource and must be placed in a different
			// ResourceMetrics instance. Create a separate ResourceMetrics item just for this metric
			// and store at resourceMetricIdx.
			ocMetricToOldResourceMetrics(ocMetric, md.Node, rms.At(initialRmsLen+resourceMetricIdx))
			resourceMetricIdx++
		}
	}
}

func ocMetricToOldResourceMetrics(ocMetric *ocmetrics.Metric, node *occommon.Node, out dataold.ResourceMetrics) {
	ocNodeResourceToInternal(node, ocMetric.Resource, out.Resource())
	ilms := out.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	metrics := ilms.At(0).Metrics()
	metrics.Resize(1)
	ocMetricToOldMetrics(ocMetric, metrics.At(0))
}

// ocMetricToOldMetrics conversts ocMetric to internal representation and fill metric
func ocMetricToOldMetrics(ocMetric *ocmetrics.Metric, metric dataold.Metric) {
	descriptorToOldMetrics(ocMetric.GetMetricDescriptor(), metric.MetricDescriptor())
	setDataPointsToOldMetrics(ocMetric, metric)
}

func descriptorToOldMetrics(ocDescriptor *ocmetrics.MetricDescriptor, descriptor dataold.MetricDescriptor) {
	if ocDescriptor == nil {
		return
	}

	descriptorType := descriptorTypeToOldMetrics(ocDescriptor.Type)
	if descriptorType == invalidMetricType {
		return
	}

	descriptor.InitEmpty()
	descriptor.SetType(descriptorType)
	descriptor.SetDescription(ocDescriptor.GetDescription())
	descriptor.SetName(ocDescriptor.GetName())
	descriptor.SetUnit(ocDescriptor.GetUnit())
}

func descriptorTypeToOldMetrics(t ocmetrics.MetricDescriptor_Type) dataold.MetricType {
	switch t {
	case ocmetrics.MetricDescriptor_UNSPECIFIED:
		return dataold.MetricTypeInvalid
	case ocmetrics.MetricDescriptor_GAUGE_INT64:
		return dataold.MetricTypeInt64
	case ocmetrics.MetricDescriptor_GAUGE_DOUBLE:
		return dataold.MetricTypeDouble
	case ocmetrics.MetricDescriptor_CUMULATIVE_INT64:
		return dataold.MetricTypeMonotonicInt64
	case ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE:
		return dataold.MetricTypeMonotonicDouble
	case ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		return dataold.MetricTypeHistogram
	case ocmetrics.MetricDescriptor_SUMMARY:
		return dataold.MetricTypeSummary
	default:
		return invalidMetricType
	}
}

// setDataPointsToOldMetrics converts OC timeseries to internal datapoints based on metric type
func setDataPointsToOldMetrics(ocMetric *ocmetrics.Metric, metric dataold.Metric) {
	var int64DataPointsNum, doubleDataPointsNum, histogramDataPointsNum, summaryDataPointsNum int
	ocLabelsKeys := ocMetric.GetMetricDescriptor().GetLabelKeys()
	ocPointsCount := getPointsCount(ocMetric)
	for _, timeseries := range ocMetric.GetTimeseries() {
		if timeseries == nil {
			continue
		}
		startTimestamp := internal.TimestampToUnixNano(timeseries.GetStartTimestamp())

		for _, point := range timeseries.GetPoints() {
			if point == nil {
				continue
			}
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
				dataPoint.SetValue(point.GetInt64Value())
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
				dataPoint.SetValue(point.GetDoubleValue())
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
				setSummaryDataPointValueToOldMetrics(dataPoint, point)
				setLabelsMap(ocLabelsKeys, timeseries.GetLabelValues(), dataPoint.LabelsMap())
				summaryDataPointsNum++
			}
		}
	}

	// If we allocated any of the data points slices, we did allocate them with empty points (resize),
	// so make sure that we do not leave empty points at the end of any slice.

	if metric.Int64DataPoints().Len() != 0 {
		metric.Int64DataPoints().Resize(int64DataPointsNum)
	}
	if metric.DoubleDataPoints().Len() != 0 {
		metric.DoubleDataPoints().Resize(doubleDataPointsNum)
	}
	if metric.HistogramDataPoints().Len() != 0 {
		metric.HistogramDataPoints().Resize(histogramDataPointsNum)
	}
	if metric.SummaryDataPoints().Len() != 0 {
		metric.SummaryDataPoints().Resize(summaryDataPointsNum)
	}
}

func setHistogramDataPointValue(dataPoint dataold.HistogramDataPoint, point *ocmetrics.Point) {
	distributionValue := point.GetDistributionValue()
	dataPoint.SetSum(distributionValue.GetSum())
	dataPoint.SetCount(uint64(distributionValue.GetCount()))
	histogramBucketsToOldMetrics(distributionValue.GetBuckets(), dataPoint.Buckets())
	dataPoint.SetExplicitBounds(distributionValue.GetBucketOptions().GetExplicit().GetBounds())
}

func histogramBucketsToOldMetrics(ocBuckets []*ocmetrics.DistributionValue_Bucket, buckets dataold.HistogramBucketSlice) {
	buckets.Resize(len(ocBuckets))
	for i := 0; i < buckets.Len(); i++ {
		bucket := buckets.At(i)
		bucket.SetCount(uint64(ocBuckets[i].GetCount()))
		if ocBuckets[i].GetExemplar() != nil {
			bucket.Exemplar().InitEmpty()
			exemplarToOldMetrics(ocBuckets[i].GetExemplar(), bucket.Exemplar())
		}
	}
}

func exemplarToOldMetrics(ocExemplar *ocmetrics.DistributionValue_Exemplar, exemplar dataold.HistogramBucketExemplar) {
	if ocExemplar.GetTimestamp() != nil {
		exemplar.SetTimestamp(internal.TimestampToUnixNano(ocExemplar.GetTimestamp()))
	}
	exemplar.SetValue(ocExemplar.GetValue())
	attachments := exemplar.Attachments()
	ocAttachments := ocExemplar.GetAttachments()
	attachments.InitEmptyWithCapacity(len(ocAttachments))
	for k, v := range ocAttachments {
		attachments.Upsert(k, v)
	}
}

func setSummaryDataPointValueToOldMetrics(dataPoint dataold.SummaryDataPoint, point *ocmetrics.Point) {
	summaryValue := point.GetSummaryValue()
	dataPoint.SetSum(summaryValue.GetSum().GetValue())
	dataPoint.SetCount(uint64(summaryValue.GetCount().GetValue()))
	percentileToOldMetrics(summaryValue.GetSnapshot().GetPercentileValues(), dataPoint.ValueAtPercentiles())
}

func percentileToOldMetrics(
	ocPercentiles []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile,
	percentiles dataold.SummaryValueAtPercentileSlice,
) {
	percentiles.Resize(len(ocPercentiles))
	for i := 0; i < percentiles.Len(); i++ {
		percentile := percentiles.At(i)
		percentile.SetPercentile(ocPercentiles[i].GetPercentile())
		percentile.SetValue(ocPercentiles[i].GetValue())
	}
}
