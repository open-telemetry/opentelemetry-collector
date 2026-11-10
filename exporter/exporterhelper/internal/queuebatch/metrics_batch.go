// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MergeSplit splits and/or merges the provided metrics request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *metricsRequest) MergeSplit(_ context.Context, maxSize int, szt request.SizerType, r2 request.Request) ([]request.Request, error) {
	var sz sizer.MetricsSizer
	switch szt {
	case request.SizerTypeItems:
		sz = &sizer.MetricsCountSizer{}
	case request.SizerTypeBytes:
		sz = &sizer.MetricsBytesSizer{}
	default:
		return nil, errors.New("unknown sizer type")
	}

	if r2 != nil {
		req2, ok := r2.(*metricsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sz, szt)
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []request.Request{req}, nil
	}
	return req.split(maxSize, sz, szt)
}

func (req *metricsRequest) mergeTo(dst *metricsRequest, sz sizer.MetricsSizer, szt request.SizerType) {
	if sz != nil {
		dst.sizes.Update(szt, dst.size(sz, szt)+req.size(sz, szt))
		req.sizes.Update(szt, 0)
	}
	req.md.ResourceMetrics().MoveAndAppendTo(dst.md.ResourceMetrics())
}

func (req *metricsRequest) split(maxSize int, sz sizer.MetricsSizer, szt request.SizerType) ([]request.Request, error) {
	if req.size(sz, szt) <= maxSize {
		return []request.Request{req}, nil
	}
	var res []request.Request
	droppedItems := 0
	droppedOversizedContainer := false
	for req.size(sz, szt) > maxSize {
		dataPointsBefore := req.md.DataPointCount()
		md, removedSize, containerTooLarge := extractMetricsTracked(req.md, maxSize, sz)
		if containerTooLarge {
			droppedOversizedContainer = true
		}
		if removedSize == 0 {
			// Nothing left the source, so no progress is possible. Stop rather than loop.
			return res, fmt.Errorf("request size is greater than max size and cannot be split further, dropping items: %d", droppedItems+req.md.DataPointCount())
		}
		req.sizes.Update(szt, req.size(sz, szt)-removedSize)
		// Data points that left src without landing in dest were dropped as oversized.
		droppedItems += dataPointsBefore - req.md.DataPointCount() - md.DataPointCount()
		if md.DataPointCount() > 0 {
			res = append(res, newMetricsRequest(md))
		}
	}
	// Splitting can leave nothing to export once oversized data points and data-point-less resources are gone.
	if req.md.DataPointCount() > 0 {
		res = append(res, req)
	}
	if droppedItems > 0 {
		if droppedOversizedContainer {
			return res, fmt.Errorf("request size is greater than max size and cannot be split further, dropping items: %d", droppedItems)
		}
		return res, fmt.Errorf("single data point exceeds the max size limit, dropping items: %d", droppedItems)
	}
	return res, nil
}

// deltaGrowth returns the length-prefix overhead added when a message of size v is
// wrapped in a parent repeated field, clamping non-positive sizes so sz.DeltaSize
// is never invoked with negative values.
func deltaGrowth(sz sizer.MetricsSizer, v int) int {
	if v <= 0 {
		return sz.DeltaSize(0)
	}
	return sz.DeltaSize(v) - v
}

// extractMetrics extracts metrics from srcMetrics until capacity is reached.
func extractMetrics(srcMetrics pmetric.Metrics, capacity int, sz sizer.MetricsSizer) (pmetric.Metrics, int) {
	destMetrics, removedSize, _ := extractMetricsTracked(srcMetrics, capacity, sz)
	return destMetrics, removedSize
}

func extractMetricsTracked(srcMetrics pmetric.Metrics, capacity int, sz sizer.MetricsSizer) (pmetric.Metrics, int, bool) {
	destMetrics := pmetric.NewMetrics()
	capacityLeft := capacity - sz.MetricsSize(destMetrics)
	removedSize := 0
	droppedOversizedContainer := false
	srcMetrics.ResourceMetrics().RemoveIf(func(srcRM pmetric.ResourceMetrics) bool {
		// If there is no more capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}
		rawRmSize := sz.ResourceMetricsSize(srcRM)
		rmSize := sz.DeltaSize(rawRmSize)
		if rmSize > capacityLeft {
			extSrcRM, extRmSize, rmContainerTooLarge := extractResourceMetrics(srcRM, capacityLeft, capacity, sz)
			if rmContainerTooLarge {
				droppedOversizedContainer = true
			}
			// It is possible that for the bytes scenario, the extracted field contains no scope metrics.
			// Do not add it to the destination if that is the case.
			if extSrcRM.ScopeMetrics().Len() > 0 {
				extSrcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
				// This cannot make it to exactly 0 for the bytes,
				// force it to be 0 since that is the stopping condition.
				capacityLeft = 0
			}
			if srcRM.ScopeMetrics().Len() == 0 {
				// Nothing is left in the source resource, so all of it is removed.
				removedSize += rmSize
				return true
			}
			// Remaining scopes in srcRM did not fit this batch, so stop here until the next pass.
			capacityLeft = 0
			// The source resource shrinks to the delta size of what is left in it.
			removedSize += rmSize - sz.DeltaSize(rawRmSize-extRmSize)
			return false
		}
		capacityLeft -= rmSize
		removedSize += rmSize
		srcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
		return true
	})
	return destMetrics, removedSize, droppedOversizedContainer
}

