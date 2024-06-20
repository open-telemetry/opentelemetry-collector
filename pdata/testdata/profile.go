// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata // import "go.opentelemetry.io/collector/pdata/testdata"

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var (
	profileStartTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC))
	profileEndTimestamp   = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC))
)

// GenerateProfiles generates dummy profiling data for tests
func GenerateProfiles(profilesCount int) pprofile.Profiles {
	td := pprofile.NewProfiles()
	initResource(td.ResourceProfiles().AppendEmpty().Resource())
	ss := td.ResourceProfiles().At(0).ScopeProfiles().AppendEmpty().Profiles()
	ss.EnsureCapacity(profilesCount)
	for i := 0; i < profilesCount; i++ {
		switch i % 2 {
		case 0:
			fillProfileOne(ss.AppendEmpty())
		case 1:
			fillProfileTwo(ss.AppendEmpty())
		}
	}
	return td
}

func fillProfileOne(profile pprofile.ProfileContainer) {
	profile.ProfileID().FromRaw([]byte("profileA"))
	profile.SetStartTime(profileStartTimestamp)
	profile.SetEndTime(profileEndTimestamp)
	profile.SetDroppedAttributesCount(1)
}

func fillProfileTwo(profile pprofile.ProfileContainer) {
	profile.ProfileID().FromRaw([]byte("profileB"))
	profile.SetStartTime(profileStartTimestamp)
	profile.SetEndTime(profileEndTimestamp)
}
