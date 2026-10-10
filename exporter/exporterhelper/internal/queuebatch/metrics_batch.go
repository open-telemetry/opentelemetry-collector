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
	for req.size(sz, szt) > maxSize {
		dataPointsBefore := req.md.DataPointCount()
		md, removedSize := extractMetrics(req.md, maxSize, sz)
		if removedSize == 0 {
			// Nothing left the source, so no progress is possible. Stop rather than loop.
			return res, fmt.Errorf("request size is greater than max size and cannot be split further, dropping items: %d", droppedItems+req.md.DataPointCount())
		}
		req.sizes.Update(szt, req.size(sz, szt)-removedSize)
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
		return res, fmt.Errorf("single data point exceeds the max size limit, dropping items: %d", droppedItems)
	}
	return res, nil
}

// extractMetrics extracts metrics from srcMetrics until capacity is reached.
func extractMetrics(srcMetrics pmetric.Metrics, capacity int, sz sizer.MetricsSizer) (pmetric.Metrics, int) {
	destMetrics := pmetric.NewMetrics()
	capacityLeft := capacity - sz.MetricsSize(destMetrics)
	removedSize := 0
	srcMetrics.ResourceMetrics().RemoveIf(func(srcRM pmetric.ResourceMetrics) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRmSize := sz.ResourceMetricsSize(srcRM)
		rmSize := sz.DeltaSize(rawRmSize)
		if rmSize > capacityLeft {
			extSrcRM, extRmSize := extractResourceMetrics(srcRM, capacityLeft, capacity, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no scope metrics.
			// Do not add it to the destination if that is the case.
			if extSrcRM.ScopeMetrics().Len() > 0 {
				extSrcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
			}
			if srcRM.ScopeMetrics().Len() == 0 {
				// Nothing is left in the source resource, so all of it is removed.
				removedSize += rmSize
				return true
			}
			// The source resource shrinks to the delta size of what is left in it.
			removedSize += rmSize - sz.DeltaSize(rawRmSize-extRmSize)
			return false
		}
		capacityLeft -= rmSize
		removedSize += rmSize
		srcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
		return true
	})
	return destMetrics, removedSize
}

// extractResourceMetrics extracts resource metrics and returns a new resource metrics with the specified number of data points.
func extractResourceMetrics(srcRM pmetric.ResourceMetrics, capacity, maxSize int, sz sizer.MetricsSizer) (pmetric.ResourceMetrics, int) {
	destRM := pmetric.NewResourceMetrics()
	destRM.SetSchemaUrl(srcRM.SchemaUrl())
	srcRM.Resource().CopyTo(destRM.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceMetricsSize(destRM)
	// Room for a scope in an otherwise empty batch, once this resource's header and attributes are paid for.
	maxScopeSize := maxSize - (sz.DeltaSize(maxSize) - maxSize) - sz.ResourceMetricsSize(destRM)
	removedSize := 0
	srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
		// If there is no more capacity left just return, unless nothing in this resource can ever
		// fit a batch: then keep going so the data points are dropped instead of blocking the split.
		if capacityLeft == 0 && maxScopeSize > 0 {
			return false
		}
		rawSmSize := sz.ScopeMetricsSize(srcSM)
		smSize := sz.DeltaSize(rawSmSize)
		if smSize > capacityLeft {
			extSrcSM, extSmSize := extractScopeMetrics(srcSM, capacityLeft, maxScopeSize, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no metrics.
			// Do not add it to the destination if that is the case.
			if extSrcSM.Metrics().Len() > 0 {
				extSrcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
			}
			if srcSM.Metrics().Len() == 0 {
				// Nothing is left in the source scope, so all of it is removed.
				removedSize += smSize
				return true
			}
			// The source scope shrinks to the delta size of what is left in it.
			removedSize += smSize - sz.DeltaSize(rawSmSize-extSmSize)
			return false
		}
		capacityLeft -= smSize
		removedSize += smSize
		srcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
		return true
	})
	return destRM, removedSize
}

