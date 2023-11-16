// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/internal/otlptext"

import "go.opentelemetry.io/collector/pdata/pmetric"

// NewTextMetricsMarshaler returns a pmetric.Marshaler to encode to OTLP text bytes.
func NewTextMetricsMarshaler() pmetric.Marshaler {
	return textMetricsMarshaler{}
}

type textMetricsMarshaler struct{}

// MarshalMetrics pmetric.Metrics to OTLP text.
func (textMetricsMarshaler) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	buf := dataBuffer{}
	md.ResourceMetrics().ForEachIndex(func(i int, rm pmetric.ResourceMetrics) {
		buf.logEntry("ResourceMetrics #%d", i)
		buf.logEntry("Resource SchemaURL: %s", rm.SchemaUrl())
		buf.logAttributes("Resource attributes", rm.Resource().Attributes())
		rm.ScopeMetrics().ForEachIndex(func(j int, ilm pmetric.ScopeMetrics) {
			buf.logEntry("ScopeMetrics #%d", j)
			buf.logEntry("ScopeMetrics SchemaURL: %s", ilm.SchemaUrl())
			buf.logInstrumentationScope(ilm.Scope())
			ilm.Metrics().ForEachIndex(func(k int, metric pmetric.Metric) {
				buf.logEntry("Metric #%d", k)
				buf.logMetricDescriptor(metric)
				buf.logMetricDataPoints(metric)
			})
		})
	})
	return buf.buf.Bytes(), nil
}
