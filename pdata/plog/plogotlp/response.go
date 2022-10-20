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

package plogotlp // import "go.opentelemetry.io/collector/pdata/plog/plogotlp"
import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"
	"go.opentelemetry.io/collector/pdata/plog/internal/plogjson"
)

// Response represents the response for gRPC/HTTP client/server.
type Response struct {
	orig *otlpcollectorlog.ExportLogsServiceResponse
}

// NewResponse returns an empty Response.
func NewResponse() Response {
	return Response{orig: &otlpcollectorlog.ExportLogsServiceResponse{}}
}

// MarshalProto marshals Response into proto bytes.
func (lr Response) MarshalProto() ([]byte, error) {
	return lr.orig.Marshal()
}

// UnmarshalProto unmarshalls Response from proto bytes.
func (lr Response) UnmarshalProto(data []byte) error {
	return lr.orig.Unmarshal(data)
}

// MarshalJSON marshals Response into JSON bytes.
func (lr Response) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := plogjson.JSONMarshaler.Marshal(&buf, lr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls Response from JSON bytes.
func (lr Response) UnmarshalJSON(data []byte) error {
	return plogjson.UnmarshalExportLogsServiceResponse(data, lr.orig)
}

// PartialSuccess returns the ExportLogsPartialSuccess associated with this Response.
func (lr Response) PartialSuccess() ExportLogsPartialSuccess {
	return ExportLogsPartialSuccess(internal.NewExportLogsPartialSuccess(&lr.orig.PartialSuccess))
}
