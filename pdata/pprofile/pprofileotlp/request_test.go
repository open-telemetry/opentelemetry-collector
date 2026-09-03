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

func TestRequestUnmarshalProtoInvalid(t *testing.T) {
	tr := NewExportRequest()
	err := tr.UnmarshalProto([]byte{0xFF, 0xFF, 0xFF})
	require.Error(t, err)
}

func TestRequestUnmarshalJSONInvalid(t *testing.T) {
	tr := NewExportRequest()
	err := tr.UnmarshalJSON([]byte(`{"resourceProfiles":`))
	require.Error(t, err)
}

func TestProfilesProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.
	pd := NewExportRequestFromProfiles(generateProfiles())
	pd.Profiles().MarkReadOnly()

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
	requireProfilesEqualIgnoringAppendedStrings(t, pd.Profiles(), pd2.Profiles())
}

func generateProfiles() pprofile.Profiles {
	profiles := pprofile.NewProfiles()
	profiles.Dictionary().StringTable().Append("") // index 0 is the required empty sentinel
	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("service.name", "checkout")
	sp := rp.ScopeProfiles().AppendEmpty()
	sp.Scope().SetName("test-scope")
	sp.Scope().Attributes().PutStr("scope.attr", "scope-value")
	sp.Profiles().AppendEmpty().Samples().AppendEmpty().SetStackIndex(1)
	return profiles
}

// Marshaling appends attribute strings to the dictionary and unmarshaling inlines
// them again without pruning, so got's string table is want's plus an unused tail.
func requireProfilesEqualIgnoringAppendedStrings(t *testing.T, want, got pprofile.Profiles) {
	t.Helper()
	w, g := pprofile.NewProfiles(), pprofile.NewProfiles()
	want.CopyTo(w)
	got.CopyTo(g)
	wantStrings, gotStrings := w.Dictionary().StringTable(), g.Dictionary().StringTable()
	require.GreaterOrEqual(t, gotStrings.Len(), wantStrings.Len())
	gotStrings.FromRaw(gotStrings.AsRaw()[:wantStrings.Len()])
	require.Equal(t, w, g)
}
