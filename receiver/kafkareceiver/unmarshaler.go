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

package kafkareceiver

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// TracesUnmarshaler deserializes the message body.
type TracesUnmarshaler interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (pdata.Traces, error)

	// Encoding of the serialized messages.
	Encoding() string
}

// LogsUnmarshaler deserializes the message body.
type LogsUnmarshaler interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (pdata.Logs, error)

	// Encoding of the serialized messages.
	Encoding() string
}

// defaultTracesUnmarshalers returns map of supported encodings with TracesUnmarshaler.
func defaultTracesUnmarshalers() map[string]TracesUnmarshaler {
	otlp := &otlpTracesPbUnmarshaler{}
	jaegerProto := jaegerProtoSpanUnmarshaler{}
	jaegerJSON := jaegerJSONSpanUnmarshaler{}
	zipkinProto := zipkinProtoSpanUnmarshaler{}
	zipkinJSON := zipkinJSONSpanUnmarshaler{}
	zipkinThrift := zipkinThriftSpanUnmarshaler{}
	return map[string]TracesUnmarshaler{
		otlp.Encoding():         otlp,
		jaegerProto.Encoding():  jaegerProto,
		jaegerJSON.Encoding():   jaegerJSON,
		zipkinProto.Encoding():  zipkinProto,
		zipkinJSON.Encoding():   zipkinJSON,
		zipkinThrift.Encoding(): zipkinThrift,
	}
}

func defaultLogsUnmarshalers() map[string]LogsUnmarshaler {
	otlp := &otlpLogsPbUnmarshaler{}
	return map[string]LogsUnmarshaler{
		otlp.Encoding(): otlp,
	}
}
