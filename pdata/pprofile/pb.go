// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalProfiles(pd Profiles) ([]byte, error) {
	size := internal.SizeProtoExportProfilesServiceRequest(pd.getOrig())
	buf := make([]byte, size)
	_ = internal.MarshalProtoExportProfilesServiceRequest(pd.getOrig(), buf)
	return buf, nil
}

func (e *ProtoMarshaler) ProfilesSize(pd Profiles) int {
	return internal.SizeProtoExportProfilesServiceRequest(pd.getOrig())
}

func (e *ProtoMarshaler) ResourceProfilesSize(pd ResourceProfiles) int {
	return internal.SizeProtoResourceProfiles(pd.orig)
}

func (e *ProtoMarshaler) ScopeProfilesSize(pd ScopeProfiles) int {
	return internal.SizeProtoScopeProfiles(pd.orig)
}

func (e *ProtoMarshaler) ProfileSize(pd Profile) int {
	return internal.SizeProtoProfile(pd.orig)
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	pd := NewProfiles()
	err := internal.UnmarshalProtoExportProfilesServiceRequest(pd.getOrig(), buf)
	if err != nil {
		return Profiles{}, err
	}
	return pd, nil
}
