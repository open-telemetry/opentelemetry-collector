// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalProfiles(pd Profiles) ([]byte, error) {
	size := internal.SizeProtoOrigExportProfilesServiceRequest(pd.getOrig())
	buf := make([]byte, size)
	_ = internal.MarshalProtoOrigExportProfilesServiceRequest(pd.getOrig(), buf)
	return buf, nil
}

func (e *ProtoMarshaler) ProfilesSize(pd Profiles) int {
	return internal.SizeProtoOrigExportProfilesServiceRequest(pd.getOrig())
}

func (e *ProtoMarshaler) ResourceProfilesSize(pd ResourceProfiles) int {
	return internal.SizeProtoOrigResourceProfiles(pd.orig)
}

func (e *ProtoMarshaler) ScopeProfilesSize(pd ScopeProfiles) int {
	return internal.SizeProtoOrigScopeProfiles(pd.orig)
}

func (e *ProtoMarshaler) ProfileSize(pd Profile) int {
	return internal.SizeProtoOrigProfile(pd.orig)
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	pd := NewProfiles()
	err := internal.UnmarshalProtoOrigExportProfilesServiceRequest(pd.getOrig(), buf)
	if err != nil {
		return Profiles{}, err
	}
	return pd, nil
}
