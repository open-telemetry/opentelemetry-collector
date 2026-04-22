// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MergeSplit splits and/or merges the provided metrics request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *metricsRequest) MergeSplit(_ context.Context, limits map[request.SizerType]int64, r2 request.Request) ([]request.Request, error) {
	sizers := make(map[request.SizerType]sizer.MetricsSizer)
	for szt := range limits {
		switch szt {
		case request.SizerTypeItems:
			sizers[szt] = &sizer.MetricsCountSizer{}
		case request.SizerTypeBytes:
			sizers[szt] = &sizer.MetricsBytesSizer{}
		default:
			return nil, errors.New("unknown sizer type")
		}
	}

	if r2 != nil {
		req2, ok := r2.(*metricsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req, sizers)
	}

	// If no limits we can simply merge the new request into the current and return.
	if len(limits) == 0 {
		return []request.Request{req}, nil
	}
	return req.split(limits, sizers)
}

func (req *metricsRequest) mergeTo(dst *metricsRequest, sizers map[request.SizerType]sizer.MetricsSizer) {
	for szt, sz := range sizers {
		dst.setCachedSize(szt, dst.size(szt, sz)+req.size(szt, sz))
		req.setCachedSize(szt, 0)
	}
	req.md.ResourceMetrics().MoveAndAppendTo(dst.md.ResourceMetrics())
}

func (req *metricsRequest) split(limits map[request.SizerType]int64, sizers map[request.SizerType]sizer.MetricsSizer) ([]request.Request, error) {
	var res []request.Request
	for {
		exceeded := false
		for szt, limit := range limits {
			sz := sizers[szt]
			if limit > 0 && int64(req.size(szt, sz)) > limit {
				exceeded = true
				break
			}
		}
		if !exceeded {
			break
		}

		adjustedLimits := make(map[request.SizerType]int64)
		for szt, limit := range limits {
			if limit == 0 {
				adjustedLimits[szt] = math.MaxInt64
			} else {
				adjustedLimits[szt] = limit
			}
		}
		md, rmSizes := extractMetrics(req.md, adjustedLimits, sizers)
		if md.DataPointCount() == 0 {
			return res, fmt.Errorf("one datapoint size is greater than max size, dropping items: %d", req.md.DataPointCount())
		}
		for szt, sz := range sizers {
			req.setCachedSize(szt, req.size(szt, sz)-rmSizes[szt])
		}
		res = append(res, newMetricsRequest(md))
	}
	res = append(res, req)
	return res, nil
}

// extractMetrics extracts metrics from srcMetrics until capacity is reached.
func extractMetrics(srcMetrics pmetric.Metrics, limits map[request.SizerType]int64, sizers map[request.SizerType]sizer.MetricsSizer) (pmetric.Metrics, map[request.SizerType]int) {
	destMetrics := pmetric.NewMetrics()
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = int(limit) - sz.MetricsSize(destMetrics)
		removedSizes[szt] = 0
	}

	srcMetrics.ResourceMetrics().RemoveIf(func(srcRM pmetric.ResourceMetrics) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rawRlSize := sz.ResourceMetricsSize(srcRM)
			rlSize := sz.DeltaSize(rawRlSize)
			if rlSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			extSrcRM, extRmSizes := extractResourceMetrics(srcRM, capacityLeft, sizers)
			
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			
			for szt, sz := range sizers {
				rawRlSize := sz.ResourceMetricsSize(srcRM)
				rlSize := sz.DeltaSize(rawRlSize)
				extRmSize := extRmSizes[szt]
				removedSizes[szt] += extRmSize
				removedSizes[szt] += rlSize - rawRlSize - (sz.DeltaSize(rawRlSize-extRmSize) - (rawRlSize - extRmSize))
			}

			if extSrcRM.ScopeMetrics().Len() > 0 {
				extSrcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
			}
			return extSrcRM.ScopeMetrics().Len() != 0
		}

		for szt, sz := range sizers {
			rawRlSize := sz.ResourceMetricsSize(srcRM)
			rlSize := sz.DeltaSize(rawRlSize)
			capacityLeft[szt] -= rlSize
			removedSizes[szt] += rlSize
		}
		srcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
		return true
	})
	return destMetrics, removedSizes
}

