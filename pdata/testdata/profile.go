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
	profile.SetProfileID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	profile.SetStartTime(profileStartTimestamp)
	profile.SetEndTime(profileEndTimestamp)
	profile.SetDroppedAttributesCount(1)

	sample := profile.Profile().Sample().AppendEmpty()
	sample.LocationIndex().Append(1)
	sample.SetLocationsStartIndex(2)
	sample.SetLocationsLength(10)
	sample.SetStacktraceIdIndex(3)
	sample.Value().Append(4)
	sample.SetLink(42)
	sample.Attributes().Append(5)
}

func fillProfileTwo(profile pprofile.ProfileContainer) {
	profile.SetProfileID([16]byte{0x02, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	profile.SetStartTime(profileStartTimestamp)
	profile.SetEndTime(profileEndTimestamp)

	sample := profile.Profile().Sample().AppendEmpty()
	sample.LocationIndex().Append(6)
	sample.SetLocationsStartIndex(7)
	sample.SetLocationsLength(20)
	sample.SetStacktraceIdIndex(8)
	sample.Value().Append(9)
	sample.SetLink(44)
	sample.Attributes().Append(10)
}
