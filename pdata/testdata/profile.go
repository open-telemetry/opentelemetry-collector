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

	profile.Profile().LocationIndices().FromRaw([]int64{1})

	sample := profile.Profile().Sample().AppendEmpty()
	sample.LocationIndex().Append(1)
	sample.SetLocationsStartIndex(2)
	sample.SetLocationsLength(10)
	sample.SetStacktraceIdIndex(3)
	sample.Value().Append(4)
	sample.SetLink(42)
	sample.Attributes().Append(5)

	label := sample.Label().AppendEmpty()
	label.SetKey(1)
	label.SetStr(2)
	label.SetNum(3)
	label.SetNumUnit(4)

	location := profile.Profile().Location().AppendEmpty()
	location.SetID(2)
	location.SetMappingIndex(3)
	location.SetAddress(4)
	line := location.Line().AppendEmpty()
	line.SetFunctionIndex(1)
	line.SetLine(2)
	line.SetColumn(3)
	location.SetIsFolded(true)
	location.SetTypeIndex(5)
	location.Attributes().FromRaw([]uint64{6, 7})

	profile.Profile().AttributeTable().FromRaw(map[string]any{
		"key": "answer",
		"value": map[string]any{
			"intValue": "42",
		},
	})

	attributeUnits := profile.Profile().AttributeUnits().AppendEmpty()
	attributeUnits.SetAttributeKey(1)
	attributeUnits.SetUnit(5)

	profile.Profile().StringTable().Append("foobar")
}

func fillProfileTwo(profile pprofile.ProfileContainer) {
	profile.SetProfileID([16]byte{0x02, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	profile.SetStartTime(profileStartTimestamp)
	profile.SetEndTime(profileEndTimestamp)

	profile.Profile().SetDropFrames(1)
	profile.Profile().SetKeepFrames(2)

	sample := profile.Profile().Sample().AppendEmpty()
	sample.LocationIndex().Append(6)
	sample.SetLocationsStartIndex(7)
	sample.SetLocationsLength(20)
	sample.SetStacktraceIdIndex(8)
	sample.Value().Append(9)
	sample.SetLink(44)
	sample.Attributes().Append(10)

	mapping := profile.Profile().Mapping().AppendEmpty()
	mapping.SetID(1)
	mapping.SetMemoryStart(2)
	mapping.SetMemoryLimit(3)
	mapping.SetFileOffset(4)
	mapping.SetFilename(5)
	mapping.SetBuildID(6)
	mapping.Attributes().FromRaw([]uint64{7, 8})
	mapping.SetHasFunctions(true)
	mapping.SetHasFilenames(true)
	mapping.SetHasLineNumbers(true)
	mapping.SetHasInlineFrames(true)

	function := profile.Profile().Function().AppendEmpty()
	function.SetID(1)
	function.SetName(2)
	function.SetSystemName(3)
	function.SetFilename(4)
	function.SetStartLine(5)

	linkTable := profile.Profile().LinkTable().AppendEmpty()
	linkTable.SetTraceID([16]byte{0x03, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	linkTable.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})

	profile.Profile().Comment().FromRaw([]int64{1, 2})
}
