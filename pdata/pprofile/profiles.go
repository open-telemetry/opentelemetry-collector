// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1"
)

// Profiles is the top-level struct that is propagated through the profiles pipeline.
// Use NewProfiles to create new instance, zero-initialized instance is not valid for use.
type Profiles internal.Profiles

func newProfiles(orig *otlpcollectorprofile.ExportProfilesServiceRequest) Profiles {
	return Profiles(internal.NewProfiles(orig))
}

func (ms Profiles) getOrig() *otlpcollectorprofile.ExportProfilesServiceRequest {
	return internal.GetOrigProfiles(internal.Profiles(ms))
}

// NewProfiles creates a new Profiles struct.
func NewProfiles() Profiles {
	return newProfiles(&otlpcollectorprofile.ExportProfilesServiceRequest{})
}

// CopyTo copies the Profiles instance overriding the destination.
func (ms Profiles) CopyTo(dest Profiles) {
	ms.ResourceProfiles().CopyTo(dest.ResourceProfiles())
}

// ProfileRecordCount calculates the total number of profile records.
func (ms Profiles) ProfileRecordCount() int {
	profileCount := 0
	rss := ms.ResourceProfiles()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ill := rs.ScopeProfiles()
		for i := 0; i < ill.Len(); i++ {
			profiles := ill.At(i)
			profileCount += profiles.Profiles().Len()
		}
	}
	return profileCount
}

// ResourceProfiles returns the ResourceProfilesSlice associated with this Profiles.
func (ms Profiles) ResourceProfiles() ResourceProfilesSlice {
	return newResourceProfilesSlice(&ms.getOrig().ResourceProfiles)
}
