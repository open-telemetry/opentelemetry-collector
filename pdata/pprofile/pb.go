// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalProfiles(ld Profiles) ([]byte, error) {
	pb := internal.ProfilesToProto(internal.Profiles(ld))
	return pb.Marshal()
}

func (e *ProtoMarshaler) ProfilesSize(ld Profiles) int {
	pb := internal.ProfilesToProto(internal.Profiles(ld))
	return pb.Size()
}

var _ Unmarshaler = (*ProtoUnmarshaler)(nil)

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	pb := otlpprofiles.ProfilesData{}
	err := pb.Unmarshal(buf)
	return Profiles(internal.ProfilesFromProto(pb)), err
}
