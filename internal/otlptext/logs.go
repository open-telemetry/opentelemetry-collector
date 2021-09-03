// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlptext

import (
	"go.opentelemetry.io/collector/model/pdata"
)

// NewTextLogsMarshaler returns a serializer.LogsMarshaler to encode to OTLP text bytes.
func NewTextLogsMarshaler() pdata.LogsMarshaler {
	return textLogsMarshaler{}
}

type textLogsMarshaler struct{}

// MarshalLogs pdata.Logs to OTLP text.
func (textLogsMarshaler) MarshalLogs(ld pdata.Logs) ([]byte, error) {
	buf := dataBuffer{}
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		buf.logEntry("ResourceLog #%d", i)
		rl := rls.At(i)
		buf.logAttributeMap("Resource labels", rl.Resource().Attributes())
		ills := rl.InstrumentationLibraryLogs()
		for j := 0; j < ills.Len(); j++ {
			buf.logEntry("InstrumentationLibraryLogs #%d", j)
			ils := ills.At(j)
			buf.logInstrumentationLibrary(ils.InstrumentationLibrary())

			logs := ils.Logs()
			for k := 0; k < logs.Len(); k++ {
				buf.logEntry("LogRecord #%d", k)
				lr := logs.At(k)
				buf.logEntry("Timestamp: %s", lr.Timestamp())
				buf.logEntry("Severity: %s", lr.SeverityText())
				buf.logEntry("ShortName: %s", lr.Name())
				buf.logEntry("Body: %s", attributeValueToString(lr.Body()))
				buf.logAttributeMap("Attributes", lr.Attributes())
				buf.logEntry("Trace ID: %s", lr.TraceID().HexString())
				buf.logEntry("Span ID: %s", lr.SpanID().HexString())
				buf.logEntry("Flags: %d", lr.Flags())
			}
		}
	}

	return buf.buf.Bytes(), nil
}
