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

package pmetricotlp // import "go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pmetric/internal/pmetricjson"
)

// Response represents the response for gRPC/HTTP client/server.
type Response struct {
	orig *otlpcollectormetrics.ExportMetricsServiceResponse
}

// NewResponse returns an empty Response.
func NewResponse() Response {
	return Response{orig: &otlpcollectormetrics.ExportMetricsServiceResponse{}}
}

// MarshalProto marshals Response into proto bytes.
func (mr Response) MarshalProto() ([]byte, error) {
	return mr.orig.Marshal()
}

// UnmarshalProto unmarshalls Response from proto bytes.
func (mr Response) UnmarshalProto(data []byte) error {
	return mr.orig.Unmarshal(data)
}

// MarshalJSON marshals Response into JSON bytes.
func (mr Response) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := pmetricjson.JSONMarshaler.Marshal(&buf, mr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls Response from JSON bytes.
func (mr Response) UnmarshalJSON(data []byte) error {
	return pmetricjson.UnmarshalExportMetricsServiceResponse(data, mr.orig)
}

// PartialSuccess returns the ExportLogsPartialSuccess associated with this Response.
func (mr Response) PartialSuccess() ExportMetricsPartialSuccess {
	return ExportMetricsPartialSuccess(internal.NewExportMetricsPartialSuccess(&mr.orig.PartialSuccess))
}
