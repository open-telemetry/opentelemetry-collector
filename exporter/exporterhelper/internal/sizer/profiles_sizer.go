// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	"go.opentelemetry.io/collector/pdata/pprofile"
)

type ProfilesSizer interface {
	ProfilesSize(pd pprofile.Profiles) int
	ResourceProfilesSize(rp pprofile.ResourceProfiles) int
	ScopeProfilesSize(sp pprofile.ScopeProfiles) int
	ProfileSize(p pprofile.Profile) int
	DeltaSize(newItemSize int) int
}

// TracesBytesSizer returns the byte size of serialized protos.
type ProfilesBytesSizer struct {
	pprofile.ProtoMarshaler
}

var _ ProfilesSizer = (*ProfilesBytesSizer)(nil)

// DeltaSize() returns the delta size of a proto slice when a new item is added.
// Example:
//
//	prevSize := proto1.Size()
//	proto1.RepeatedField().AppendEmpty() = proto2
//
// Then currSize of proto1 can be calculated as
//
//	currSize := (prevSize + sizer.DeltaSize(proto2.Size()))
//
// This is derived from pdata/internal/data/protogen/profiles/v1development/profiles.pb.go
// which is generated with gogo/protobuf.
func (s *ProfilesBytesSizer) DeltaSize(newItemSize int) int {
	return 1 + newItemSize + sov(uint64(newItemSize)) //nolint:gosec // disable G115
}

// ProfilesCountSizer returns the number of profiles in the profiles.
type ProfilesCountSizer struct{}

var _ ProfilesSizer = (*ProfilesCountSizer)(nil)

func (s *ProfilesCountSizer) ProfilesSize(pd pprofile.Profiles) int {
	return pd.SampleCount()
}

func (s *ProfilesCountSizer) ResourceProfilesSize(rp pprofile.ResourceProfiles) int {
	count := 0
	for k := 0; k < rp.ScopeProfiles().Len(); k++ {
		count += rp.ScopeProfiles().At(k).Profiles().Len()
	}
	return count
}

func (s *ProfilesCountSizer) ScopeProfilesSize(sp pprofile.ScopeProfiles) int {
	return sp.Profiles().Len()
}

func (s *ProfilesCountSizer) ProfileSize(_ pprofile.Profile) int {
	return 1
}

func (s *ProfilesCountSizer) DeltaSize(newItemSize int) int {
	return newItemSize
}
