// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestProfilesCountSizer(t *testing.T) {
	const (
		profilesCount     = 3
		samplesPerProfile = 5
	)
	pd := testdata.GenerateProfilesWithSamples(profilesCount, samplesPerProfile)
	sizer := ProfilesCountSizer{}

	rp := pd.ResourceProfiles().At(0)
	sp := rp.ScopeProfiles().At(0)
	require.Equal(t, profilesCount*samplesPerProfile, sizer.ProfilesSize(pd))
	require.Equal(t, profilesCount*samplesPerProfile, sizer.ResourceProfilesSize(rp))
	require.Equal(t, profilesCount*samplesPerProfile, sizer.ScopeProfilesSize(sp))
	require.Equal(t, samplesPerProfile, sizer.ProfileSize(sp.Profiles().At(0)))

	prevSize := sizer.ScopeProfilesSize(sp)
	p := sp.Profiles().At(0)
	p.CopyTo(sp.Profiles().AppendEmpty())
	require.Equal(t, prevSize+sizer.DeltaSize(sizer.ProfileSize(p)), sizer.ScopeProfilesSize(sp))
}
