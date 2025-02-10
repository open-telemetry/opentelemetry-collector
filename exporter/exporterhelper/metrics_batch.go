// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MergeSplit splits and/or merges the provided metrics request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *metricsRequest) MergeSplit(_ context.Context, cfg exporterbatcher.MaxSizeConfig, r2 Request) ([]Request, error) {
	if r2 != nil {
		req2, ok := r2.(*metricsRequest)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		req2.mergeTo(req)
	}

	// If no limit we can simply merge the new request into the current and return.
	if cfg.MaxSizeItems == 0 {
		return []Request{req}, nil
	}
	return req.split(cfg)
}

func (req *metricsRequest) mergeTo(dst *metricsRequest) {
	dst.setCachedItemsCount(dst.ItemsCount() + req.ItemsCount())
	req.setCachedItemsCount(0)
	req.md.ResourceMetrics().MoveAndAppendTo(dst.md.ResourceMetrics())
}

func (req *metricsRequest) split(cfg exporterbatcher.MaxSizeConfig) ([]Request, error) {
	var res []Request
	for req.ItemsCount() > cfg.MaxSizeItems {
		md := extractMetrics(req.md, cfg.MaxSizeItems)
		size := md.DataPointCount()
		req.setCachedItemsCount(req.ItemsCount() - size)
		res = append(res, &metricsRequest{md: md, pusher: req.pusher, cachedItemsCount: size})
	}
	res = append(res, req)
	return res, nil
}

// extractMetrics extracts metrics from srcMetrics until count of data points is reached.
func extractMetrics(srcMetrics pmetric.Metrics, count int) pmetric.Metrics {
	destMetrics := pmetric.NewMetrics()
	srcMetrics.ResourceMetrics().RemoveIf(func(srcRM pmetric.ResourceMetrics) bool {
		if count == 0 {
			return false
		}
		needToExtract := resourceDataPointsCount(srcRM) > count
		if needToExtract {
			srcRM = extractResourceMetrics(srcRM, count)
		}
		count -= resourceDataPointsCount(srcRM)
		srcRM.MoveTo(destMetrics.ResourceMetrics().AppendEmpty())
		return !needToExtract
	})
	return destMetrics
}

// extractResourceMetrics extracts resource metrics and returns a new resource metrics with the specified number of data points.
func extractResourceMetrics(srcRM pmetric.ResourceMetrics, count int) pmetric.ResourceMetrics {
	destRM := pmetric.NewResourceMetrics()
	destRM.SetSchemaUrl(srcRM.SchemaUrl())
	srcRM.Resource().CopyTo(destRM.Resource())
	srcRM.ScopeMetrics().RemoveIf(func(srcSM pmetric.ScopeMetrics) bool {
		if count == 0 {
			return false
		}
		needToExtract := scopeDataPointsCount(srcSM) > count
		if needToExtract {
			srcSM = extractScopeMetrics(srcSM, count)
		}
		count -= scopeDataPointsCount(srcSM)
		srcSM.MoveTo(destRM.ScopeMetrics().AppendEmpty())
		return !needToExtract
	})
	return destRM
}

// extractScopeMetrics extracts scope metrics and returns a new scope metrics with the specified number of data points.
func extractScopeMetrics(srcSM pmetric.ScopeMetrics, count int) pmetric.ScopeMetrics {
	destSM := pmetric.NewScopeMetrics()
	destSM.SetSchemaUrl(srcSM.SchemaUrl())
	srcSM.Scope().CopyTo(destSM.Scope())
	srcSM.Metrics().RemoveIf(func(srcMetric pmetric.Metric) bool {
		if count == 0 {
			return false
		}
		needToExtract := metricDataPointCount(srcMetric) > count
		if needToExtract {
			srcMetric = extractMetricDataPoints(srcMetric, count)
		}
		count -= metricDataPointCount(srcMetric)
		srcMetric.MoveTo(destSM.Metrics().AppendEmpty())
		return !needToExtract
	})
	return destSM
}

func extractMetricDataPoints(srcMetric pmetric.Metric, count int) pmetric.Metric {
	destMetric := pmetric.NewMetric()
	switch srcMetric.Type() {
	case pmetric.MetricTypeGauge:
		extractGaugeDataPoints(srcMetric.Gauge(), count, destMetric.SetEmptyGauge())
	case pmetric.MetricTypeSum:
		extractSumDataPoints(srcMetric.Sum(), count, destMetric.SetEmptySum())
	case pmetric.MetricTypeHistogram:
		extractHistogramDataPoints(srcMetric.Histogram(), count, destMetric.SetEmptyHistogram())
	case pmetric.MetricTypeExponentialHistogram:
		extractExponentialHistogramDataPoints(srcMetric.ExponentialHistogram(), count,
			destMetric.SetEmptyExponentialHistogram())
	case pmetric.MetricTypeSummary:
		extractSummaryDataPoints(srcMetric.Summary(), count, destMetric.SetEmptySummary())
	}
	return destMetric
}

func extractGaugeDataPoints(srcGauge pmetric.Gauge, count int, destGauge pmetric.Gauge) {
	srcGauge.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		if count == 0 {
			return false
		}
		srcDP.MoveTo(destGauge.DataPoints().AppendEmpty())
		count--
		return true
	})
}

func extractSumDataPoints(srcSum pmetric.Sum, count int, destSum pmetric.Sum) {
	srcSum.DataPoints().RemoveIf(func(srcDP pmetric.NumberDataPoint) bool {
		if count == 0 {
			return false
		}
		srcDP.MoveTo(destSum.DataPoints().AppendEmpty())
		count--
		return true
	})
}

func extractHistogramDataPoints(srcHistogram pmetric.Histogram, count int, destHistogram pmetric.Histogram) {
	srcHistogram.DataPoints().RemoveIf(func(srcDP pmetric.HistogramDataPoint) bool {
		if count == 0 {
			return false
		}
		srcDP.MoveTo(destHistogram.DataPoints().AppendEmpty())
		count--
		return true
	})
}

func extractExponentialHistogramDataPoints(srcExponentialHistogram pmetric.ExponentialHistogram, count int, destExponentialHistogram pmetric.ExponentialHistogram) {
	srcExponentialHistogram.DataPoints().RemoveIf(func(srcDP pmetric.ExponentialHistogramDataPoint) bool {
		if count == 0 {
			return false
		}
		srcDP.MoveTo(destExponentialHistogram.DataPoints().AppendEmpty())
		count--
		return true
	})
}

func extractSummaryDataPoints(srcSummary pmetric.Summary, count int, destSummary pmetric.Summary) {
	srcSummary.DataPoints().RemoveIf(func(srcDP pmetric.SummaryDataPoint) bool {
		if count == 0 {
			return false
		}
		srcDP.MoveTo(destSummary.DataPoints().AppendEmpty())
		count--
		return true
	})
}

func resourceDataPointsCount(rm pmetric.ResourceMetrics) (count int) {
	for i := 0; i < rm.ScopeMetrics().Len(); i++ {
		count += scopeDataPointsCount(rm.ScopeMetrics().At(i))
	}
	return count
}

func scopeDataPointsCount(sm pmetric.ScopeMetrics) (count int) {
	for i := 0; i < sm.Metrics().Len(); i++ {
		count += metricDataPointCount(sm.Metrics().At(i))
	}
	return count
}

func metricDataPointCount(m pmetric.Metric) int {
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
