// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp // import "go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// ExportRequest represents the request for gRPC/HTTP client/server.
// It's a wrapper for pprofile.Profiles data.
type ExportRequest struct {
	orig  *internal.ExportProfilesServiceRequest
	state *internal.State
}

// NewExportRequest returns an empty ExportRequest.
func NewExportRequest() ExportRequest {
	return ExportRequest{
		orig:  &internal.ExportProfilesServiceRequest{},
		state: internal.NewState(),
	}
}

// NewExportRequestFromProfiles returns a ExportRequest from pprofile.Profiles.
// Because ExportRequest is a wrapper for pprofile.Profiles,
// any changes to the provided Profiles struct will be reflected in the ExportRequest and vice versa.
func NewExportRequestFromProfiles(td pprofile.Profiles) ExportRequest {
	return ExportRequest{
		orig:  internal.GetProfilesOrig(internal.ProfilesWrapper(td)),
		state: internal.GetProfilesState(internal.ProfilesWrapper(td)),
	}
}

// MarshalProto marshals ExportRequest into proto bytes.
// Delegates to pprofile.ProtoMarshaler so attribute strings are referenced via
// the ProfilesDictionary string table.
func (ms ExportRequest) MarshalProto() ([]byte, error) {
	return (&pprofile.ProtoMarshaler{}).MarshalProfiles(ms.Profiles())
}

// UnmarshalProto unmarshalls ExportRequest from proto bytes.
// Delegates to pprofile.ProtoUnmarshaler so string-table references are resolved.
func (ms ExportRequest) UnmarshalProto(data []byte) error {
	pd, err := (&pprofile.ProtoUnmarshaler{}).UnmarshalProfiles(data)
	if err != nil {
		return err
	}
	pd.MoveTo(ms.Profiles())
	return nil
}

// MarshalJSON marshals ExportRequest into JSON bytes.
func (ms ExportRequest) MarshalJSON() ([]byte, error) {
	return (&pprofile.JSONMarshaler{}).MarshalProfiles(ms.Profiles())
}

// UnmarshalJSON unmarshalls ExportRequest from JSON bytes.
func (ms ExportRequest) UnmarshalJSON(data []byte) error {
	pd, err := (&pprofile.JSONUnmarshaler{}).UnmarshalProfiles(data)
	if err != nil {
		return err
	}
	pd.MoveTo(ms.Profiles())
	return nil
}

func (ms ExportRequest) Profiles() pprofile.Profiles {
	return pprofile.Profiles(internal.NewProfilesWrapper(ms.orig, ms.state))
}
