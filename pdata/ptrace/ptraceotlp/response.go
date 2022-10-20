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

package ptraceotlp // import "go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/pdata/ptrace/internal/ptracejson"
)

// Response represents the response for gRPC/HTTP client/server.
type Response struct {
	orig *otlpcollectortrace.ExportTraceServiceResponse
}

// NewResponse returns an empty Response.
func NewResponse() Response {
	return Response{orig: &otlpcollectortrace.ExportTraceServiceResponse{}}
}

// MarshalProto marshals Response into proto bytes.
func (tr Response) MarshalProto() ([]byte, error) {
	return tr.orig.Marshal()
}

// UnmarshalProto unmarshalls Response from proto bytes.
func (tr Response) UnmarshalProto(data []byte) error {
	return tr.orig.Unmarshal(data)
}

// MarshalJSON marshals Response into JSON bytes.
func (tr Response) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := ptracejson.JSONMarshaler.Marshal(&buf, tr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls Response from JSON bytes.
func (tr Response) UnmarshalJSON(data []byte) error {
	return ptracejson.UnmarshalExportTraceServiceResponse(data, tr.orig)
}

// PartialSuccess returns the ExportLogsPartialSuccess associated with this Response.
func (tr Response) PartialSuccess() ExportTracePartialSuccess {
	return ExportTracePartialSuccess(internal.NewExportTracePartialSuccess(&tr.orig.PartialSuccess))
}
