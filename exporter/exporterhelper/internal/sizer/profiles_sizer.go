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
//
// Note that the per-element methods below do not report the same unit as
// ProfilesSize. ProfilesSize returns the number of samples, while
// ResourceProfilesSize, ScopeProfilesSize, and ProfileSize count profiles.
// Neither unit reliably bounds the size of a profiles signal: a sample is a set
// of indices into the ProfilesDictionary, so a profile can hold many samples
// that reference shared dictionary entries without adding much to the
// serialized size. There is no way to tell from the counts alone whether an
// index is reused or unique.
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
