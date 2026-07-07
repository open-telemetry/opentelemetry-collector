// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectorprofiles "go.opentelemetry.io/proto/slim/otlp/collector/profiles/v1development"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var (
	_ json.Unmarshaler = ExportRequest{}
	_ json.Marshaler   = ExportRequest{}
)

var profilesRequestJSON = []byte(`
	{
		"resourceProfiles": [
			{
				"resource": {},
				"scopeProfiles": [
					{
						"scope": {},
						"profiles": [
							{
								"sampleType": {},
								"samples": [
									{
										"stackIndex": 42
									}
								],
								"periodType": {}
							}
						]
					}
				]
			}
		],
		"dictionary": {}
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewExportRequest()
	assert.Equal(t, 0, tr.Profiles().SampleCount())
	tr.Profiles().ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty()
	assert.Equal(t, 1, tr.Profiles().SampleCount())
}

func TestRequestJSON(t *testing.T) {
	tr := NewExportRequest()
	require.NoError(t, tr.UnmarshalJSON(profilesRequestJSON))
	assert.Equal(t, int32(42), tr.Profiles().ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Samples().At(0).StackIndex())

	got, err := tr.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(profilesRequestJSON)), ""), string(got))
}

func TestProfilesProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Profiles as pdata struct.
	pd := NewExportRequestFromProfiles(pprofile.Profiles(internal.GenTestProfilesWrapper()))

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := pd.MarshalProto()
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlpcollectorprofiles.ExportProfilesServiceRequest
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	pd2 := NewExportRequest()
	err = pd2.UnmarshalProto(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	// Migration logic will run, so run it on the original message as well.
	otlp.MigrateProfiles(pd.orig.ResourceProfiles)
	assert.Equal(t, pd, pd2)
}

func TestRejectInvalidUTF8(t *testing.T) {
	t.Run("invalid dictionary", func(t *testing.T) {
		pd := pprofile.NewProfiles()
		pd.Dictionary().StringTable().Append(string([]byte{0xff}))
		profiles := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
		profiles.AppendEmpty().Samples().AppendEmpty()

		assert.False(t, NewExportRequestFromProfiles(pd).ValidateUTF8())
		assert.Equal(t, 1, NewExportRequestFromProfiles(pd).RejectInvalidUTF8())
		assert.Equal(t, 0, pd.SampleCount())
	})

	t.Run("invalid resource", func(t *testing.T) {
		pd := pprofile.NewProfiles()
		rp := pd.ResourceProfiles().AppendEmpty()
		rp.Resource().Attributes().PutStr("bad", string([]byte{0xff}))
		rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty()

		assert.Equal(t, 1, NewExportRequestFromProfiles(pd).RejectInvalidUTF8())
		assert.Equal(t, 0, pd.SampleCount())
	})

	t.Run("invalid scope", func(t *testing.T) {
		pd := pprofile.NewProfiles()
		sp := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty()
		sp.Scope().SetName(string([]byte{0xff}))
		sp.Profiles().AppendEmpty().Samples().AppendEmpty()

		assert.Equal(t, 1, NewExportRequestFromProfiles(pd).RejectInvalidUTF8())
		assert.Equal(t, 0, pd.SampleCount())
	})

	t.Run("invalid profile", func(t *testing.T) {
		pd := pprofile.NewProfiles()
		profiles := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
		profiles.AppendEmpty().Samples().AppendEmpty()
		bad := profiles.AppendEmpty()
		bad.SetOriginalPayloadFormat(string([]byte{0xff}))
		bad.Samples().AppendEmpty()

		assert.Equal(t, 1, NewExportRequestFromProfiles(pd).RejectInvalidUTF8())
		assert.Equal(t, 1, pd.SampleCount())
	})
}
