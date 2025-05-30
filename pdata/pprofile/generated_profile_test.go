// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/data"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProfile_MoveTo(t *testing.T) {
	ms := generateTestProfile()
	dest := NewProfile()
	ms.MoveTo(dest)
	assert.Equal(t, NewProfile(), ms)
	assert.Equal(t, generateTestProfile(), dest)
	dest.MoveTo(dest)
	assert.Equal(t, generateTestProfile(), dest)
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { ms.MoveTo(newProfile(&otlpprofiles.Profile{}, &sharedState)) })
	assert.Panics(t, func() { newProfile(&otlpprofiles.Profile{}, &sharedState).MoveTo(dest) })
}

func TestProfile_CopyTo(t *testing.T) {
	ms := NewProfile()
	orig := NewProfile()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = generateTestProfile()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { ms.CopyTo(newProfile(&otlpprofiles.Profile{}, &sharedState)) })
}

func TestProfile_SampleType(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewValueTypeSlice(), ms.SampleType())
	fillTestValueTypeSlice(ms.SampleType())
	assert.Equal(t, generateTestValueTypeSlice(), ms.SampleType())
}

func TestProfile_Sample(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewSampleSlice(), ms.Sample())
	fillTestSampleSlice(ms.Sample())
	assert.Equal(t, generateTestSampleSlice(), ms.Sample())
}

func TestProfile_MappingTable(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewMappingSlice(), ms.MappingTable())
	fillTestMappingSlice(ms.MappingTable())
	assert.Equal(t, generateTestMappingSlice(), ms.MappingTable())
}

func TestProfile_LocationTable(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewLocationSlice(), ms.LocationTable())
	fillTestLocationSlice(ms.LocationTable())
	assert.Equal(t, generateTestLocationSlice(), ms.LocationTable())
}

func TestProfile_LocationIndices(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.NewInt32Slice(), ms.LocationIndices())
	internal.FillTestInt32Slice(internal.Int32Slice(ms.LocationIndices()))
	assert.Equal(t, pcommon.Int32Slice(internal.GenerateTestInt32Slice()), ms.LocationIndices())
}

func TestProfile_FunctionTable(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewFunctionSlice(), ms.FunctionTable())
	fillTestFunctionSlice(ms.FunctionTable())
	assert.Equal(t, generateTestFunctionSlice(), ms.FunctionTable())
}

func TestProfile_AttributeTable(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewAttributeTableSlice(), ms.AttributeTable())
	fillTestAttributeTableSlice(ms.AttributeTable())
	assert.Equal(t, generateTestAttributeTableSlice(), ms.AttributeTable())
}

func TestProfile_AttributeUnits(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewAttributeUnitSlice(), ms.AttributeUnits())
	fillTestAttributeUnitSlice(ms.AttributeUnits())
	assert.Equal(t, generateTestAttributeUnitSlice(), ms.AttributeUnits())
}

func TestProfile_LinkTable(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, NewLinkSlice(), ms.LinkTable())
	fillTestLinkSlice(ms.LinkTable())
	assert.Equal(t, generateTestLinkSlice(), ms.LinkTable())
}

func TestProfile_StringTable(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.NewStringSlice(), ms.StringTable())
	internal.FillTestStringSlice(internal.StringSlice(ms.StringTable()))
	assert.Equal(t, pcommon.StringSlice(internal.GenerateTestStringSlice()), ms.StringTable())
}

func TestProfile_Time(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.Timestamp(0), ms.Time())
	testValTime := pcommon.Timestamp(1234567890)
	ms.SetTime(testValTime)
	assert.Equal(t, testValTime, ms.Time())
}

func TestProfile_Duration(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.Timestamp(0), ms.Duration())
	testValDuration := pcommon.Timestamp(1234567890)
	ms.SetDuration(testValDuration)
	assert.Equal(t, testValDuration, ms.Duration())
}

func TestProfile_StartTime(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.Timestamp(0), ms.StartTime())
	testValStartTime := pcommon.Timestamp(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.Equal(t, testValStartTime, ms.StartTime())
}

func TestProfile_PeriodType(t *testing.T) {
	ms := NewProfile()
	fillTestValueType(ms.PeriodType())
	assert.Equal(t, generateTestValueType(), ms.PeriodType())
}

func TestProfile_Period(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, int64(0), ms.Period())
	ms.SetPeriod(int64(1))
	assert.Equal(t, int64(1), ms.Period())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { newProfile(&otlpprofiles.Profile{}, &sharedState).SetPeriod(int64(1)) })
}

