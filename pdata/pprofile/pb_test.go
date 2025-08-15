// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
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
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		profiles, err := unmarshaler.UnmarshalProfiles(buf)
		require.NoError(b, err)
		assert.Equal(b, baseProfiles.ResourceProfiles().Len(), profiles.ResourceProfiles().Len())
	}
}

func generateBenchmarkProfiles(samplesCount int) Profiles {
	md := NewProfiles()
	ilm := md.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	ilm.Sample().EnsureCapacity(samplesCount)
	for i := 0; i < samplesCount; i++ {
		im := ilm.Sample().AppendEmpty()
		im.SetLocationsStartIndex(2)
		im.SetLocationsLength(10)
	}
	return md
}