// extractResourceMetrics extracts resource metrics and returns a new resource metrics with the specified number of data points.
func extractResourceMetrics(srcRM pmetric.ResourceMetrics, capacity, maxSize int, sz sizer.MetricsSizer) (pmetric.ResourceMetrics, int, bool) {
	destRM := pmetric.NewResourceMetrics()
	destRM.SetSchemaUrl(srcRM.SchemaUrl())
	srcRM.Resource().CopyTo(destRM.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - deltaGrowth(sz, capacity) - sz.ResourceMetricsSize(destRM)
	// Room for a scope in an otherwise empty batch, once this resource's header and attributes are paid for.
	maxScopeSize := maxSize - deltaGrowth(sz, maxSize) - sz.ResourceMetricsSize(destRM)
	removedSize := 0
	droppedOversizedContainer := false
	if maxScopeSize <= 0 {
		// The resource header alone fills or exceeds an empty batch, so none of its scopes,
		// metrics, or data points can ever fit. Drop the whole resource so trailing valid
		// resources are not blocked.
		srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
			removedSize += sz.DeltaSize(sz.ScopeMetricsSize(srcSM))
			srcSM.Metrics().RemoveIf(func(srcMetric pmetric.Metric) bool {
				if dataPointsLen(srcMetric) > 0 {
					droppedOversizedContainer = true
				}
				return true
			})
			return true
		})
		return destRM, removedSize, droppedOversizedContainer
	}
	srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
		rawSmSize := sz.ScopeMetricsSize(srcSM)
		smSize := sz.DeltaSize(rawSmSize)
		// Prune empty metric wrappers before checking capacityLeft so itemless oversized
		// resources do not leave stale wrappers behind and block subsequent resources.
		srcSM.Metrics().RemoveIf(func(srcMetric pmetric.Metric) bool {
			return dataPointsLen(srcMetric) == 0
		})
		if srcSM.Metrics().Len() == 0 {
			removedSize += smSize
			return true
		}
		if newRawSmSize := sz.ScopeMetricsSize(srcSM); newRawSmSize != rawSmSize {
			newSmSize := sz.DeltaSize(newRawSmSize)
			removedSize += smSize - newSmSize
			rawSmSize = newRawSmSize
			smSize = newSmSize
		}
		// If there is no capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}
		if smSize > capacityLeft {
			extSrcSM, extSmSize, smContainerTooLarge := extractScopeMetrics(srcSM, capacityLeft, maxScopeSize, sz)
			if smContainerTooLarge {
				droppedOversizedContainer = true
			}
			// It is possible that for the bytes scenario, the extracted field contains no metrics.
			// Do not add it to the destination if that is the case.
			if extSrcSM.Metrics().Len() > 0 {
				extSrcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
				// This cannot make it to exactly 0 for the bytes,
				// force it to be 0 since that is the stopping condition.
				capacityLeft = 0
			}
			if srcSM.Metrics().Len() == 0 {
				// Nothing is left in the source scope, so all of it is removed.
				removedSize += smSize
				return true
			}
			// Remaining metrics in srcSM did not fit this batch, so stop here until the next pass.
			capacityLeft = 0
			// The source scope shrinks to the delta size of what is left in it.
			removedSize += smSize - sz.DeltaSize(rawSmSize-extSmSize)
			return false
		}
		capacityLeft -= smSize
		removedSize += smSize
		srcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
		return true
	})
	return destRM, removedSize, droppedOversizedContainer
}

