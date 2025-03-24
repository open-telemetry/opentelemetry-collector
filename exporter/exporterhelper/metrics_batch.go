// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MergeSplit splits and/or merges the provided metrics request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *metricsRequest) MergeSplit(_ context.Context, cfg exporterbatcher.SizeConfig, r2 Request) ([]Request, error) {
	var sz sizer.MetricsSizer
	switch cfg.Sizer {
	case exporterbatcher.SizerTypeItems:
		sz = &sizer.MetricsCountSizer{}
	case exporterbatcher.SizerTypeBytes:
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
	if cfg.MaxSize == 0 {
		return []Request{req}, nil
	}
	return req.split(cfg.MaxSize, sz), nil
}

func (req *metricsRequest) mergeTo(dst *metricsRequest, sz sizer.MetricsSizer) {
	if sz != nil {
		dst.setCachedSize(dst.size(sz) + req.size(sz))
		req.setCachedSize(0)
	}
	req.md.ResourceMetrics().MoveAndAppendTo(dst.md.ResourceMetrics())
}

func (req *metricsRequest) split(maxSize int, sz sizer.MetricsSizer) []Request {
	var res []Request
	for req.size(sz) > maxSize {
		md, rmSize := extractMetrics(req.md, maxSize, sz)
		req.setCachedSize(req.size(sz) - rmSize)
		res = append(res, newMetricsRequest(md))
	}
	res = append(res, req)
	return res
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
			extSrcSM, extRmSize := extractMetricDataPoints(srcSM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			removedSize += extRmSize
			// There represents the delta between the delta sizes.
			removedSize += rmSize - rawRmSize - (sz.DeltaSize(rawRmSize-extRmSize) - (rawRmSize - extRmSize))
			// It is possible that for the bytes scenario, the extracted field contains no datapoints.
			// Do not add it to the destination if that is the case.
			if sz.MetricSize(extSrcSM) > 0 {
				extSrcSM.MoveTo(destSM.Metrics().AppendEmpty())
			}
			return sz.MetricSize(extSrcSM) != 0
		}
		capacityLeft -= rmSize
		removedSize += rmSize
		srcSM.MoveTo(destSM.Metrics().AppendEmpty())
		return true
	})
	return destSM, removedSize
}

func extractMetricDataPoints(srcMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	var destMetric pmetric.Metric
	var removedSize int
	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		destMetric, removedSize = extractGaugeDataPoints(srcMetric.Gauge(), capacity, sz)
	case pmetric.MetricTypeSum:
		destMetric, removedSize = extractSumDataPoints(srcMetric.Sum(), capacity, sz)
	case pmetric.MetricTypeHistogram:
		destMetric, removedSize = extractHistogramDataPoints(srcMetric.Histogram(), capacity, sz)
	case pmetric.MetricTypeExponentialHistogram:
		destMetric, removedSize = extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), capacity, sz)
	case pmetric.MetricTypeSummary:
		destMetric, removedSize = extractSummaryDataPoints(srcMetric.Summary(), capacity, sz)
	}
	return destMetric, removedSize
}

func extractGaugeDataPoints(srcGauge pmetric.Gauge, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	m := pmetric.NewMetric()
	destGauge := m.SetEmptyGauge()

	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(m)
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
	return m, removedSize
}

func extractSumDataPoints(srcSum pmetric.Sum, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	m := pmetric.NewMetric()
	destSum := m.SetEmptySum()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(m)
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
	return m, removedSize
}

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	m := pmetric.NewMetric()
	destHistogram := m.SetEmptyHistogram()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(m)
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
	return m, removedSize
}

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	m := pmetric.NewMetric()
	destExponentialHistogram := m.SetEmptyExponentialHistogram()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(m)
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
	return m, removedSize
}

func extractSummaryDataPoints(srcSummary pmetric.Summary, capacity int, sz sizer.MetricsSizer) (pmetric.Metric, int) {
	m := pmetric.NewMetric()
	destSummary := m.SetEmptySummary()
	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(m)
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
	return m, removedSize
}
