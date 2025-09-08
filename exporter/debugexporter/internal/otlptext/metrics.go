// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"go.opentelemetry.io/collector/exporter/debugexporter/internal"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// NewTextMetricsMarshaler returns a pmetric.Marshaler to encode to OTLP text bytes.
func NewTextMetricsMarshaler(outputConfig internal.OutputConfig) pmetric.Marshaler {
	return textMetricsMarshaler{
		outputConfig: outputConfig,
	}
}

type textMetricsMarshaler struct {
	outputConfig internal.OutputConfig
}

// MarshalMetrics pmetric.Metrics to OTLP text.
func (t textMetricsMarshaler) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	buf := dataBuffer{}
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)

		if t.outputConfig.Resource.Enabled {
			buf.logEntry("ResourceMetrics #%d", i)
			buf.logEntry("Resource SchemaURL: %s", rm.SchemaUrl())
			buf.logAttributes("Resource attributes", rm.Resource().Attributes(), &t.outputConfig.Resource.AttributesOutputConfig)
			buf.logEntityRefs(rm.Resource())
		}
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)

			if t.outputConfig.Scope.Enabled {
				buf.logEntry("ScopeMetrics #%d", j)
				buf.logEntry("ScopeMetrics SchemaURL: %s", ilm.SchemaUrl())
				buf.logInstrumentationScope(ilm.Scope(), &t.outputConfig.Scope.Attributes)
			}
			if !t.outputConfig.Record.Enabled {
				continue
			}
			metrics := ilm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				buf.logEntry("Metric #%d", k)
				metric := metrics.At(k)
				buf.logMetricDescriptor(metric)
				buf.logMetricDataPoints(metric, &t.outputConfig.Record.Attributes)
			}
		}
	}

	return buf.buf.Bytes(), nil
}
