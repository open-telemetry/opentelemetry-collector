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
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/internal/ptracejson"
)

// Request represents the request for gRPC/HTTP client/server.
// It's a wrapper for ptrace.Traces data.
type Request struct {
	orig *otlpcollectortrace.ExportTraceServiceRequest
}

// NewRequest returns an empty Request.
func NewRequest() Request {
	return Request{orig: &otlpcollectortrace.ExportTraceServiceRequest{}}
}

// NewRequestFromTraces returns a Request from ptrace.Traces.
// Because Request is a wrapper for ptrace.Traces,
// any changes to the provided Traces struct will be reflected in the Request and vice versa.
func NewRequestFromTraces(td ptrace.Traces) Request {
	return Request{orig: internal.GetOrigTraces(internal.Traces(td))}
}

// MarshalProto marshals Request into proto bytes.
func (tr Request) MarshalProto() ([]byte, error) {
	return tr.orig.Marshal()
}

// UnmarshalProto unmarshalls Request from proto bytes.
func (tr Request) UnmarshalProto(data []byte) error {
	if err := tr.orig.Unmarshal(data); err != nil {
		return err
	}
	otlp.MigrateTraces(tr.orig.ResourceSpans)
	return nil
}

// MarshalJSON marshals Request into JSON bytes.
func (tr Request) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := ptracejson.JSONMarshaler.Marshal(&buf, tr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls Request from JSON bytes.
func (tr Request) UnmarshalJSON(data []byte) error {
	return ptracejson.UnmarshalExportTraceServiceRequest(data, tr.orig)
}

func (tr Request) Traces() ptrace.Traces {
	return ptrace.Traces(internal.NewTraces(tr.orig))
}
