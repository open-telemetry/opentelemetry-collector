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

// ProfilesCountSizer returns the number of samples in the profiles.
//
// Every method counts samples, so the size of a batch is comparable at each
// level. Sample count does not bound the serialized size of a profiles signal:
// a sample holds indices into the ProfilesDictionary, so many samples can
// reference the same dictionary entries. Use ProfilesBytesSizer when the limit
// must bound the payload size.
type ProfilesCountSizer struct{}

var _ ProfilesSizer = (*ProfilesCountSizer)(nil)

func (s *ProfilesCountSizer) ProfilesSize(pd pprofile.Profiles) int {
	return pd.SampleCount()
}

func (s *ProfilesCountSizer) ResourceProfilesSize(rp pprofile.ResourceProfiles) int {
	count := 0
	for k := 0; k < rp.ScopeProfiles().Len(); k++ {
		count += s.ScopeProfilesSize(rp.ScopeProfiles().At(k))
	}
	return count
}

func (s *ProfilesCountSizer) ScopeProfilesSize(sp pprofile.ScopeProfiles) int {
	count := 0
	for k := 0; k < sp.Profiles().Len(); k++ {
		count += s.ProfileSize(sp.Profiles().At(k))
	}
	return count
}

func (s *ProfilesCountSizer) ProfileSize(p pprofile.Profile) int {
	return p.Samples().Len()
}

func (s *ProfilesCountSizer) DeltaSize(newItemSize int) int {
	return newItemSize
}
