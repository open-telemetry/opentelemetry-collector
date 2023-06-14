// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlpcollectorprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"
)

func TestProfileRecordCount(t *testing.T) {
	profiles := NewProfiles()
	assert.EqualValues(t, 0, profiles.ProfileRecordCount())

	rl := profiles.ResourceProfiles().AppendEmpty()
	assert.EqualValues(t, 0, profiles.ProfileRecordCount())

	ill := rl.ScopeProfiles().AppendEmpty()
	assert.EqualValues(t, 0, profiles.ProfileRecordCount())

	ill.ProfileRecords().AppendEmpty()
	assert.EqualValues(t, 1, profiles.ProfileRecordCount())

	rms := profiles.ResourceProfiles()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeProfiles().AppendEmpty()
	illl := rms.AppendEmpty().ScopeProfiles().AppendEmpty().ProfileRecords()
	for i := 0; i < 5; i++ {
		illl.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.EqualValues(t, 6, profiles.ProfileRecordCount())
}

func TestProfileRecordCountWithEmpty(t *testing.T) {
	assert.Zero(t, NewProfiles().ProfileRecordCount())
	assert.Zero(t, newProfiles(&otlpcollectorprofile.ExportProfilesServiceRequest{
		ResourceProfiles: []*otlpprofiles.ResourceProfiles{{}},
	}).ProfileRecordCount())
	assert.Zero(t, newProfiles(&otlpcollectorprofile.ExportProfilesServiceRequest{
		ResourceProfiles: []*otlpprofiles.ResourceProfiles{
			{
				ScopeProfiles: []*otlpprofiles.ScopeProfiles{{}},
			},
		},
	}).ProfileRecordCount())
	assert.Equal(t, 1, newProfiles(&otlpcollectorprofile.ExportProfilesServiceRequest{
		ResourceProfiles: []*otlpprofiles.ResourceProfiles{
			{
				ScopeProfiles: []*otlpprofiles.ScopeProfiles{
					{
						ProfileRecords: []*otlpprofiles.ProfileRecord{{}},
					},
				},
			},
		},
	}).ProfileRecordCount())
}

func TestToFromProfileOtlp(t *testing.T) {
	otlp := &otlpcollectorprofile.ExportProfilesServiceRequest{}
	profiles := newProfiles(otlp)
	assert.EqualValues(t, NewProfiles(), profiles)
	assert.EqualValues(t, otlp, profiles.getOrig())
}

func TestResourceProfilesWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceProfiles as pdata struct.
	profiles := NewProfiles()
	fillTestResourceProfilesSlice(profiles.ResourceProfiles())

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(profiles.getOrig())
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage emptypb.Empty
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRS2 otlpcollectorprofile.ExportProfilesServiceRequest
	err = gogoproto.Unmarshal(wire2, &gogoprotoRS2)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.EqualValues(t, profiles.getOrig(), &gogoprotoRS2)
}

func TestProfilesCopyTo(t *testing.T) {
	profiles := NewProfiles()
	fillTestResourceProfilesSlice(profiles.ResourceProfiles())
	profilesCopy := NewProfiles()
	profiles.CopyTo(profilesCopy)
	assert.EqualValues(t, profiles, profilesCopy)
}
