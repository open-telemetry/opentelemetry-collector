// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import "unicode/utf8"

// ValidateUTF8ExportMetricsServiceRequest returns false when any string in req contains invalid UTF-8.
func ValidateUTF8ExportMetricsServiceRequest(req *ExportMetricsServiceRequest) bool {
	for _, rm := range req.ResourceMetrics {
		if !validUTF8Resource(&rm.Resource) || !utf8.ValidString(rm.SchemaUrl) ||
			!validUTF8ScopeMetricsSlice(rm.ScopeMetrics) || !validUTF8ScopeMetricsSlice(rm.DeprecatedScopeMetrics) {
			return false
		}
	}
	return true
}

func validUTF8ScopeMetricsSlice(scopes []*ScopeMetrics) bool {
	for _, sm := range scopes {
		if !validUTF8InstrumentationScope(&sm.Scope) || !utf8.ValidString(sm.SchemaUrl) {
			return false
		}
		for _, m := range sm.Metrics {
			if !validUTF8MetricHeader(m) || !validUTF8MetricDataPoints(m) {
				return false
			}
		}
	}
	return true
}

func validUTF8MetricHeader(m *Metric) bool {
	return utf8.ValidString(m.Name) && utf8.ValidString(m.Description) &&
		utf8.ValidString(m.Unit) && validUTF8KeyValueSlice(m.Metadata)
}

func validUTF8NumberDataPoint(dp *NumberDataPoint) bool {
	return validUTF8KeyValueSlice(dp.Attributes) && validUTF8Exemplars(dp.Exemplars)
}

func validUTF8HistogramDataPoint(dp *HistogramDataPoint) bool {
	return validUTF8KeyValueSlice(dp.Attributes) && validUTF8Exemplars(dp.Exemplars)
}

func validUTF8ExponentialHistogramDataPoint(dp *ExponentialHistogramDataPoint) bool {
	return validUTF8KeyValueSlice(dp.Attributes) && validUTF8Exemplars(dp.Exemplars)
}

func validUTF8SummaryDataPoint(dp *SummaryDataPoint) bool {
	return validUTF8KeyValueSlice(dp.Attributes)
}

func validUTF8DataPoints[T any](s []*T, valid func(*T) bool) bool {
	for _, dp := range s {
		if !valid(dp) {
			return false
		}
	}
	return true
}

func validUTF8MetricDataPoints(m *Metric) bool {
	switch d := m.Data.(type) {
	case *Metric_Gauge:
		return d.Gauge == nil || validUTF8DataPoints(d.Gauge.DataPoints, validUTF8NumberDataPoint)
	case *Metric_Sum:
		return d.Sum == nil || validUTF8DataPoints(d.Sum.DataPoints, validUTF8NumberDataPoint)
	case *Metric_Histogram:
		return d.Histogram == nil || validUTF8DataPoints(d.Histogram.DataPoints, validUTF8HistogramDataPoint)
	case *Metric_ExponentialHistogram:
		return d.ExponentialHistogram == nil ||
			validUTF8DataPoints(d.ExponentialHistogram.DataPoints, validUTF8ExponentialHistogramDataPoint)
	case *Metric_Summary:
		return d.Summary == nil || validUTF8DataPoints(d.Summary.DataPoints, validUTF8SummaryDataPoint)
	}
	return true
}

func countMetricDataPoints(m *Metric) int {
	switch d := m.Data.(type) {
	case *Metric_Gauge:
		if d.Gauge != nil {
			return len(d.Gauge.DataPoints)
		}
	case *Metric_Sum:
		if d.Sum != nil {
			return len(d.Sum.DataPoints)
		}
	case *Metric_Histogram:
		if d.Histogram != nil {
			return len(d.Histogram.DataPoints)
		}
	case *Metric_ExponentialHistogram:
		if d.ExponentialHistogram != nil {
			return len(d.ExponentialHistogram.DataPoints)
		}
	case *Metric_Summary:
		if d.Summary != nil {
			return len(d.Summary.DataPoints)
		}
	}
	return 0
}

// rejectMetricDataPoints filters invalid data points inside the metric oneof and returns the number removed.
func rejectMetricDataPoints(m *Metric) int {
	rejected := 0
	switch d := m.Data.(type) {
	case *Metric_Gauge:
		if d.Gauge != nil {
			d.Gauge.DataPoints = rejectDataPoints(d.Gauge.DataPoints, DeleteNumberDataPoint, validUTF8NumberDataPoint, &rejected)
		}
	case *Metric_Sum:
		if d.Sum != nil {
			d.Sum.DataPoints = rejectDataPoints(d.Sum.DataPoints, DeleteNumberDataPoint, validUTF8NumberDataPoint, &rejected)
		}
	case *Metric_Histogram:
		if d.Histogram != nil {
			d.Histogram.DataPoints = rejectDataPoints(d.Histogram.DataPoints, DeleteHistogramDataPoint, validUTF8HistogramDataPoint, &rejected)
		}
	case *Metric_ExponentialHistogram:
		if d.ExponentialHistogram != nil {
			d.ExponentialHistogram.DataPoints = rejectDataPoints(d.ExponentialHistogram.DataPoints,
				DeleteExponentialHistogramDataPoint, validUTF8ExponentialHistogramDataPoint, &rejected)
		}
	case *Metric_Summary:
		if d.Summary != nil {
			d.Summary.DataPoints = rejectDataPoints(d.Summary.DataPoints, DeleteSummaryDataPoint, validUTF8SummaryDataPoint, &rejected)
		}
	}
	return rejected
}

func rejectDataPoints[T any](s []*T, del func(*T, bool), valid func(*T) bool, rejected *int) []*T {
	return removeIf(s, del, func(dp *T) bool {
		invalid := !valid(dp)
		if invalid {
			*rejected++
		}
		return invalid
	})
}

// RejectInvalidUTF8ExportMetricsServiceRequest removes data points containing invalid UTF-8 from req
// and returns the number removed. A metric with invalid metadata is removed with all its data points.
func RejectInvalidUTF8ExportMetricsServiceRequest(req *ExportMetricsServiceRequest) int {
	rejected := 0
	req.ResourceMetrics = removeIf(req.ResourceMetrics, DeleteResourceMetrics, func(rm *ResourceMetrics) bool {
		if !validUTF8Resource(&rm.Resource) || !utf8.ValidString(rm.SchemaUrl) {
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					rejected += countMetricDataPoints(m)
				}
			}
			return true
		}
		if len(rm.ScopeMetrics) == 0 {
			return false
		}
		rm.ScopeMetrics = removeIf(rm.ScopeMetrics, DeleteScopeMetrics, func(sm *ScopeMetrics) bool {
			if !validUTF8InstrumentationScope(&sm.Scope) || !utf8.ValidString(sm.SchemaUrl) {
				for _, m := range sm.Metrics {
					rejected += countMetricDataPoints(m)
				}
				return true
			}
			if len(sm.Metrics) == 0 {
				return false
			}
			sm.Metrics = removeIf(sm.Metrics, DeleteMetric, func(m *Metric) bool {
				if !validUTF8MetricHeader(m) {
					rejected += countMetricDataPoints(m)
					return true
				}
				if countMetricDataPoints(m) == 0 {
					return false
				}
				rejected += rejectMetricDataPoints(m)
				return countMetricDataPoints(m) == 0
			})
			return len(sm.Metrics) == 0
		})
		return len(rm.ScopeMetrics) == 0
	})
	return rejected
}
