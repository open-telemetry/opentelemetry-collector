// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package sizer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestProfilesCountSizer(t *testing.T) {
	pd := testdata.GenerateProfiles(5)
	sizer := ProfilesCountSizer{}
	require.Equal(t, 5, sizer.ProfilesSize(pd))

	rp := pd.ResourceProfiles().At(0)
	require.Equal(t, 5, sizer.ResourceProfilesSize(rp))

	sp := rp.ScopeProfiles().At(0)
	require.Equal(t, 5, sizer.ScopeProfilesSize(sp))

	require.Equal(t, 1, sizer.ProfileSize(sp.Profiles().At(0)))
	require.Equal(t, 1, sizer.ProfileSize(sp.Profiles().At(1)))
	require.Equal(t, 1, sizer.ProfileSize(sp.Profiles().At(2)))
	require.Equal(t, 1, sizer.ProfileSize(sp.Profiles().At(3)))
	require.Equal(t, 1, sizer.ProfileSize(sp.Profiles().At(4)))

	prevSize := sizer.ScopeProfilesSize(sp)
	p := sp.Profiles().At(2)
	p.CopyTo(sp.Profiles().AppendEmpty())
	require.Equal(t, sizer.ScopeProfilesSize(sp), prevSize+sizer.DeltaSize(sizer.ProfileSize(p)))
}

func TestProfilesBytesSizer(t *testing.T) {
	pd := testdata.GenerateProfiles(5)
	sizer := ProfilesBytesSizer{}
	totalSize := sizer.ProfilesSize(pd)
	require.Positive(t, totalSize, 0)

	rp := pd.ResourceProfiles().At(0)
	resourceSize := sizer.ResourceProfilesSize(rp)
	require.Positive(t, resourceSize)

	sp := rp.ScopeProfiles().At(0)
	scopeSize := sizer.ScopeProfilesSize(sp)
	require.Positive(t, scopeSize)

	profile0Size := sizer.ProfileSize(sp.Profiles().At(0))
	require.Positive(t, profile0Size)

	prevSize := sizer.ScopeProfilesSize(sp)
	p := sp.Profiles().At(2)
	p.CopyTo(sp.Profiles().AppendEmpty())
	require.Equal(t, sizer.ScopeProfilesSize(sp), prevSize+sizer.DeltaSize(sizer.ProfileSize(p)))
}
