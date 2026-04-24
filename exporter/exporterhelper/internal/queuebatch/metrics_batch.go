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
func (req *metricsRequest) MergeSplit(_ context.Context, maxSizePerSizer map[request.SizerType]int64, r2 request.Request) ([]request.Request, error) {
	sizers := make(map[request.SizerType]sizer.MetricsSizer)
	for szt := range maxSizePerSizer {
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
	if len(maxSizePerSizer) == 0 {
		return []request.Request{req}, nil
	}
	return req.split(maxSizePerSizer, sizers)
}

func (req *metricsRequest) mergeTo(dst *metricsRequest, sizers map[request.SizerType]sizer.MetricsSizer) {
	for szt, sz := range sizers {
		dst.setCachedSize(szt, dst.size(szt, sz)+req.size(szt, sz))
		req.setCachedSize(szt, 0)
	}
	req.md.ResourceMetrics().MoveAndAppendTo(dst.md.ResourceMetrics())
}

func (req *metricsRequest) split(maxSizePerSizer map[request.SizerType]int64, sizers map[request.SizerType]sizer.MetricsSizer) ([]request.Request, error) {
	var res []request.Request
	sortedSzt := getSortedSizerTypes(maxSizePerSizer)
	for req.md.DataPointCount() > 0 {
		md := req.md
		isInitial := true
		exceededAny := false

		for _, szt := range sortedSzt {
			maxSize := maxSizePerSizer[szt]
			sz := sizers[szt]
			if maxSize > 0 && int64(sz.MetricsSize(md)) > maxSize {
				exceededAny = true
				mdNew := extractMetrics(md, int(maxSize), sz)
				if mdNew.DataPointCount() == 0 {
					return res, fmt.Errorf("one datapoint size is greater than max size, dropping items: %d", md.DataPointCount())
				}

				if isInitial {
					// md was req.md. Remainder is already in req.md due to in-place modification by extractMetrics.
					isInitial = false
				} else {
					// md was not req.md. Remainder is in md, and we must prepend it to req.md to maintain order.
					newReqMd := pmetric.NewMetrics()
					md.ResourceMetrics().MoveAndAppendTo(newReqMd.ResourceMetrics())
					req.md.ResourceMetrics().MoveAndAppendTo(newReqMd.ResourceMetrics())
					req.md = newReqMd
				}

				md = mdNew
			}
		}

		if !exceededAny {
			break
		}

		req.cachedSizes = make(map[request.SizerType]int)
		res = append(res, newMetricsRequest(md))
	}
	res = append(res, req)
	return res, nil
}

// extractMetrics extracts metrics from srcMetrics until capacity is reached.
func extractMetrics(srcMetrics pmetric.Metrics, capacity int, sz sizer.MetricsSizer) pmetric.Metrics {
	destMetrics := pmetric.NewMetrics()
	capacityLeft := capacity - sz.MetricsSize(destMetrics)

	srcMetrics.ResourceMetrics().RemoveIf(func(srcRM pmetric.ResourceMetrics) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRlSize := sz.ResourceMetricsSize(srcRM)
		rlSize := sz.DeltaSize(rawRlSize)

		if rlSize > capacityLeft {
			extSrcRM := extractResourceMetrics(srcRM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no scope metrics.
			// Do not add it to the destination if that is the case.
			if extSrcRM.ScopeMetrics().Len() > 0 {
				extSrcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
			}
			return extSrcRM.ScopeMetrics().Len() != 0
		}
		capacityLeft -= rlSize

		srcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
		return true
	})
	return destMetrics
}

// extractResourceMetrics extracts resource metrics and returns a new resource metrics with the specified number of data points.
func extractResourceMetrics(srcRM pmetric.ResourceMetrics, capacity int, sz sizer.MetricsSizer) pmetric.ResourceMetrics {
	destRM := pmetric.NewResourceMetrics()
	destRM.SetSchemaUrl(srcRM.SchemaUrl())
	srcRM.Resource().CopyTo(destRM.Resource())

	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ResourceMetricsSize(destRM)
	srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawSmSize := sz.ScopeMetricsSize(srcSM)
		smSize := sz.DeltaSize(rawSmSize)
		if smSize > capacityLeft {
			extSrcSM := extractScopeMetrics(srcSM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no metrics.
			// Do not add it to the destination if that is the case.
			if extSrcSM.Metrics().Len() > 0 {
				extSrcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
			}
			return extSrcSM.Metrics().Len() != 0
		}
		capacityLeft -= smSize
		srcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
		return true
	})
	return destRM
}

