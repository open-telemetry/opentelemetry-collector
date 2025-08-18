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
								"sample": [
									{
										"locationsStartIndex": 42
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
	tr.Profiles().ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Sample().AppendEmpty()
	assert.Equal(t, 1, tr.Profiles().SampleCount())
}

func TestRequestJSON(t *testing.T) {
	tr := NewExportRequest()
	require.NoError(t, tr.UnmarshalJSON(profilesRequestJSON))
	assert.Equal(t, int32(42), tr.Profiles().ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Sample().At(0).LocationsStartIndex())

	got, err := tr.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(profilesRequestJSON)), ""), string(got))
}

func TestProfilesProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Profiles as pdata struct.
	td := NewExportRequestFromProfiles(pprofile.Profiles(internal.GenerateTestProfiles()))

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := td.MarshalProto()
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
	td2 := NewExportRequest()
	err = td2.UnmarshalProto(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.Equal(t, td, td2)
}
