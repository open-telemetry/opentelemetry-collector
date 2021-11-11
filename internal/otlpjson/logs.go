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

package otlpjson // import "go.opentelemetry.io/collector/internal/otlpjson"

import (
	"go.opentelemetry.io/collector/model/pdata"
)

// NewJSONLogsMarshaler returns a serializer.LogsMarshaler to encode to OTLP JSON bytes.
func NewJSONLogsMarshaler() pdata.LogsMarshaler {
	return jsonLogsMarshaler{}
}

type jsonLogsMarshaler struct{}

// MarshalLogs pdata.Logs to OTLP JSON.
func (jsonLogsMarshaler) MarshalLogs(ld pdata.Logs) ([]byte, error) {
	buf := dataBuffer{}
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)

		attrs := rl.Resource().Attributes()

		ills := rl.InstrumentationLibraryLogs()
		if ills.Len() == 0 {
			buf.object(func() {
				buf.resource("resourceLog", attrs)
			})
			continue
		}
		for j := 0; j < ills.Len(); j++ {
			ils := ills.At(j)

			lib := ils.InstrumentationLibrary()

			logs := ils.Logs()
			if logs.Len() == 0 {
				buf.object(func() {
					buf.resource("instrumentationLibraryLog", attrs)
					buf.instrumentationLibrary(lib)
				})
				continue
			}
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)

				buf.object(func() {
					buf.resource("log", attrs)
					buf.instrumentationLibrary(lib)

					buf.fieldTime("timestamp", log.Timestamp())
					buf.fieldString("severityText", log.SeverityText())
					buf.fieldString("severityNumber", log.SeverityNumber().String())
					buf.fieldString("name", log.Name())
					buf.fieldAttr("body", log.Body())
					buf.fieldUint32("droppedAttributesCount", log.DroppedAttributesCount())
					buf.fieldAttrs("attributes", log.Attributes())
					buf.fieldString("traceID", log.TraceID().HexString())
					buf.fieldString("spanID", log.SpanID().HexString())
					buf.fieldUint32("flags", log.Flags())
				})
			}
		}
	}

	return buf.buf.Bytes(), nil
}
