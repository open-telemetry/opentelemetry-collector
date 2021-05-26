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

// MetricsDecoder interface to decode bytes into protocol-specific data model.
type MetricsDecoder interface {
	// DecodeMetrics decodes bytes into protocol-specific data model.
	DecodeMetrics(buf []byte) (interface{}, error)
}

// TracesDecoder interface to decode bytes into protocol-specific data model.
type TracesDecoder interface {
	// DecodeTraces decodes bytes into protocol-specific data model.
	DecodeTraces(buf []byte) (interface{}, error)
}

// LogsDecoder interface to decode bytes into protocol-specific data model.
type LogsDecoder interface {
	// DecodeLogs decodes bytes into protocol-specific data model.
	DecodeLogs(buf []byte) (interface{}, error)
}
