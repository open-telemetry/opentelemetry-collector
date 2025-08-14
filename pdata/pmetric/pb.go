// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalMetrics(md Metrics) ([]byte, error) {
	size := internal.SizeProtoOrigExportMetricsServiceRequest(md.getOrig())
	buf := make([]byte, size)
	_ = internal.MarshalProtoOrigExportMetricsServiceRequest(md.getOrig(), buf)
	return buf, nil
}

func (e *ProtoMarshaler) MetricsSize(md Metrics) int {
	return internal.SizeProtoOrigExportMetricsServiceRequest(md.getOrig())
}

func (e *ProtoMarshaler) ResourceMetricsSize(rm ResourceMetrics) int {
	return internal.SizeProtoOrigResourceMetrics(rm.orig)
}

func (e *ProtoMarshaler) ScopeMetricsSize(sm ScopeMetrics) int {
	return internal.SizeProtoOrigScopeMetrics(sm.orig)
}

func (e *ProtoMarshaler) MetricSize(m Metric) int {
	return internal.SizeProtoOrigMetric(m.orig)
}

func (e *ProtoMarshaler) NumberDataPointSize(ndp NumberDataPoint) int {
	return internal.SizeProtoOrigNumberDataPoint(ndp.orig)
}

func (e *ProtoMarshaler) SummaryDataPointSize(sdps SummaryDataPoint) int {
	return internal.SizeProtoOrigSummaryDataPoint(sdps.orig)
}

func (e *ProtoMarshaler) HistogramDataPointSize(hdp HistogramDataPoint) int {
	return internal.SizeProtoOrigHistogramDataPoint(hdp.orig)
}

func (e *ProtoMarshaler) ExponentialHistogramDataPointSize(ehdp ExponentialHistogramDataPoint) int {
	return internal.SizeProtoOrigExponentialHistogramDataPoint(ehdp.orig)
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	md := NewMetrics()
	err := internal.UnmarshalProtoOrigExportMetricsServiceRequest(md.getOrig(), buf)
	if err != nil {
		return Metrics{}, err
	}
	return md, nil
}
