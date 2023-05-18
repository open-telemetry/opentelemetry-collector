// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogotlp // import "go.opentelemetry.io/collector/pdata/plog/plogotlp"

import (
	"bytes"

	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"
	"go.opentelemetry.io/collector/pdata/plog/internal/plogjson"
)

// ExportResponse represents the response for gRPC/HTTP client/server.
type ExportResponse struct {
	orig *otlpcollectorlog.ExportLogsServiceResponse
}

// NewExportResponse returns an empty ExportResponse.
func NewExportResponse() ExportResponse {
	return ExportResponse{orig: &otlpcollectorlog.ExportLogsServiceResponse{}}
}

// MarshalProto marshals ExportResponse into proto bytes.
func (lr ExportResponse) MarshalProto() ([]byte, error) {
	return lr.orig.Marshal()
}

// UnmarshalProto unmarshalls ExportResponse from proto bytes.
func (lr ExportResponse) UnmarshalProto(data []byte) error {
	return lr.orig.Unmarshal(data)
}

// MarshalJSON marshals ExportResponse into JSON bytes.
func (lr ExportResponse) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := plogjson.JSONMarshaler.Marshal(&buf, lr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls ExportResponse from JSON bytes.
func (lr ExportResponse) UnmarshalJSON(data []byte) error {
	return plogjson.UnmarshalExportLogsServiceResponse(data, lr.orig)
}

// PartialSuccess returns the ExportPartialSuccess associated with this ExportResponse.
func (lr ExportResponse) PartialSuccess() ExportPartialSuccess {
	return newExportPartialSuccess(&lr.orig.PartialSuccess)
}
