// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetricotlp // import "go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

// ExportResponse represents the response for gRPC/HTTP client/server.
type ExportResponse struct {
	orig  *otlpcollectormetrics.ExportMetricsServiceResponse
	state *internal.State
}

// NewExportResponse returns an empty ExportResponse.
func NewExportResponse() ExportResponse {
	state := internal.StateMutable
	return ExportResponse{
		orig:  &otlpcollectormetrics.ExportMetricsServiceResponse{},
		state: &state,
	}
}

// MarshalProto marshals ExportResponse into proto bytes.
func (ms ExportResponse) MarshalProto() ([]byte, error) {
	return ms.orig.Marshal()
}

// UnmarshalProto unmarshalls ExportResponse from proto bytes.
func (ms ExportResponse) UnmarshalProto(data []byte) error {
	return ms.orig.Unmarshal(data)
}

// MarshalJSON marshals ExportResponse into JSON bytes.
func (ms ExportResponse) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Marshal(&buf, ms.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls ExportResponse from JSON bytes.
func (ms ExportResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(bytes.NewBuffer(data), ms.orig)
}

// PartialSuccess returns the ExportLogsPartialSuccess associated with this ExportResponse.
func (ms ExportResponse) PartialSuccess() ExportPartialSuccess {
	return newExportPartialSuccess(&ms.orig.PartialSuccess, ms.state)
}