// extractScopeMetrics extracts scope metrics and returns a new scope metrics with the specified number of data points.
func extractScopeMetrics(srcSM pmetric.ScopeMetrics, capacity, maxScopeSize int, sz sizer.MetricsSizer) (pmetric.ScopeMetrics, int) {
	destSM := pmetric.NewScopeMetrics()
	destSM.SetSchemaUrl(srcSM.SchemaUrl())
	srcSM.Scope().CopyTo(destSM.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeMetricsSize(destSM)
	// Room for a metric in an otherwise empty batch, once the resource and scope headers and attributes are paid for.
	maxMetricSize := maxScopeSize - (sz.DeltaSize(maxScopeSize) - maxScopeSize) - sz.ScopeMetricsSize(destSM)
	removedSize := 0
	srcSM.Metrics().RemoveIf(func(srcMetric pmetric.Metric) bool {
		// If there is no more capacity left just return, unless nothing in this scope can ever
		// fit a batch: then keep going so the data points are dropped instead of blocking the split.
		if capacityLeft == 0 && maxMetricSize > 0 {
			return false
		}
		mSize := sz.DeltaSize(sz.MetricSize(srcMetric))
		if mSize > capacityLeft {
			extSrcMetric, _ := extractMetricDataPoints(srcMetric, capacityLeft, maxMetricSize, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no datapoints.
			// Do not add it to the destination if that is the case.
			if dataPointsLen(extSrcMetric) > 0 {
				extSrcMetric.MoveTo(destSM.Metrics().AppendEmpty())
			}
			if dataPointsLen(srcMetric) == 0 {
				// Nothing is left in the source metric, so all of it is removed.
				removedSize += mSize
				return true
			}
			// The source metric shrinks to the delta size of what is left in it. The data points sit
			// inside a nested data message (gauge, sum, ...) whose own length prefix can shrink too,
			// so the remaining size is measured rather than derived from the extracted data point sizes.
			removedSize += mSize - sz.DeltaSize(sz.MetricSize(srcMetric))
			return false
		}
		capacityLeft -= mSize
		removedSize += mSize
		srcMetric.MoveTo(destSM.Metrics().AppendEmpty())
		return true
	})
	return destSM, removedSize
}

// extractMetricDataPoints extracts data points and returns a new metric with the specified number of data points.
func extractMetricDataPoints(srcMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	destMetric := pmetric.NewMetric()
	destMetric.SetName(srcMetric.Name())
	destMetric.SetDescription(srcMetric.Description())
	destMetric.SetUnit(srcMetric.Unit())
	srcMetric.Metadata().CopyTo(destMetric.Metadata())

	var removedSize int
	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		removedSize = extractGaugeDataPoints(srcMetric.Gauge(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeSum:
		removedSize = extractSumDataPoints(srcMetric.Sum(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeHistogram:
		removedSize = extractHistogramDataPoints(srcMetric.Histogram(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeExponentialHistogram:
		removedSize = extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), destMetric, capacity, maxMetricSize, sz)
	case pmetric.MetricTypeSummary:
		removedSize = extractSummaryDataPoints(srcMetric.Summary(), destMetric, capacity, maxMetricSize, sz)
	}
	return destMetric, removedSize
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

// dataPointCapacity returns the capacity left for data points in destMetric and the largest
// data point that fits an otherwise empty batch, once the resource, scope and metric headers
// and attributes are paid for.
func dataPointCapacity(destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) (capacityLeft, maxDataPointSize int) {
	// The data points sit inside a nested data message (gauge, sum, ...) whose length prefix grows
	// with its content. destMetric only pays for that message while it is empty, so reserve the
	// rest of what the prefix can grow to for the largest content that fits.
	dataOverhead := (sz.DeltaSize(capacity) - capacity) - sz.DeltaSize(0)
	maxDataOverhead := (sz.DeltaSize(maxMetricSize) - maxMetricSize) - sz.DeltaSize(0)
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft = capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric) - dataOverhead
	maxDataPointSize = maxMetricSize - (sz.DeltaSize(maxMetricSize) - maxMetricSize) - sz.MetricSize(destMetric) - maxDataOverhead
	return capacityLeft, maxDataPointSize
}

func extractGaugeDataPoints(srcGauge pmetric.Gauge, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) int {
	destGauge := destMetric.SetEmptyGauge()
	capacityLeft, maxDataPointSize := dataPointCapacity(destMetric, capacity, maxMetricSize, sz)
	removedSize := 0

	srcGauge.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If there is no more capacity left just return, unless no data point can ever fit a batch:
		// then keep going so they are dropped instead of blocking the split.
		if capacityLeft == 0 && maxDataPointSize > 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			removedSize += rdSize
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destGauge.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractSumDataPoints(srcSum pmetric.Sum, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) int {
	destSum := destMetric.SetEmptySum()
	// Copy the fields that travel with every batch before measuring the header, so they are paid for.
	destSum.SetIsMonotonic(srcSum.IsMonotonic())
	destSum.SetAggregationTemporality(srcSum.AggregationTemporality())
	capacityLeft, maxDataPointSize := dataPointCapacity(destMetric, capacity, maxMetricSize, sz)
	removedSize := 0
	srcSum.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If there is no more capacity left just return, unless no data point can ever fit a batch:
		// then keep going so they are dropped instead of blocking the split.
		if capacityLeft == 0 && maxDataPointSize > 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			removedSize += rdSize
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destSum.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) int {
	destHistogram := destMetric.SetEmptyHistogram()
	// Copy the fields that travel with every batch before measuring the header, so they are paid for.
	destHistogram.SetAggregationTemporality(srcHistogram.AggregationTemporality())
	capacityLeft, maxDataPointSize := dataPointCapacity(destMetric, capacity, maxMetricSize, sz)
	removedSize := 0
	srcHistogram.DataPoints().RemoveIf(func(srcDP pmetric.HistogramDataPoint) bool {
		// If there is no more capacity left just return, unless no data point can ever fit a batch:
		// then keep going so they are dropped instead of blocking the split.
		if capacityLeft == 0 && maxDataPointSize > 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.HistogramDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			removedSize += rdSize
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destHistogram.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) int {
	destExponentialHistogram := destMetric.SetEmptyExponentialHistogram()
	// Copy the fields that travel with every batch before measuring the header, so they are paid for.
	destExponentialHistogram.SetAggregationTemporality(srcExponentialHistogram.AggregationTemporality())
	capacityLeft, maxDataPointSize := dataPointCapacity(destMetric, capacity, maxMetricSize, sz)
	removedSize := 0
	srcExponentialHistogram.DataPoints().RemoveIf(func(srcDP pmetric.ExponentialHistogramDataPoint) bool {
		// If there is no more capacity left just return, unless no data point can ever fit a batch:
		// then keep going so they are dropped instead of blocking the split.
		if capacityLeft == 0 && maxDataPointSize > 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.ExponentialHistogramDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			removedSize += rdSize
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destExponentialHistogram.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}

func extractSummaryDataPoints(srcSummary pmetric.Summary, destMetric pmetric.Metric, capacity, maxMetricSize int, sz sizer.MetricsSizer) int {
	destSummary := destMetric.SetEmptySummary()
	capacityLeft, maxDataPointSize := dataPointCapacity(destMetric, capacity, maxMetricSize, sz)
	removedSize := 0
	srcSummary.DataPoints().RemoveIf(func(srcDP pmetric.SummaryDataPoint) bool {
		// If there is no more capacity left just return, unless no data point can ever fit a batch:
		// then keep going so they are dropped instead of blocking the split.
		if capacityLeft == 0 && maxDataPointSize > 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.SummaryDataPointSize(srcDP))
		if rdSize > maxDataPointSize {
			// It can never be exported and would block every data point behind it, so drop it.
			removedSize += rdSize
			return true
		}
		if rdSize > capacityLeft {
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		removedSize += rdSize
		srcDP.MoveTo(destSummary.DataPoints().AppendEmpty())
		return true
	})
	return removedSize
}