// extractResourceMetrics extracts resource metrics and returns a new resource metrics with the specified number of data points.
func extractResourceMetrics(srcRM pmetric.ResourceMetrics, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) (pmetric.ResourceMetrics, map[request.SizerType]int) {
	destRM := pmetric.NewResourceMetrics()
	destRM.SetSchemaUrl(srcRM.SchemaUrl())
	srcRM.Resource().CopyTo(destRM.Resource())
	
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)
	
	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.ResourceMetricsSize(destRM)
		removedSizes[szt] = 0
	}

	srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rawSmSize := sz.ScopeMetricsSize(srcSM)
			smSize := sz.DeltaSize(rawSmSize)
			if smSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			extSrcSM, extSmSizes := extractScopeMetrics(srcSM, capacityLeft, sizers)
			
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			
			for szt, sz := range sizers {
				rawSmSize := sz.ScopeMetricsSize(srcSM)
				smSize := sz.DeltaSize(rawSmSize)
				extSmSize := extSmSizes[szt]
				removedSizes[szt] += extSmSize
				removedSizes[szt] += smSize - rawSmSize - (sz.DeltaSize(rawSmSize-extSmSize) - (rawSmSize - extSmSize))
			}

			if extSrcSM.Metrics().Len() > 0 {
				extSrcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
			}
			return extSrcSM.Metrics().Len() != 0
		}

		for szt, sz := range sizers {
			rawSmSize := sz.ScopeMetricsSize(srcSM)
			smSize := sz.DeltaSize(rawSmSize)
			capacityLeft[szt] -= smSize
			removedSizes[szt] += smSize
		}
		srcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
		return true
	})
	return destRM, removedSizes
}

// extractScopeMetrics extracts scope metrics and returns a new scope metrics with the specified number of data points.
func extractScopeMetrics(srcSM pmetric.ScopeMetrics, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) (pmetric.ScopeMetrics, map[request.SizerType]int) {
	destSM := pmetric.NewScopeMetrics()
	destSM.SetSchemaUrl(srcSM.SchemaUrl())
	srcSM.Scope().CopyTo(destSM.Scope())
	
	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)
	
	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.ScopeMetricsSize(destSM)
		removedSizes[szt] = 0
	}

	srcSM.Metrics().RemoveIf(func(srcSM pmetric.Metric) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rawRmSize := sz.MetricSize(srcSM)
			rmSize := sz.DeltaSize(rawRmSize)
			if rmSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			extSrcDP, extRmSizes := extractMetricDataPoints(srcSM, capacityLeft, sizers)
			
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			
			for szt, sz := range sizers {
				rawRmSize := sz.MetricSize(srcSM)
				rmSize := sz.DeltaSize(rawRmSize)
				extRmSize := extRmSizes[szt]
				removedSizes[szt] += extRmSize
				removedSizes[szt] += rmSize - rawRmSize - (sz.DeltaSize(rawRmSize-extRmSize) - (rawRmSize - extRmSize))
			}

			if dataPointsLen(extSrcDP) > 0 {
				extSrcDP.MoveTo(destSM.Metrics().AppendEmpty())
			}
			return dataPointsLen(extSrcDP) != 0
		}

		for szt, sz := range sizers {
			rawRmSize := sz.MetricSize(srcSM)
			rmSize := sz.DeltaSize(rawRmSize)
			capacityLeft[szt] -= rmSize
			removedSizes[szt] += rmSize
		}
		srcSM.MoveTo(destSM.Metrics().AppendEmpty())
		return true
	})
	return destSM, removedSizes
}

func extractMetricDataPoints(srcMetric pmetric.Metric, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) (pmetric.Metric, map[request.SizerType]int) {
	destMetric := pmetric.NewMetric()
	destMetric.SetName(srcMetric.Name())
	destMetric.SetDescription(srcMetric.Description())
	destMetric.SetUnit(srcMetric.Unit())
	srcMetric.Metadata().CopyTo(destMetric.Metadata())

	var removedSizes map[request.SizerType]int
	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		removedSizes = extractGaugeDataPoints(srcMetric.Gauge(), destMetric, limits, sizers)
	case pmetric.MetricTypeSum:
		removedSizes = extractSumDataPoints(srcMetric.Sum(), destMetric, limits, sizers)
		destMetric.Sum().SetIsMonotonic(srcMetric.Sum().IsMonotonic())
		destMetric.Sum().SetAggregationTemporality(srcMetric.Sum().AggregationTemporality())
	case pmetric.MetricTypeHistogram:
		removedSizes = extractHistogramDataPoints(srcMetric.Histogram(), destMetric, limits, sizers)
		destMetric.Histogram().SetAggregationTemporality(srcMetric.Histogram().AggregationTemporality())
	case pmetric.MetricTypeExponentialHistogram:
		removedSizes = extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), destMetric, limits, sizers)
		destMetric.ExponentialHistogram().SetAggregationTemporality(srcMetric.ExponentialHistogram().AggregationTemporality())
	case pmetric.MetricTypeSummary:
		removedSizes = extractSummaryDataPoints(srcMetric.Summary(), destMetric, limits, sizers)
	}
	return destMetric, removedSizes
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

