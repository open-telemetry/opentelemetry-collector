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

package batchprocessor

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// splitMetrics removes metrics from the input data and returns a new data of the specified size.
func splitMetrics(size int, src pdata.Metrics) pdata.Metrics {
	_, dataPoints := src.MetricAndDataPointCount()
	if dataPoints <= size {
		return src
	}
	totalCopiedDataPoints := 0
	dest := pdata.NewMetrics()

	src.ResourceMetrics().RemoveIf(func(srcRs pdata.ResourceMetrics) bool {
		// If we are done skip everything else.
		if totalCopiedDataPoints == size {
			return false
		}

		destRs := dest.ResourceMetrics().AppendEmpty()
		srcRs.Resource().CopyTo(destRs.Resource())

		srcRs.InstrumentationLibraryMetrics().RemoveIf(func(srcIlm pdata.InstrumentationLibraryMetrics) bool {
			// If we are done skip everything else.
			if totalCopiedDataPoints == size {
				return false
			}

			destIlm := destRs.InstrumentationLibraryMetrics().AppendEmpty()
			srcIlm.InstrumentationLibrary().CopyTo(destIlm.InstrumentationLibrary())

			// If possible to move all metrics do that.
			srcDataPointCount := metricSliceDataPointCount(srcIlm.Metrics())
			if size-totalCopiedDataPoints >= srcDataPointCount {
				totalCopiedDataPoints += srcDataPointCount
				srcIlm.Metrics().MoveAndAppendTo(destIlm.Metrics())
				return true
			}

			srcIlm.Metrics().RemoveIf(func(srcMetric pdata.Metric) bool {
				// If we are done skip everything else.
				if totalCopiedDataPoints == size {
					return false
				}
				// If the metric has more data points than free slots we should split it.
				copiedDataPoints, remove := splitMetric(srcMetric, destIlm.Metrics().AppendEmpty(), size-totalCopiedDataPoints)
				totalCopiedDataPoints += copiedDataPoints
				return remove
			})
			return false
		})
		return srcRs.InstrumentationLibraryMetrics().Len() == 0
	})

	return dest
}

// metricSliceDataPointCount calculates the total number of  data points.
func metricSliceDataPointCount(ms pdata.MetricSlice) (dataPointCount int) {
	for k := 0; k < ms.Len(); k++ {
		dataPointCount += metricDataPointCount(ms.At(k))
	}
	return
}

// metricDataPointCount calculates the total number of  data points.
func metricDataPointCount(ms pdata.Metric) (dataPointCount int) {
	switch ms.DataType() {
	case pdata.MetricDataTypeIntGauge:
		dataPointCount = ms.IntGauge().DataPoints().Len()
	case pdata.MetricDataTypeDoubleGauge:
		dataPointCount = ms.DoubleGauge().DataPoints().Len()
	case pdata.MetricDataTypeIntSum:
		dataPointCount = ms.IntSum().DataPoints().Len()
	case pdata.MetricDataTypeDoubleSum:
		dataPointCount = ms.DoubleSum().DataPoints().Len()
	case pdata.MetricDataTypeIntHistogram:
		dataPointCount = ms.IntHistogram().DataPoints().Len()
	case pdata.MetricDataTypeHistogram:
		dataPointCount = ms.Histogram().DataPoints().Len()
	case pdata.MetricDataTypeSummary:
		dataPointCount = ms.Summary().DataPoints().Len()
	}
	return
}

// splitMetric removes metric points from the input data and moves data of the specified size to destination.
// Returns size of moved data and boolean describing, whether the metric should be removed from original slice.
func splitMetric(ms, dest pdata.Metric, size int) (int, bool) {
	ms.CopyTo(dest)
	if metricDataPointCount(ms) <= size {
		return metricDataPointCount(ms), true
	}

	msSize, i := metricDataPointCount(ms)-size, 0
	filterDataPoints := func() bool { i++; return i <= msSize }
	switch ms.DataType() {
	case pdata.MetricDataTypeIntGauge:
		dest.IntGauge().DataPoints().Resize(size)
		ms.IntGauge().DataPoints().RemoveIf(func(_ pdata.IntDataPoint) bool {
			return filterDataPoints()
		})
	case pdata.MetricDataTypeDoubleGauge:
		dest.DoubleGauge().DataPoints().Resize(size)
		ms.DoubleGauge().DataPoints().RemoveIf(func(_ pdata.DoubleDataPoint) bool {
			return filterDataPoints()
		})
	case pdata.MetricDataTypeIntSum:
		dest.IntSum().DataPoints().Resize(size)
		ms.IntSum().DataPoints().RemoveIf(func(_ pdata.IntDataPoint) bool {
			return filterDataPoints()
		})
	case pdata.MetricDataTypeDoubleSum:
		dest.DoubleSum().DataPoints().Resize(size)
		ms.DoubleSum().DataPoints().RemoveIf(func(_ pdata.DoubleDataPoint) bool {
			return filterDataPoints()
		})
	case pdata.MetricDataTypeIntHistogram:
		dest.IntHistogram().DataPoints().Resize(size)
		ms.IntHistogram().DataPoints().RemoveIf(func(_ pdata.IntHistogramDataPoint) bool {
			return filterDataPoints()
		})
	case pdata.MetricDataTypeHistogram:
		dest.Histogram().DataPoints().Resize(size)
		ms.Histogram().DataPoints().RemoveIf(func(_ pdata.HistogramDataPoint) bool {
			return filterDataPoints()
		})
	case pdata.MetricDataTypeSummary:
		dest.Summary().DataPoints().Resize(size)
		ms.Summary().DataPoints().RemoveIf(func(_ pdata.SummaryDataPoint) bool {
			return filterDataPoints()
		})
	}
	return size, false
}
