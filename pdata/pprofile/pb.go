// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalProfiles(pd Profiles) ([]byte, error) {
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		return pd.getOrig().Marshal()
	}
	size := internal.SizeProtoOrigExportProfilesServiceRequest(pd.getOrig())
	buf := make([]byte, size)
	_ = internal.MarshalProtoOrigExportProfilesServiceRequest(pd.getOrig(), buf)
	return buf, nil
}

func (e *ProtoMarshaler) ProfilesSize(pd Profiles) int {
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		return pd.getOrig().Size()
	}
	return internal.SizeProtoOrigExportProfilesServiceRequest(pd.getOrig())
}

func (e *ProtoMarshaler) ResourceProfilesSize(pd ResourceProfiles) int {
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		return pd.orig.Size()
	}
	return internal.SizeProtoOrigResourceProfiles(pd.orig)
}

func (e *ProtoMarshaler) ScopeProfilesSize(pd ScopeProfiles) int {
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		return pd.orig.Size()
	}
	return internal.SizeProtoOrigScopeProfiles(pd.orig)
}

func (e *ProtoMarshaler) ProfileSize(pd Profile) int {
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		return pd.orig.Size()
	}
	return internal.SizeProtoOrigProfile(pd.orig)
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	pd := NewProfiles()
	if !internal.UseCustomProtoEncoding.IsEnabled() {
		err := pd.getOrig().Unmarshal(buf)
		if err != nil {
			return Profiles{}, err
		}
		return pd, nil
	}
	err := internal.UnmarshalProtoOrigExportProfilesServiceRequest(pd.getOrig(), buf)
	if err != nil {
		return Profiles{}, err
	}
	return pd, nil
}
