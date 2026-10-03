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

// ProfilesSamplesCountSizer returns the number of samples in the profiles.
//
// Note that the per-element methods below count profiles, not samples. The two
// do not measure the same thing: a sample is a set of indices into the
// ProfilesDictionary, so a profile can hold many samples that reference shared
// dictionary entries without adding much to the serialized size. Batching on
// either count does not reliably bound the payload of a profiles signal.
type ProfilesSamplesCountSizer struct{}

var _ ProfilesSizer = (*ProfilesSamplesCountSizer)(nil)

func (s *ProfilesSamplesCountSizer) ProfilesSize(pd pprofile.Profiles) int {
	return pd.SampleCount()
}

func (s *ProfilesSamplesCountSizer) ResourceProfilesSize(rp pprofile.ResourceProfiles) int {
	count := 0
	for k := 0; k < rp.ScopeProfiles().Len(); k++ {
		count += rp.ScopeProfiles().At(k).Profiles().Len()
	}
	return count
}

func (s *ProfilesSamplesCountSizer) ScopeProfilesSize(sp pprofile.ScopeProfiles) int {
	return sp.Profiles().Len()
}

func (s *ProfilesSamplesCountSizer) ProfileSize(_ pprofile.Profile) int {
	return 1
}

func (s *ProfilesSamplesCountSizer) DeltaSize(newItemSize int) int {
	return newItemSize
}
