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
		req2.mergeTo(req, sz)
	}

	// If no limit we can simply merge the new request into the current and return.
	if maxSize == 0 {
		return []request.Request{req}, nil
	}
	return req.split(maxSize, sz)
}

func (req *metricsRequest) mergeTo(dst *metricsRequest, sz sizer.MetricsSizer) {
	if sz != nil {
		dst.setCachedSize(dst.size(sz) + req.size(sz))
		req.setCachedSize(0)
	}
	req.md.ResourceMetrics().MoveAndAppendTo(dst.md.ResourceMetrics())
}

func (req *metricsRequest) split(maxSize int, sz sizer.MetricsSizer) ([]request.Request, error) {
	var res []request.Request
	for req.size(sz) > maxSize {
		md, rmSize := extractMetrics(req.md, maxSize, sz)
		if md.DataPointCount() == 0 {
			return res, fmt.Errorf("one datapoint size is greater than max size, dropping items: %d", req.md.DataPointCount())
		}
		req.setCachedSize(req.size(sz) - rmSize)
		res = append(res, newMetricsRequest(md))
	}
	res = append(res, req)
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
		rawRlSize := sz.ResourceMetricsSize(srcRM)
		rlSize := sz.DeltaSize(rawRlSize)
		if rlSize > capacityLeft {
			extSrcRM, extRmSize := extractResourceMetrics(srcRM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRmSize
			// There represents the delta between the delta sizes.
			removedSize += rlSize - rawRlSize - (sz.DeltaSize(rawRlSize-extRmSize) - (rawRlSize - extRmSize))
			// It is possible that for the bytes scenario, the extracted field contains no scope metrics.
			// Do not add it to the destination if that is the case.
			if extSrcRM.ScopeMetrics().Len() > 0 {
				extSrcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
			}
			return extSrcRM.ScopeMetrics().Len() != 0
		}
		capacityLeft -= rlSize
		removedSize += rlSize
		srcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
		return true
	})
	return destMetrics, removedSize
}

// extractResourceMetrics extracts resource metrics and returns a new resource metrics with the specified number of data points.
func extractResourceMetrics(srcRM pmetric.ResourceMetrics, capacity int, sz sizer.MetricsSizer) (pmetric.ResourceMetrics, int) {
	destRM := pmetric.NewResourceMetrics()
	destRM.SetSchemaUrl(srcRM.SchemaUrl())
	srcRM.Resource().CopyTo(destRM.Resource())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceMetricsSize(destRM)
	removedSize := 0
	srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawSmSize := sz.ScopeMetricsSize(srcSM)
		smSize := sz.DeltaSize(rawSmSize)
		if smSize > capacityLeft {
			extSrcSM, extSmSize := extractScopeMetrics(srcSM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extSmSize
			// There represents the delta between the delta sizes.
			removedSize += smSize - rawSmSize - (sz.DeltaSize(rawSmSize-extSmSize) - (rawSmSize - extSmSize))
			// It is possible that for the bytes scenario, the extracted field contains no scope metrics.
			// Do not add it to the destination if that is the case.
			if extSrcSM.Metrics().Len() > 0 {
				extSrcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
			}
			return extSrcSM.Metrics().Len() != 0
		}
		capacityLeft -= smSize
		removedSize += smSize
		srcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
		return true
	})
	return destRM, removedSize
}

// extractScopeMetrics extracts scope metrics and returns a new scope metrics with the specified number of data points.
func extractScopeMetrics(srcSM pmetric.ScopeMetrics, capacity int, sz sizer.MetricsSizer) (pmetric.ScopeMetrics, int) {
	destSM := pmetric.NewScopeMetrics()
	destSM.SetSchemaUrl(srcSM.SchemaUrl())
	srcSM.Scope().CopyTo(destSM.Scope())
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeMetricsSize(destSM)
	removedSize := 0
	srcSM.Metrics().RemoveIf(func(srcSM pmetric.Metric) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRmSize := sz.MetricSize(srcSM)
		rmSize := sz.DeltaSize(rawRmSize)
		if rmSize > capacityLeft {
			extSrcDP, extRmSize := extractMetricDataPoints(srcSM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRmSize
			// There represents the delta between the delta sizes.
			removedSize += rmSize - rawRmSize - (sz.DeltaSize(rawRmSize-extRmSize) - (rawRmSize - extRmSize))
			// It is possible that for the bytes scenario, the extracted field contains no datapoints.
			// Do not add it to the destination if that is the case.
			if dataPointsLen(extSrcDP) > 0 {
				extSrcDP.MoveTo(destSM.Metrics().AppendEmpty())
			}
			return dataPointsLen(extSrcDP) != 0
		}
		capacityLeft -= rmSize
		removedSize += rmSize
		srcSM.MoveTo(destSM.Metrics().AppendEmpty())
		return true
	})
	return destSM, removedSize
}

func extractMetricDataPoints(srcMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	destMetric := pmetric.NewMetric()
	destMetric.SetName(srcMetric.Name())
	destMetric.SetDescription(srcMetric.Description())
	destMetric.SetUnit(srcMetric.Unit())
	srcMetric.Metadata().CopyTo(destMetric.Metadata())

	var removedSize int
	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		removedSize = extractGaugeDataPoints(srcMetric.Gauge(), destMetric, capacity, sz)
	case pmetric.MetricTypeSum:
		removedSize = extractSumDataPoints(srcMetric.Sum(), destMetric, capacity, sz)
		destMetric.Sum().SetIsMonotonic(srcMetric.Sum().IsMonotonic())
		destMetric.Sum().SetAggregationTemporality(srcMetric.Sum().AggregationTemporality())
	case pmetric.MetricTypeHistogram:
		removedSize = extractHistogramDataPoints(srcMetric.Histogram(), destMetric, capacity, sz)
		destMetric.Histogram().SetAggregationTemporality(srcMetric.Histogram().AggregationTemporality())
	case pmetric.MetricTypeExponentialHistogram:
		removedSize = extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), destMetric, capacity, sz)
		destMetric.ExponentialHistogram().SetAggregationTemporality(srcMetric.ExponentialHistogram().AggregationTemporality())
	case pmetric.MetricTypeSummary:
		removedSize = extractSummaryDataPoints(srcMetric.Summary(), destMetric, capacity, sz)
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

func extractGaugeDataPoints(srcGauge pmetric.Gauge, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destGauge := destMetric.SetEmptyGauge()

	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0

	srcGauge.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
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

func extractSumDataPoints(srcSum pmetric.Sum, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destSum := destMetric.SetEmptySum()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcSum.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
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

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destHistogram := destMetric.SetEmptyHistogram()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcHistogram.DataPoints().RemoveIf(func(srcDP pmetric.HistogramDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.HistogramDataPointSize(srcDP))
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

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destExponentialHistogram := destMetric.SetEmptyExponentialHistogram()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcExponentialHistogram.DataPoints().RemoveIf(func(srcDP pmetric.ExponentialHistogramDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.ExponentialHistogramDataPointSize(srcDP))
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

func extractSummaryDataPoints(srcSummary pmetric.Summary, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) int {
	destSummary := destMetric.SetEmptySummary()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)
	removedSize := 0
	srcSummary.DataPoints().RemoveIf(func(srcDP pmetric.SummaryDataPoint) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}

		rdSize := sz.DeltaSize(sz.SummaryDataPointSize(srcDP))
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
