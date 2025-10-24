// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalProfiles(pd Profiles) ([]byte, error) {
	size := pd.getOrig().SizeProto()
	buf := make([]byte, size)
	_ = pd.getOrig().MarshalProto(buf)
	return buf, nil
}

func (e *ProtoMarshaler) ProfilesSize(pd Profiles) int {
	return pd.getOrig().SizeProto()
}

func (e *ProtoMarshaler) ResourceProfilesSize(pd ResourceProfiles) int {
	return pd.orig.SizeProto()
}

func (e *ProtoMarshaler) ScopeProfilesSize(pd ScopeProfiles) int {
	return pd.orig.SizeProto()
}

func (e *ProtoMarshaler) ProfileSize(pd Profile) int {
	return pd.orig.SizeProto()
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	pd := NewProfiles()
	err := pd.getOrig().UnmarshalProto(buf)
	if err != nil {
		return Profiles{}, err
	}
	return pd, nil
}
