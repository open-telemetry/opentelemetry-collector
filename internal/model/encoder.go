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

package model

// MetricsEncoder encodes protocol-specific data model into bytes.
type MetricsEncoder interface {
	// EncodeMetrics converts protocol-specific data model into bytes.
	EncodeMetrics(model interface{}) ([]byte, error)
}

// TracesEncoder encodes protocol-specific data model into bytes.
type TracesEncoder interface {
	// EncodeTraces converts protocol-specific data model into bytes.
	EncodeTraces(model interface{}) ([]byte, error)
}

// LogsEncoder encodes protocol-specific data model into bytes.
type LogsEncoder interface {
	// EncodeLogs converts protocol-specific data model into bytes.
	EncodeLogs(model interface{}) ([]byte, error)
}
