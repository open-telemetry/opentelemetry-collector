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

package serializer

// MetricsMarshaler encodes protocol-specific data model into bytes.
type MetricsMarshaler interface {
	MarshalMetrics(model interface{}) ([]byte, error)
}

// TracesMarshaler encodes protocol-specific data model into bytes.
type TracesMarshaler interface {
	MarshalTraces(model interface{}) ([]byte, error)
}

// LogsMarshaler encodes protocol-specific data model into bytes.
type LogsMarshaler interface {
	MarshalLogs(model interface{}) ([]byte, error)
}