func extractGaugeDataPoints(srcGauge pmetric.Gauge, destMetric pmetric.Metric, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) map[request.SizerType]int {
	destGauge := destMetric.SetEmptyGauge()

	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.MetricSize(destMetric)
		removedSizes[szt] = 0
	}

	srcGauge.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
			if rdSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			return false
		}

		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
			capacityLeft[szt] -= rdSize
			removedSizes[szt] += rdSize
		}
		srcDP.MoveTo(destGauge.DataPoints().AppendEmpty())
		return true
	})
	return removedSizes
}

func extractSumDataPoints(srcSum pmetric.Sum, destMetric pmetric.Metric, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) map[request.SizerType]int {
	destSum := destMetric.SetEmptySum()

	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.MetricSize(destMetric)
		removedSizes[szt] = 0
	}

	srcSum.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
			if rdSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			return false
		}

		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
			capacityLeft[szt] -= rdSize
			removedSizes[szt] += rdSize
		}
		srcDP.MoveTo(destSum.DataPoints().AppendEmpty())
		return true
	})
	return removedSizes
}

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, destMetric pmetric.Metric, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) map[request.SizerType]int {
	destHistogram := destMetric.SetEmptyHistogram()

	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.MetricSize(destMetric)
		removedSizes[szt] = 0
	}

	srcHistogram.DataPoints().RemoveIf(func(srcDP pmetric.HistogramDataPoint) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.HistogramDataPointSize(srcDP))
			if rdSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			return false
		}

		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.HistogramDataPointSize(srcDP))
			capacityLeft[szt] -= rdSize
			removedSizes[szt] += rdSize
		}
		srcDP.MoveTo(destHistogram.DataPoints().AppendEmpty())
		return true
	})
	return removedSizes
}

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, destMetric pmetric.Metric, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) map[request.SizerType]int {
	destExponentialHistogram := destMetric.SetEmptyExponentialHistogram()

	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.MetricSize(destMetric)
		removedSizes[szt] = 0
	}

	srcExponentialHistogram.DataPoints().RemoveIf(func(srcDP pmetric.ExponentialHistogramDataPoint) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.ExponentialHistogramDataPointSize(srcDP))
			if rdSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			return false
		}

		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.ExponentialHistogramDataPointSize(srcDP))
			capacityLeft[szt] -= rdSize
			removedSizes[szt] += rdSize
		}
		srcDP.MoveTo(destExponentialHistogram.DataPoints().AppendEmpty())
		return true
	})
	return removedSizes
}

func extractSummaryDataPoints(srcSummary pmetric.Summary, destMetric pmetric.Metric, limits map[request.SizerType]int, sizers map[request.SizerType]sizer.MetricsSizer) map[request.SizerType]int {
	destSummary := destMetric.SetEmptySummary()

	capacityLeft := make(map[request.SizerType]int)
	removedSizes := make(map[request.SizerType]int)

	for szt, limit := range limits {
		sz := sizers[szt]
		capacityLeft[szt] = limit - (sz.DeltaSize(limit) - limit) - sz.MetricSize(destMetric)
		removedSizes[szt] = 0
	}

	srcSummary.DataPoints().RemoveIf(func(srcDP pmetric.SummaryDataPoint) bool {
		for _, cap := range capacityLeft {
			if cap <= 0 {
				return false
			}
		}

		fitsAll := true
		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.SummaryDataPointSize(srcDP))
			if rdSize > capacityLeft[szt] {
				fitsAll = false
				break
			}
		}

		if !fitsAll {
			for szt := range sizers {
				capacityLeft[szt] = 0
			}
			return false
		}

		for szt, sz := range sizers {
			rdSize := sz.DeltaSize(sz.SummaryDataPointSize(srcDP))
			capacityLeft[szt] -= rdSize
			removedSizes[szt] += rdSize
		}
		srcDP.MoveTo(destSummary.DataPoints().AppendEmpty())
		return true
	})
	return removedSizes
}
