// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	"go.opentelemetry.io/collector/pdata/pmetric"
) // MetricsCountSizer returns the nunmber of metrics entries.

type MetricsSizer interface {
	MetricsSize(md pmetric.Metrics) (count int)
	ResourceMetricsSize(rm pmetric.ResourceMetrics) (count int)
	ScopeMetricsSize(sm pmetric.ScopeMetrics) (count int)
	MetricSize(m pmetric.Metric) int
	DeltaSize(newItemSize int) int
	NumberDataPointSize(ndp pmetric.NumberDataPoint) int
	HistogramDataPointSize(hdp pmetric.HistogramDataPoint) int
	ExponentialHistogramDataPointSize(ehdp pmetric.ExponentialHistogramDataPoint) int
	SummaryDataPointSize(sdps pmetric.SummaryDataPoint) int
}

type MetricsBytesSizer struct {
	pmetric.ProtoMarshaler
}

var _ MetricsSizer = &MetricsBytesSizer{}

// DeltaSize() returns the delta size of a proto slice when a new item is added.
// Example:
//
//	prevSize := proto1.Size()
//	proto1.RepeatedField().AppendEmpty() = proto2
//
// Then currSize of proto1 can be calculated as
//
//	currSize := (prevSize + sizer.DeltaSize(proto2.Size()))
//
// This is derived from opentelemetry-collector/pdata/internal/data/protogen/metrics/v1/metrics.pb.go
// which is generated with gogo/protobuf.
func (s *MetricsBytesSizer) DeltaSize(newItemSize int) int {
	return 1 + newItemSize + sov(uint64(newItemSize)) //nolint:gosec // disable G115
}

type MetricsCountSizer struct{}

var _ MetricsSizer = &MetricsCountSizer{}

func (s *MetricsCountSizer) MetricsSize(md pmetric.Metrics) int {
	return md.DataPointCount()
}

func (s *MetricsCountSizer) ResourceMetricsSize(rm pmetric.ResourceMetrics) (count int) {
	for i := 0; i < rm.ScopeMetrics().Len(); i++ {
		count += s.ScopeMetricsSize(rm.ScopeMetrics().At(i))
	}
	return count
}

func (s *MetricsCountSizer) ScopeMetricsSize(sm pmetric.ScopeMetrics) (count int) {
	for i := 0; i < sm.Metrics().Len(); i++ {
		count += s.MetricSize(sm.Metrics().At(i))
	}
	return count
}

func (s *MetricsCountSizer) MetricSize(m pmetric.Metric) int {
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

func (s *MetricsCountSizer) DeltaSize(newItemSize int) int {
	return newItemSize
}

func (s *MetricsCountSizer) NumberDataPointSize(_ pmetric.NumberDataPoint) int {
	return 1
}

func (s *MetricsCountSizer) HistogramDataPointSize(_ pmetric.HistogramDataPoint) int {
	return 1
}

func (s *MetricsCountSizer) ExponentialHistogramDataPointSize(_ pmetric.ExponentialHistogramDataPoint) int {
	return 1
}

func (s *MetricsCountSizer) SummaryDataPointSize(_ pmetric.SummaryDataPoint) int {
	return 1
}

// func (s *LogsCountSizer) LogsSize(ld plog.Logs) int {
// 	return ld.LogRecordCount()
// }

// func (s *LogsCountSizer) ResourceLogsSize(rl plog.ResourceLogs) int {
// 	count := 0
// 	for k := 0; k < rl.ScopeLogs().Len(); k++ {
// 		count += rl.ScopeLogs().At(k).LogRecords().Len()
// 	}
// 	return count
// }

// func (s *LogsCountSizer) ScopeLogsSize(sl plog.ScopeLogs) int {
// 	return sl.LogRecords().Len()
// }

// func (s *LogsCountSizer) LogRecordSize(_ plog.LogRecord) int {
// 	return 1
// }

// func (s *LogsCountSizer) DeltaSize(newItemSize int) int {
// 	return newItemSize
// }
