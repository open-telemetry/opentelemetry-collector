// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestSample_MoveTo(t *testing.T) {
	ms := generateTestSample()
	dest := NewSample()
	ms.MoveTo(dest)
	assert.Equal(t, NewSample(), ms)
	assert.Equal(t, generateTestSample(), dest)
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { ms.MoveTo(newSample(&otlpprofiles.Sample{}, &sharedState)) })
	assert.Panics(t, func() { newSample(&otlpprofiles.Sample{}, &sharedState).MoveTo(dest) })
}

func TestSample_CopyTo(t *testing.T) {
	ms := NewSample()
	orig := NewSample()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	orig = generateTestSample()
	orig.CopyTo(ms)
	assert.Equal(t, orig, ms)
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { ms.CopyTo(newSample(&otlpprofiles.Sample{}, &sharedState)) })
}

func TestSample_LocationsStartIndex(t *testing.T) {
	ms := NewSample()
	assert.Equal(t, int32(0), ms.LocationsStartIndex())
	ms.SetLocationsStartIndex(int32(1))
	assert.Equal(t, int32(1), ms.LocationsStartIndex())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { newSample(&otlpprofiles.Sample{}, &sharedState).SetLocationsStartIndex(int32(1)) })
}

func TestSample_LocationsLength(t *testing.T) {
	ms := NewSample()
	assert.Equal(t, int32(0), ms.LocationsLength())
	ms.SetLocationsLength(int32(1))
	assert.Equal(t, int32(1), ms.LocationsLength())
	sharedState := internal.StateReadOnly
	assert.Panics(t, func() { newSample(&otlpprofiles.Sample{}, &sharedState).SetLocationsLength(int32(1)) })
}

func TestSample_Value(t *testing.T) {
	ms := NewSample()
	assert.Equal(t, pcommon.NewInt64Slice(), ms.Value())
	internal.FillTestInt64Slice(internal.Int64Slice(ms.Value()))
	assert.Equal(t, pcommon.Int64Slice(internal.GenerateTestInt64Slice()), ms.Value())
}

func TestSample_AttributeIndices(t *testing.T) {
	ms := NewSample()
	assert.Equal(t, pcommon.NewInt32Slice(), ms.AttributeIndices())
	internal.FillTestInt32Slice(internal.Int32Slice(ms.AttributeIndices()))
	assert.Equal(t, pcommon.Int32Slice(internal.GenerateTestInt32Slice()), ms.AttributeIndices())
}

func TestSample_LinkIndex(t *testing.T) {
	ms := NewSample()
	assert.Equal(t, int32(0), ms.LinkIndex())
	ms.SetLinkIndex(int32(1))
	assert.True(t, ms.HasLinkIndex())
	assert.Equal(t, int32(1), ms.LinkIndex())
	ms.RemoveLinkIndex()
	assert.False(t, ms.HasLinkIndex())
}

func TestSample_TimestampsUnixNano(t *testing.T) {
	ms := NewSample()
	assert.Equal(t, pcommon.NewUInt64Slice(), ms.TimestampsUnixNano())
	internal.FillTestUInt64Slice(internal.UInt64Slice(ms.TimestampsUnixNano()))
	assert.Equal(t, pcommon.UInt64Slice(internal.GenerateTestUInt64Slice()), ms.TimestampsUnixNano())
}

func generateTestSample() Sample {
	tv := NewSample()
	fillTestSample(tv)
	return tv
}

func fillTestSample(tv Sample) {
	tv.orig.LocationsStartIndex = int32(1)
	tv.orig.LocationsLength = int32(1)
	internal.FillTestInt64Slice(internal.NewInt64Slice(&tv.orig.Value, tv.state))
	internal.FillTestInt32Slice(internal.NewInt32Slice(&tv.orig.AttributeIndices, tv.state))
	tv.orig.LinkIndex_ = &otlpprofiles.Sample_LinkIndex{LinkIndex: int32(1)}
	internal.FillTestUInt64Slice(internal.NewUInt64Slice(&tv.orig.TimestampsUnixNano, tv.state))
}
