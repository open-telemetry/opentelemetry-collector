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

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// splitMetrics removes metrics from the input data and returns a new data of the specified size.
func splitMetrics(size int, src pmetric.Metrics) pmetric.Metrics {
	dataPoints := src.DataPointCount()
	if dataPoints <= size {
		return src
	}
	totalCopiedDataPoints := 0
	dest := pmetric.NewMetrics()

	src.MutableResourceMetrics().RemoveIf(func(srcRs pmetric.MutableResourceMetrics) bool {
		// If we are done skip everything else.
		if totalCopiedDataPoints == size {
			return false
		}

		// If it fully fits
		srcRsDataPointCount := resourceMetricsDPC(srcRs)
		if (totalCopiedDataPoints + srcRsDataPointCount) <= size {
			totalCopiedDataPoints += srcRsDataPointCount
			srcRs.MoveTo(dest.MutableResourceMetrics().AppendEmpty())
			return true
		}

		destRs := dest.MutableResourceMetrics().AppendEmpty()
		srcRs.Resource().CopyTo(destRs.Resource())
		srcRs.ScopeMetrics().RemoveIf(func(srcIlm pmetric.MutableScopeMetrics) bool {
			// If we are done skip everything else.
			if totalCopiedDataPoints == size {
				return false
			}

			// If possible to move all metrics do that.
			srcIlmDataPointCount := scopeMetricsDPC(srcIlm)
			if srcIlmDataPointCount+totalCopiedDataPoints <= size {
				totalCopiedDataPoints += srcIlmDataPointCount
				srcIlm.MoveTo(destRs.ScopeMetrics().AppendEmpty())
				return true
			}

			destIlm := destRs.ScopeMetrics().AppendEmpty()
			srcIlm.Scope().CopyTo(destIlm.Scope())
			srcIlm.Metrics().RemoveIf(func(srcMetric pmetric.MutableMetric) bool {
				// If we are done skip everything else.
				if totalCopiedDataPoints == size {
					return false
				}

				// If possible to move all points do that.
				srcMetricPointCount := metricDPC(srcMetric)
				if srcMetricPointCount+totalCopiedDataPoints <= size {
					totalCopiedDataPoints += srcMetricPointCount
					srcMetric.MoveTo(destIlm.Metrics().AppendEmpty())
					return true
				}

				// If the metric has more data points than free slots we should split it.
				copiedDataPoints, remove := splitMetric(srcMetric, destIlm.Metrics().AppendEmpty(), size-totalCopiedDataPoints)
				totalCopiedDataPoints += copiedDataPoints
				return remove
			})
			return false
		})
		return srcRs.ScopeMetrics().Len() == 0
	})

	return dest
}

// resourceMetricsDPC calculates the total number of data points in the pmetric.ResourceMetrics.
func resourceMetricsDPC(rs pmetric.MutableResourceMetrics) int {
	dataPointCount := 0
	ilms := rs.ScopeMetrics()
	for k := 0; k < ilms.Len(); k++ {
		dataPointCount += scopeMetricsDPC(ilms.At(k))
	}
	return dataPointCount
}

// scopeMetricsDPC calculates the total number of data points in the pmetric.ScopeMetrics.
func scopeMetricsDPC(ilm pmetric.MutableScopeMetrics) int {
	dataPointCount := 0
	ms := ilm.Metrics()
	for k := 0; k < ms.Len(); k++ {
		dataPointCount += metricDPC(ms.At(k))
	}
	return dataPointCount
}

// metricDPC calculates the total number of data points in the pmetric.Metric.
func metricDPC(ms pmetric.MutableMetric) int {
	switch ms.Type() {
	case pmetric.MetricTypeGauge:
		return ms.Gauge().DataPoints().Len()
	case pmetric.MetricTypeSum:
		return ms.Sum().DataPoints().Len()
	case pmetric.MetricTypeHistogram:
		return ms.Histogram().DataPoints().Len()
	case pmetric.MetricTypeExponentialHistogram:
		return ms.ExponentialHistogram().DataPoints().Len()
	case pmetric.MetricTypeSummary:
		return ms.Summary().DataPoints().Len()
	}
	return 0
}

// splitMetric removes metric points from the input data and moves data of the specified size to destination.
// Returns size of moved data and boolean describing, whether the metric should be removed from original slice.
func splitMetric(ms, dest pmetric.MutableMetric, size int) (int, bool) {
	dest.SetName(ms.Name())
	dest.SetDescription(ms.Description())
	dest.SetUnit(ms.Unit())

	switch ms.Type() {
	case pmetric.MetricTypeGauge:
		return splitNumberDataPoints(ms.Gauge().DataPoints(), dest.SetEmptyGauge().DataPoints(), size)
	case pmetric.MetricTypeSum:
		destSum := dest.SetEmptySum()
		destSum.SetAggregationTemporality(ms.Sum().AggregationTemporality())
		destSum.SetIsMonotonic(ms.Sum().IsMonotonic())
		return splitNumberDataPoints(ms.Sum().DataPoints(), destSum.DataPoints(), size)
	case pmetric.MetricTypeHistogram:
		destHistogram := dest.SetEmptyHistogram()
		destHistogram.SetAggregationTemporality(ms.Histogram().AggregationTemporality())
		return splitHistogramDataPoints(ms.Histogram().DataPoints(), destHistogram.DataPoints(), size)
	case pmetric.MetricTypeExponentialHistogram:
		destHistogram := dest.SetEmptyExponentialHistogram()
		destHistogram.SetAggregationTemporality(ms.ExponentialHistogram().AggregationTemporality())
		return splitExponentialHistogramDataPoints(ms.ExponentialHistogram().DataPoints(), destHistogram.DataPoints(), size)
	case pmetric.MetricTypeSummary:
		return splitSummaryDataPoints(ms.Summary().DataPoints(), dest.SetEmptySummary().DataPoints(), size)
	}
	return size, false
}

func splitNumberDataPoints(src, dst pmetric.MutableNumberDataPointSlice, size int) (int, bool) {
	dst.EnsureCapacity(size)
	i := 0
	src.RemoveIf(func(dp pmetric.MutableNumberDataPoint) bool {
		if i < size {
			dp.MoveTo(dst.AppendEmpty())
			i++
			return true
		}
		return false
	})
	return size, false
}

func splitHistogramDataPoints(src, dst pmetric.MutableHistogramDataPointSlice, size int) (int, bool) {
	dst.EnsureCapacity(size)
	i := 0
	src.RemoveIf(func(dp pmetric.MutableHistogramDataPoint) bool {
		if i < size {
			dp.MoveTo(dst.AppendEmpty())
			i++
			return true
		}
		return false
	})
	return size, false
}

func splitExponentialHistogramDataPoints(src, dst pmetric.MutableExponentialHistogramDataPointSlice, size int) (int, bool) {
	dst.EnsureCapacity(size)
	i := 0
	src.RemoveIf(func(dp pmetric.MutableExponentialHistogramDataPoint) bool {
		if i < size {
			dp.MoveTo(dst.AppendEmpty())
			i++
			return true
		}
		return false
	})
	return size, false
}

func splitSummaryDataPoints(src, dst pmetric.MutableSummaryDataPointSlice, size int) (int, bool) {
	dst.EnsureCapacity(size)
	i := 0
	src.RemoveIf(func(dp pmetric.MutableSummaryDataPoint) bool {
		if i < size {
			dp.MoveTo(dst.AppendEmpty())
			i++
			return true
		}
		return false
	})
	return size, false
}
