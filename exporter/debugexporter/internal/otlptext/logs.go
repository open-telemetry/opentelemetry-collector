// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"go.opentelemetry.io/collector/exporter/debugexporter/internal"
	"go.opentelemetry.io/collector/pdata/plog"
)

// NewTextLogsMarshaler returns a plog.Marshaler to encode to OTLP text bytes.
func NewTextLogsMarshaler(outputConfig internal.OutputConfig) plog.Marshaler {
	return textLogsMarshaler{
		outputConfig: outputConfig,
	}
}

type textLogsMarshaler struct {
	outputConfig internal.OutputConfig
}

// MarshalLogs plog.Logs to OTLP text.
func (t textLogsMarshaler) MarshalLogs(ld plog.Logs) ([]byte, error) {
	buf := dataBuffer{}
	rls := ld.ResourceLogs()
	if !t.outputConfig.Resource.Enabled {
		return buf.buf.Bytes(), nil
	}
	for i := 0; i < rls.Len(); i++ {
		buf.logEntry("ResourceLog #%d", i)
		rl := rls.At(i)
		buf.logEntry("Resource SchemaURL: %s", rl.SchemaUrl())
		buf.logAttributes("Resource attributes", rl.Resource().Attributes(), &t.outputConfig.Resource.AttributesOutputConfig)
		buf.logEntityRefs(rl.Resource())
		ills := rl.ScopeLogs()
		if !t.outputConfig.Scope.Enabled {
			continue
		}
		for j := 0; j < ills.Len(); j++ {
			buf.logEntry("ScopeLogs #%d", j)
			ils := ills.At(j)
			buf.logEntry("ScopeLogs SchemaURL: %s", ils.SchemaUrl())
			buf.logInstrumentationScope(ils.Scope(), &t.outputConfig.Scope.AttributesOutputConfig)

			logs := ils.LogRecords()
			if !t.outputConfig.Record.Enabled {
				continue
			}
			for k := 0; k < logs.Len(); k++ {
				buf.logEntry("LogRecord #%d", k)
				lr := logs.At(k)
				buf.logEntry("ObservedTimestamp: %s", lr.ObservedTimestamp())
				buf.logEntry("Timestamp: %s", lr.Timestamp())
				buf.logEntry("SeverityText: %s", lr.SeverityText())
				buf.logEntry("SeverityNumber: %s(%d)", lr.SeverityNumber(), lr.SeverityNumber())
				if lr.EventName() != "" {
					buf.logEntry("EventName: %s", lr.EventName())
				}
				buf.logEntry("Body: %s", valueToString(lr.Body()))
				buf.logAttributes("Attributes", lr.Attributes(), &t.outputConfig.Record.AttributesOutputConfig)
				buf.logEntry("Trace ID: %s", lr.TraceID())
				buf.logEntry("Span ID: %s", lr.SpanID())
				buf.logEntry("Flags: %d", lr.Flags())
			}
		}
	}

	return buf.buf.Bytes(), nil
}
