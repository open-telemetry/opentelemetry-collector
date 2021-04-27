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

package protocols

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// Type is a protocol type such as `otlp/protobuf`.
type Type string

// String returns protocol type in string format.
func (e Type) String() string {
	return string(e)
}

// Unmarshalers contains list of unmarshalers for all telemetry types.
type Unmarshalers struct {
	// Metrics unmarshalers.
	Metrics []MetricsUnmarshaler
	// Traces unmarshalers.
	Traces []TracesUnmarshaler
	// Logs unmarshalers.
	Logs []LogsUnmarshaler
}

// UnmarshalersMap contains a mapping of protocol types to unmarshalers for all telemetry types.
type UnmarshalersMap struct {
	// Metrics is map of protocol type to metrics unmarshalers.
	Metrics map[Type]MetricsUnmarshaler
	// Traces is map of protocol type to metrics unmarshalers.
	Traces map[Type]TracesUnmarshaler
	// Logs is map of protocol type to metrics unmarshalers.
	Logs map[Type]LogsUnmarshaler
}

// ByType returns an UnmarshalersMap from the set marshalers.
func (u Unmarshalers) ByType() UnmarshalersMap {
	metrics := map[Type]MetricsUnmarshaler{}
	for _, unmarshaler := range u.Metrics {
		metrics[unmarshaler.Type()] = unmarshaler
	}

	traces := map[Type]TracesUnmarshaler{}
	for _, unmarshaler := range u.Traces {
		traces[unmarshaler.Type()] = unmarshaler
	}

	logs := map[Type]LogsUnmarshaler{}
	for _, unmarshaler := range u.Logs {
		logs[unmarshaler.Type()] = unmarshaler
	}

	return UnmarshalersMap{
		Metrics: metrics,
		Traces:  traces,
		Logs:    logs,
	}
}

// MetricsUnmarshaler deserializes the message body.
type MetricsUnmarshaler interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (pdata.Metrics, error)

	// Type of the serialized messages.
	Type() Type
}

// TracesUnmarshaler deserializes the message body.
type TracesUnmarshaler interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (pdata.Traces, error)

	// Type of the serialized messages.
	Type() Type
}

// LogsUnmarshaler deserializes the message body.
type LogsUnmarshaler interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (pdata.Logs, error)

	// Type of the serialized messages.
	Type() Type
}

// DefaultUnmarshalers returns set of default unmarshalers.
func DefaultUnmarshalers() *Unmarshalers {
	return &Unmarshalers{
		Metrics: nil,
		Traces: []TracesUnmarshaler{
			&otlpTracesPbUnmarshaler{},
			&jaegerProtoSpanUnmarshaler{},
			&jaegerJSONSpanUnmarshaler{},
			&zipkinProtoSpanUnmarshaler{},
			&zipkinJSONSpanUnmarshaler{},
			&zipkinThriftSpanUnmarshaler{},
		},
		Logs: []LogsUnmarshaler{&otlpLogsPbUnmarshaler{}},
	}
}