func TestProfile_CommentStrindices(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.NewInt32Slice(), ms.CommentStrindices())
	internal.FillTestInt32Slice(internal.Int32Slice(ms.CommentStrindices()))
	assert.Equal(t, pcommon.Int32Slice(internal.GenerateTestInt32Slice()), ms.CommentStrindices())
}

func TestProfile_DefaultSampleTypeStrindex(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, int32(0), ms.DefaultSampleTypeStrindex())
	ms.SetDefaultSampleTypeStrindex(int32(1))
	assert.Equal(t, int32(1), ms.DefaultSampleTypeStrindex())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { newProfile(&otlpprofiles.Profile{}, &sharedState).SetDefaultSampleTypeStrindex(int32(1)) })
}

func TestProfile_ProfileID(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, ProfileID(data.ProfileID([16]byte{})), ms.ProfileID())
	testValProfileID := ProfileID(data.ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
	ms.SetProfileID(testValProfileID)
	assert.Equal(t, testValProfileID, ms.ProfileID())
}

func TestProfile_AttributeIndices(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.NewInt32Slice(), ms.AttributeIndices())
	internal.FillTestInt32Slice(internal.Int32Slice(ms.AttributeIndices()))
	assert.Equal(t, pcommon.Int32Slice(internal.GenerateTestInt32Slice()), ms.AttributeIndices())
}

func TestProfile_DroppedAttributesCount(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, uint32(0), ms.DroppedAttributesCount())
	ms.SetDroppedAttributesCount(uint32(17))
	assert.Equal(t, uint32(17), ms.DroppedAttributesCount())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { newProfile(&otlpprofiles.Profile{}, &sharedState).SetDroppedAttributesCount(uint32(17)) })
}

func TestProfile_OriginalPayloadFormat(t *testing.T) {
	ms := NewProfile()
	assert.Empty(t, ms.OriginalPayloadFormat())
	ms.SetOriginalPayloadFormat("original payload")
	assert.Equal(t, "original payload", ms.OriginalPayloadFormat())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { newProfile(&otlpprofiles.Profile{}, &sharedState).SetOriginalPayloadFormat("original payload") })
}

func TestProfile_OriginalPayload(t *testing.T) {
	ms := NewProfile()
	assert.Equal(t, pcommon.NewByteSlice(), ms.OriginalPayload())
	internal.FillTestByteSlice(internal.ByteSlice(ms.OriginalPayload()))
	assert.Equal(t, pcommon.ByteSlice(internal.GenerateTestByteSlice()), ms.OriginalPayload())
}

func generateTestProfile() Profile {
	tv := NewProfile()
	fillTestProfile(tv)
	return tv
}

func fillTestProfile(tv Profile) {
	fillTestValueTypeSlice(newValueTypeSlice(&tv.orig.SampleType, tv.state))
	fillTestSampleSlice(newSampleSlice(&tv.orig.Sample, tv.state))
	fillTestMappingSlice(newMappingSlice(&tv.orig.MappingTable, tv.state))
	fillTestLocationSlice(newLocationSlice(&tv.orig.LocationTable, tv.state))
	internal.FillTestInt32Slice(internal.NewInt32Slice(&tv.orig.LocationIndices, tv.state))
	fillTestFunctionSlice(newFunctionSlice(&tv.orig.FunctionTable, tv.state))
	fillTestAttributeTableSlice(newAttributeTableSlice(&tv.orig.AttributeTable, tv.state))
	fillTestAttributeUnitSlice(newAttributeUnitSlice(&tv.orig.AttributeUnits, tv.state))
	fillTestLinkSlice(newLinkSlice(&tv.orig.LinkTable, tv.state))
	internal.FillTestStringSlice(internal.NewStringSlice(&tv.orig.StringTable, tv.state))
	tv.orig.TimeNanos = 1234567890
	tv.orig.DurationNanos = 1234567890
	tv.orig.TimeNanos = 1234567890
	fillTestValueType(newValueType(&tv.orig.PeriodType, tv.state))
	tv.orig.Period = int64(1)
	internal.FillTestInt32Slice(internal.NewInt32Slice(&tv.orig.CommentStrindices, tv.state))
	tv.orig.DefaultSampleTypeStrindex = int32(1)
	tv.orig.ProfileId = data.ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	internal.FillTestInt32Slice(internal.NewInt32Slice(&tv.orig.AttributeIndices, tv.state))
	tv.orig.DroppedAttributesCount = uint32(17)
	tv.orig.OriginalPayloadFormat = "original payload"
	internal.FillTestByteSlice(internal.NewByteSlice(&tv.orig.OriginalPayload, tv.state))
}
