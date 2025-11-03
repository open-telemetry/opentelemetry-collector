// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp // import "go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
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
func (ms ExportRequest) MarshalProto() ([]byte, error) {
	size := ms.orig.SizeProto()
	buf := make([]byte, size)
	_ = ms.orig.MarshalProto(buf)
	return buf, nil
}

// UnmarshalProto unmarshalls ExportRequest from proto bytes.
func (ms ExportRequest) UnmarshalProto(data []byte) error {
	err := ms.orig.UnmarshalProto(data)
	if err != nil {
		return err
	}
	otlp.MigrateProfiles(ms.orig.ResourceProfiles)
	return nil
}

// MarshalJSON marshals ExportRequest into JSON bytes.
func (ms ExportRequest) MarshalJSON() ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	ms.orig.MarshalJSON(dest)
	if dest.Error() != nil {
		return nil, dest.Error()
	}
	return slices.Clone(dest.Buffer()), nil
}

// UnmarshalJSON unmarshalls ExportRequest from JSON bytes.
func (ms ExportRequest) UnmarshalJSON(data []byte) error {
	iter := json.BorrowIterator(data)
	defer json.ReturnIterator(iter)
	ms.orig.UnmarshalJSON(iter)
	return iter.Error()
}

func (ms ExportRequest) Profiles() pprofile.Profiles {
	return pprofile.Profiles(internal.NewProfilesWrapper(ms.orig, ms.state))
}
