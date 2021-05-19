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

// MetricsSerializer encodes protocol-specific data model into bytes.
type MetricsSerializer interface {
	SerializeMetrics(model interface{}) ([]byte, error)
}

// TracesSerializer encodes protocol-specific data model into bytes.
type TracesSerializer interface {
	SerializeTraces(model interface{}) ([]byte, error)
}

// LogsSerializer encodes protocol-specific data model into bytes.
type LogsSerializer interface {
	SerializeLogs(model interface{}) ([]byte, error)
}