// extractScopeMetrics extracts scope metrics and returns a new scope metrics with the specified number of data points.
func extractScopeMetrics(srcSM pmetric.ScopeMetrics, capacity int, sz sizer.MetricsSizer) pmetric.ScopeMetrics {
	destSM := pmetric.NewScopeMetrics()
	destSM.SetSchemaUrl(srcSM.SchemaUrl())
	srcSM.Scope().CopyTo(destSM.Scope())

	// Take into account that this can have max "capacity", so when added to the parent will need space for the extra delta size.
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.ScopeMetricsSize(destSM)
	srcSM.Metrics().RemoveIf(func(srcM pmetric.Metric) bool {
		// If the no more capacity left just return.
		if capacityLeft == 0 {
			return false
		}
		rawRmSize := sz.MetricSize(srcM)
		rmSize := sz.DeltaSize(rawRmSize)
		if rmSize > capacityLeft {
			extSrcM := extractMetric(srcM, capacityLeft, sz)
			// This cannot make it to exactly 0 for the bytes,
			// force it to be 0 since that is the stopping condition.
			capacityLeft = 0
			// It is possible that for the bytes scenario, the extracted field contains no data points.
			// Do not add it to the destination if that is the case.
			if dataPointsLen(extSrcM) > 0 {
				extSrcM.MoveTo(destSM.Metrics().AppendEmpty())
			}
			return dataPointsLen(extSrcM) != 0
		}
		capacityLeft -= rmSize
		srcM.MoveTo(destSM.Metrics().AppendEmpty())
		return true
	})
	return destSM
}

// extractMetric extracts metric and returns a new metric with the specified number of data points.
func extractMetric(srcMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) pmetric.Metric {
	destMetric := pmetric.NewMetric()
	destMetric.SetName(srcMetric.Name())
	destMetric.SetDescription(srcMetric.Description())
	destMetric.SetUnit(srcMetric.Unit())
	srcMetric.Metadata().CopyTo(destMetric.Metadata())

	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		extractGaugeDataPoints(srcMetric.Gauge(), destMetric, capacity, sz)
	case pmetric.MetricTypeSum:
		extractSumDataPoints(srcMetric.Sum(), destMetric, capacity, sz)
		destMetric.Sum().SetIsMonotonic(srcMetric.Sum().IsMonotonic())
		destMetric.Sum().SetAggregationTemporality(srcMetric.Sum().AggregationTemporality())
	case pmetric.MetricTypeHistogram:
		extractHistogramDataPoints(srcMetric.Histogram(), destMetric, capacity, sz)
		destMetric.Histogram().SetAggregationTemporality(srcMetric.Histogram().AggregationTemporality())
	case pmetric.MetricTypeExponentialHistogram:
		extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), destMetric, capacity, sz)
		destMetric.ExponentialHistogram().SetAggregationTemporality(srcMetric.ExponentialHistogram().AggregationTemporality())
	case pmetric.MetricTypeSummary:
		extractSummaryDataPoints(srcMetric.Summary(), destMetric, capacity, sz)
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

func extractGaugeDataPoints(srcGauge pmetric.Gauge, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) {
	destGauge := destMetric.SetEmptyGauge()
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)

	srcGauge.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		if capacityLeft == 0 {
			return false
		}
		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > capacityLeft {
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destGauge.DataPoints().AppendEmpty())
		return true
	})
}

func extractSumDataPoints(srcSum pmetric.Sum, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) {
	destSum := destMetric.SetEmptySum()
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)

	srcSum.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		if capacityLeft == 0 {
			return false
		}
		rdSize := sz.DeltaSize(sz.NumberDataPointSize(srcDP))
		if rdSize > capacityLeft {
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destSum.DataPoints().AppendEmpty())
		return true
	})
}

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) {
	destHistogram := destMetric.SetEmptyHistogram()
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)

	srcHistogram.DataPoints().RemoveIf(func(srcDP pmetric.HistogramDataPoint) bool {
		if capacityLeft == 0 {
			return false
		}
		rdSize := sz.DeltaSize(sz.HistogramDataPointSize(srcDP))
		if rdSize > capacityLeft {
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destHistogram.DataPoints().AppendEmpty())
		return true
	})
}

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) {
	destExponentialHistogram := destMetric.SetEmptyExponentialHistogram()
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)

	srcExponentialHistogram.DataPoints().RemoveIf(func(srcDP pmetric.ExponentialHistogramDataPoint) bool {
		if capacityLeft == 0 {
			return false
		}
		rdSize := sz.DeltaSize(sz.ExponentialHistogramDataPointSize(srcDP))
		if rdSize > capacityLeft {
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destExponentialHistogram.DataPoints().AppendEmpty())
		return true
	})
}

func extractSummaryDataPoints(srcSummary pmetric.Summary, destMetric pmetric.Metric, capacity int, sz sizer.MetricsSizer) {
	destSummary := destMetric.SetEmptySummary()
	capacityLeft := capacity - (sz.DeltaSize(capacity) - capacity) - sz.MetricSize(destMetric)

	srcSummary.DataPoints().RemoveIf(func(srcDP pmetric.SummaryDataPoint) bool {
		if capacityLeft == 0 {
			return false
		}
		rdSize := sz.DeltaSize(sz.SummaryDataPointSize(srcDP))
		if rdSize > capacityLeft {
			capacityLeft = 0
			return false
		}
		capacityLeft -= rdSize
		srcDP.MoveTo(destSummary.DataPoints().AppendEmpty())
		return true
	})
}
