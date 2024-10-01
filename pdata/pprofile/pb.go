// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1experimental"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalProfiles(td Profiles) ([]byte, error) {
	pb := internal.ProfilesToProto(internal.Profiles(td))
	return pb.Marshal()
}

func (e *ProtoMarshaler) ProfilesSize(td Profiles) int {
	pb := internal.ProfilesToProto(internal.Profiles(td))
	return pb.Size()
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	pb := otlpprofile.ProfilesData{}
	err := pb.Unmarshal(buf)
	return Profiles(internal.ProfilesFromProto(pb)), err
}
