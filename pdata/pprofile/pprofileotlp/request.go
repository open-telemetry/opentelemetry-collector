// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp // import "go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1experimental"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var jsonUnmarshaler = &pprofile.JSONUnmarshaler{}

// ExportRequest represents the request for gRPC/HTTP client/server.
// It's a wrapper for pprofile.Profiles data.
type ExportRequest struct {
	orig  *otlpcollectorprofile.ExportProfilesServiceRequest
	state *internal.State
}

// NewExportRequest returns an empty ExportRequest.
func NewExportRequest() ExportRequest {
	state := internal.StateMutable
	return ExportRequest{
		orig:  &otlpcollectorprofile.ExportProfilesServiceRequest{},
		state: &state,
	}
}

// NewExportRequestFromProfiles returns a ExportRequest from pprofile.Profiles.
// Because ExportRequest is a wrapper for pprofile.Profiles,
// any changes to the provided Profiles struct will be reflected in the ExportRequest and vice versa.
func NewExportRequestFromProfiles(td pprofile.Profiles) ExportRequest {
	return ExportRequest{
		orig:  internal.GetOrigProfiles(internal.Profiles(td)),
		state: internal.GetProfilesState(internal.Profiles(td)),
	}
}

// MarshalProto marshals ExportRequest into proto bytes.
func (ms ExportRequest) MarshalProto() ([]byte, error) {
	return ms.orig.Marshal()
}

// UnmarshalProto unmarshalls ExportRequest from proto bytes.
func (ms ExportRequest) UnmarshalProto(data []byte) error {
	if err := ms.orig.Unmarshal(data); err != nil {
		return err
	}
	otlp.MigrateProfiles(ms.orig.ResourceProfiles)
	return nil
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
	td, err := jsonUnmarshaler.UnmarshalProfiles(data)
	if err != nil {
		return err
	}
	*ms.orig = *internal.GetOrigProfiles(internal.Profiles(td))
	return nil
}

func (ms ExportRequest) Profiles() pprofile.Profiles {
	return pprofile.Profiles(internal.NewProfiles(ms.orig, ms.state))
}
