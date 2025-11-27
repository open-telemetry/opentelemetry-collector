// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpprofiles "go.opentelemetry.io/proto/slim/otlp/profiles/v1development"
	goproto "google.golang.org/protobuf/proto"
)

func TestProfilesProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Profiles as pdata struct.
	td := generateTestProfiles()

	// Marshal its underlying ProtoBuf to wire.
	marshaler := &ProtoMarshaler{}
	wire1, err := marshaler.MarshalProfiles(td)
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlpprofiles.ProfilesData
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var td2 Profiles
	unmarshaler := &ProtoUnmarshaler{}
	td2, err = unmarshaler.UnmarshalProfiles(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.Equal(t, td, td2)
}

func TestProtoProfilesUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalProfiles([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	td := NewProfiles()
	td.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	td.Dictionary().StringTable().Append("foobar")

	size := marshaler.ProfilesSize(td)

	bytes, err := marshaler.MarshalProfiles(td)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizerEmptyProfiles(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 2, sizer.ProfilesSize(NewProfiles()))
}

func BenchmarkProfilesToProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	profiles := generateBenchmarkProfiles(128)

	for b.Loop() {
		buf, err := marshaler.MarshalProfiles(profiles)
		require.NoError(b, err)
		assert.NotEmpty(b, buf)
	}
}

func BenchmarkProfilesFromProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseProfiles := generateBenchmarkProfiles(128)
	buf, err := marshaler.MarshalProfiles(baseProfiles)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)

	b.ReportAllocs()
	for b.Loop() {
		profiles, err := unmarshaler.UnmarshalProfiles(buf)
		require.NoError(b, err)
		assert.Equal(b, baseProfiles.ResourceProfiles().Len(), profiles.ResourceProfiles().Len())
	}
}

func generateBenchmarkProfiles(samplesCount int) Profiles {
	md := NewProfiles()
	ilm := md.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	ilm.Samples().EnsureCapacity(samplesCount)
	for range samplesCount {
		im := ilm.Samples().AppendEmpty()
		im.SetStackIndex(0)
	}
	return md
}
