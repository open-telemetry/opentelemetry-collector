// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentityotlp // import "go.opentelemetry.io/collector/pdata/pentity/pentityotlp"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorentity "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/entities/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/pentity"
)

var jsonUnmarshaler = &pentity.JSONUnmarshaler{}

// ExportRequest represents the request for gRPC/HTTP client/server.
// It's a wrapper for pentity.Entities data.
type ExportRequest struct {
	orig  *otlpcollectorentity.ExportEntitiesServiceRequest
	state *internal.State
}

// NewExportRequest returns an empty ExportRequest.
func NewExportRequest() ExportRequest {
	state := internal.StateMutable
	return ExportRequest{
		orig:  &otlpcollectorentity.ExportEntitiesServiceRequest{},
		state: &state,
	}
}

// NewExportRequestFromEntities returns a ExportRequest from pentity.Entities.
// Because ExportRequest is a wrapper for pentity.Entities,
// any changes to the provided Entities struct will be reflected in the ExportRequest and vice versa.
func NewExportRequestFromEntities(ld pentity.Entities) ExportRequest {
	return ExportRequest{
		orig:  internal.GetOrigEntities(internal.Entities(ld)),
		state: internal.GetEntitiesState(internal.Entities(ld)),
	}
}

// MarshalProto marshals ExportRequest into proto bytes.
func (ms ExportRequest) MarshalProto() ([]byte, error) {
	return ms.orig.Marshal()
}

// UnmarshalProto unmarshalls ExportRequest from proto bytes.
func (ms ExportRequest) UnmarshalProto(data []byte) error {
	return ms.orig.Unmarshal(data)
}

// MarshalJSON marshals ExportRequest into JSON bytes.
func (ms ExportRequest) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Marshal(&buf, ms.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls ExportRequest from JSON bytes.
func (ms ExportRequest) UnmarshalJSON(data []byte) error {
	ld, err := jsonUnmarshaler.UnmarshalEntities(data)
	if err != nil {
		return err
	}
	*ms.orig = *internal.GetOrigEntities(internal.Entities(ld))
	return nil
}

func (ms ExportRequest) Entities() pentity.Entities {
	return pentity.Entities(internal.NewEntities(ms.orig, ms.state))
}
