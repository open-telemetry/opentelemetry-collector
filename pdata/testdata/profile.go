// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var (
	profileTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC))
)

func GenerateProfiles(count int) pprofile.Profiles {
	ld := pprofile.NewProfiles()
	initResource(ld.ResourceProfiles().AppendEmpty().Resource())
	profiles := ld.ResourceProfiles().At(0).ScopeProfiles().AppendEmpty().Profiles()
	profiles.EnsureCapacity(count)
	for i := 0; i < count; i++ {
		switch i % 2 {
		case 0:
			fillProfileOne(profiles.AppendEmpty())
		case 1:
			fillProfileTwo(profiles.AppendEmpty())
		}
	}
	return ld
}

func fillProfileOne(profile pprofile.Profile) {
	profile.SetStartTime(profileTimestamp)
	profile.SetEndTime(profileTimestamp)
	profile.SetProfileID([16]byte{0x08, 0x04, 0x02, 0x01})

	attrs := profile.Attributes()
	attrs.PutStr("app", "server")
	attrs.PutInt("instance_num", 1)
}

func fillProfileTwo(profile pprofile.Profile) {
	profile.SetStartTime(profileTimestamp)
	profile.SetEndTime(profileTimestamp)
	profile.SetDroppedAttributesCount(1)

	attrs := profile.Attributes()
	attrs.PutStr("customer", "acme")
	attrs.PutStr("env", "dev")
}
