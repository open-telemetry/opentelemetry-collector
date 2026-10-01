// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package sizer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestProfilesCountSizer(t *testing.T) {
	// Three profiles carrying five samples each, so the sample count and the
	// profile count differ and the unit the sizer reports is unambiguous.
	td := testdata.GenerateProfilesMultiSample(3, 5)
	sizer := ProfilesCountSizer{}
	require.Equal(t, 15, sizer.ProfilesSize(td))

	rp := td.ResourceProfiles().At(0)
	require.Equal(t, 15, sizer.ResourceProfilesSize(rp))

	sp := rp.ScopeProfiles().At(0)
	require.Equal(t, 15, sizer.ScopeProfilesSize(sp))

	ps := sp.Profiles()
	require.Equal(t, 3, ps.Len())
	for k := 0; k < ps.Len(); k++ {
		require.Equal(t, 5, sizer.ProfileSize(ps.At(k)))
	}

	// The sizes must stay additive as profiles are added, since that is the
	// property the split loop relies on when it subtracts extracted sizes from
	// the remaining capacity.
	prevScopeSize := sizer.ScopeProfilesSize(sp)
	extra := ps.AppendEmpty()
	require.Equal(t, 0, sizer.ProfileSize(extra))
	prevScopeSize += sizer.ProfileSize(extra)
	require.Equal(t, prevScopeSize, sizer.ScopeProfilesSize(sp))

	require.Equal(t, prevScopeSize, sizer.ResourceProfilesSize(rp))
	require.Equal(t, prevScopeSize, sizer.ProfilesSize(td))
}

func TestProfilesCountSizerEmptyProfile(t *testing.T) {
	td := testdata.GenerateProfilesMultiSample(1, 2)
	sizer := ProfilesCountSizer{}

	rp := td.ResourceProfiles().At(0)
	sp := rp.ScopeProfiles().At(0)

	// A profile with no samples contributes nothing, so it must not be counted
	// as one unit.
	empty := sp.Profiles().AppendEmpty()
	require.Equal(t, 0, sizer.ProfileSize(empty))
	require.Equal(t, 2, sizer.ScopeProfilesSize(sp))
	require.Equal(t, 2, sizer.ProfilesSize(td))
}

func TestProfilesCountSizerLeafEqualsCountableUnit(t *testing.T) {
	// Each leaf must report its own contribution in samples so that summing the
	// levels is consistent with ProfilesSize, which reports SampleCount.
	td := testdata.GenerateProfilesMultiSample(2, 4)
	sizer := ProfilesCountSizer{}

	rp := td.ResourceProfiles().At(0)
	for k := 0; k < rp.ScopeProfiles().Len(); k++ {
		sp := rp.ScopeProfiles().At(k)
		ps := sp.Profiles()
		var leafSum int
		for j := 0; j < ps.Len(); j++ {
			leafSum += sizer.ProfileSize(ps.At(j))
		}
		require.Equal(t, sizer.ScopeProfilesSize(sp), leafSum)
	}
	require.Equal(t, sizer.ProfilesSize(td), sizer.ResourceProfilesSize(rp))
}

func TestProfilesCountSizerDoesNotCountProfiles(t *testing.T) {
	// Guards against regressing to counting profiles: with five samples per
	// profile, a profile-counting implementation returns 3 here.
	td := testdata.GenerateProfilesMultiSample(3, 5)
	sizer := &ProfilesCountSizer{}
	require.Equal(t, 3, td.ProfileCount())
	require.Equal(t, 15, td.SampleCount())
	require.NotEqual(t, sizer.ProfilesSize(td), td.ProfileCount())
}
