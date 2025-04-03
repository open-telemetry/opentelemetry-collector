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
	protoDeltaSizer
}

var _ ProfilesSizer = (*ProfilesBytesSizer)(nil)

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