// extractScopeMetrics extracts scope metrics and returns a new scope metrics with the specified number of data points.
func extractScopeMetrics(srcSM pmetric.ScopeMetrics, capacity, maxScopeSize int, sz sizer.MetricsSizer) (pmetric.ScopeMetrics, int, bool) {
	destSM := pmetric.NewScopeMetrics()
	destSM.SetSchemaUrl(srcSM.SchemaUrl())
	srcSM.Scope().CopyTo(destSM.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - deltaGrowth(sz, capacity) - sz.ScopeMetricsSize(destSM)
	// Room for a metric in an otherwise empty batch, once the resource and scope headers and attributes are paid for.
	maxMetricSize := maxScopeSize - deltaGrowth(sz, maxScopeSize) - sz.ScopeMetricsSize(destSM)
	removedSize := 0
	if maxMetricSize <= 0 {
		// The resource and scope headers alone fill or exceed an empty batch, so none of this
		// scope's metrics or data points can ever fit. Drop the whole scope so trailing valid
		// scopes or resources are not blocked.
		droppedOversizedContainer := false
		srcSM.Metrics().RemoveIf(func(srcMetric pmetric.Metric) bool {
			if dataPointsLen(srcMetric) > 0 {
				droppedOversizedContainer = true
			}
			removedSize += sz.DeltaSize(sz.MetricSize(srcMetric))
			return true
		})
		return destSM, removedSize, droppedOversizedContainer
	}
	srcSM.Metrics().RemoveIf(func(srcMetric pmetric.Metric) bool {
		if dataPointsLen(srcMetric) == 0 {
			removedSize += sz.DeltaSize(sz.MetricSize(srcMetric))
			return true
		}
		// If there is no capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}
		mSize := sz.DeltaSize(sz.MetricSize(srcMetric))
		if mSize > capacityLeft {
			extSrcMetric := extractMetricDataPoints(srcMetric, capacityLeft, maxMetricSize, sz)
			// It is possible that for the bytes scenario, the extracted field contains no datapoints.
			// Do not add it to the destination if that is the case.
			if dataPointsLen(extSrcMetric) > 0 {
				extSrcMetric.MoveTo(destSM.Metrics().AppendEmpty())
				// This cannot make it to exactly 0 for the bytes,
				// force it to be 0 since that is the stopping condition.
				capacityLeft = 0
			}
			if dataPointsLen(srcMetric) == 0 {
				// Nothing is left in the source metric, so all of it is removed.
				removedSize += mSize
				return true
			}
			// Remaining data points in srcMetric did not fit this batch, so stop here until the next pass.
			capacityLeft = 0
			// Re-measure: the nested data message length prefix can shrink too.
			removedSize += mSize - sz.DeltaSize(sz.MetricSize(srcMetric))
			return false
		}
		capacityLeft -= mSize
		removedSize += mSize
		srcMetric.MoveTo(destSM.Metrics().AppendEmpty())
		return true
	})
	return destSM, removedSize, false
}

// extractMetricDataPoints extracts data points and returns a new metric with the specified number of data points.
func extractMetricDataPoints(srcMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) pmetric.Metric {
	destMetric := pmetric.NewMetric()
	destMetric.SetName(srcMetric.Name())
	destMetric.SetDescription(srcMetric.Description())
	destMetric.SetUnit(srcMetric.Unit())
	srcMetric.Metadata().CopyTo(destMetric.Metadata())

	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		extractGaugeDataPoints(srcMetric.Gauge(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeSum:
		extractSumDataPoints(srcMetric.Sum(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeHistogram:
		extractHistogramDataPoints(srcMetric.Histogram(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeExponentialHistogram:
		extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeSummary:
		extractSummaryDataPoints(srcMetric.Summary(), destMetric, capacity, maxMetricSize, sz)
	}
	return destMetric
}

func dataPointsLen(m pmetric.Metric) int {
	switch m.Type() {
	case pmetric.MetricTypeGauge:
		return m.Gauge().DataPoints().Len()
	case pmetric.MetricTypeSum:
		return m.Sum().DataPoints().Len()
	case pmetric.MetricTypeHistogram:
		return m.Histogram().DataPoints().Len()
	case pmetric.MetricTypeExponentialHistogram:
		return m.ExponentialHistogram().DataPoints().Len()
	case pmetric.MetricTypeSummary:
		return m.Summary().DataPoints().Len()
	}
	return 0
}

// dataPointBudget reserves capacityLeft and maxDataPointSize after accounting for the metric
// header and the worst-case varint length-prefix growth of the nested data message at capacity
// and maxMetricSize. Reserving prefix growth conservatively at the full budget is intentional
// so split batches never exceed maxSize across varint width boundaries.
func dataPointBudget(destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) (capacityLeft, maxDataPointSize int) {
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size,
	// plus any extra varint width growth of the nested data message above an empty one.
	metricHeaderSize := sz.MetricSize(destMetric)
	capacityLeft = capacity - deltaGrowth(sz, capacity) - metricHeaderSize - (deltaGrowth(sz, capacity) - sz.DeltaSize(0))
	maxDataPointSize = maxMetricSize - deltaGrowth(sz, maxMetricSize) - metricHeaderSize - (deltaGrowth(sz, maxMetricSize) - sz.DeltaSize(0))
	return capacityLeft, maxDataPointSize
}

func extractGaugeDataPoints(srcGauge pmetric.Gauge, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) {
	destGauge := destMetric.SetEmptyGauge()
	capacityLeft, maxDataPointSize := dataPointBudget(destMetric, capacity, maxMetricSize, sz)
	if maxDataPointSize <= 0 {
		// The metric header alone fills or exceeds an empty batch, so none of its data points can ever fit.
		srcGauge.DataPoints().RemoveIf(func(pmetric.NumberDataPoint) bool { return true })
		return
	}
	if capacityLeft <= 0 {
		return
	}

	srcGauge.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If there is no more capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destGauge.DataPoints().AppendEmpty())
		return true
	})
}

func extractSumDataPoints(srcSum pmetric.Sum, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) {
	destSum := destMetric.SetEmptySum()
	destSum.SetIsMonotonic(srcSum.IsMonotonic())
	destSum.SetAggregationTemporality(srcSum.AggregationTemporality())

	capacityLeft, maxDataPointSize := dataPointBudget(destMetric, capacity, maxMetricSize, sz)
	if maxDataPointSize <= 0 {
		// The metric header alone fills or exceeds an empty batch, so none of its data points can ever fit.
		srcSum.DataPoints().RemoveIf(func(pmetric.NumberDataPoint) bool { return true })
		return
	}
	if capacityLeft <= 0 {
		return
	}

	srcSum.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If there is no more capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destSum.DataPoints().AppendEmpty())
		return true
	})
}

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) {
	destHistogram := destMetric.SetEmptyHistogram()
	destHistogram.SetAggregationTemporality(srcHistogram.AggregationTemporality())

	capacityLeft, maxDataPointSize := dataPointBudget(destMetric, capacity, maxMetricSize, sz)
	if maxDataPointSize <= 0 {
		// The metric header alone fills or exceeds an empty batch, so none of its data points can ever fit.
		srcHistogram.DataPoints().RemoveIf(func(pmetric.HistogramDataPoint) bool { return true })
		return
	}
	if capacityLeft <= 0 {
		return
	}

	srcHistogram.DataPoints().RemoveIf(func(srcDP pmetric.HistogramDataPoint) bool {
		// If there is no more capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.HistogramDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destHistogram.DataPoints().AppendEmpty())
		return true
	})
}

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) {
	destExponentialHistogram := destMetric.SetEmptyExponentialHistogram()
	destExponentialHistogram.SetAggregationTemporality(srcExponentialHistogram.AggregationTemporality())

	capacityLeft, maxDataPointSize := dataPointBudget(destMetric, capacity, maxMetricSize, sz)
	if maxDataPointSize <= 0 {
		// The metric header alone fills or exceeds an empty batch, so none of its data points can ever fit.
		srcExponentialHistogram.DataPoints().RemoveIf(func(pmetric.ExponentialHistogramDataPoint) bool { return true })
		return
	}
	if capacityLeft <= 0 {
		return
	}

	srcExponentialHistogram.DataPoints().RemoveIf(func(srcDP pmetric.ExponentialHistogramDataPoint) bool {
		// If there is no more capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.ExponentialHistogramDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destExponentialHistogram.DataPoints().AppendEmpty())
		return true
	})
}

func extractSummaryDataPoints(srcSummary pmetric.Summary, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) {
	destSummary := destMetric.SetEmptySummary()

	capacityLeft, maxDataPointSize := dataPointBudget(destMetric, capacity, maxMetricSize, sz)
	if maxDataPointSize <= 0 {
		// The metric header alone fills or exceeds an empty batch, so none of its data points can ever fit.
		srcSummary.DataPoints().RemoveIf(func(pmetric.SummaryDataPoint) bool { return true })
		return
	}
	if capacityLeft <= 0 {
		return
	}

	srcSummary.DataPoints().RemoveIf(func(srcDP pmetric.SummaryDataPoint) bool {
		// If there is no more capacity left, stop.
		if capacityLeft <= 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.SummaryDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destSummary.DataPoints().AppendEmpty())
		return true
	})
}
