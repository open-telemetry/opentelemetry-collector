// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProtoProfilesUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalProfiles([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	ld := NewProfiles()
	ld.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().SetSeverityText("error")

	size := marshaler.ProfilesSize(ld)

	bytes, err := marshaler.MarshalProfiles(ld)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)

}

func TestProtoSizerEmptyProfiles(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 0, sizer.ProfilesSize(NewProfiles()))
}

func BenchmarkProfilesToProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	profiles := generateBenchmarkProfiles(128)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalProfiles(profiles)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkProfilesFromProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseProfiles := generateBenchmarkProfiles(128)
	buf, err := marshaler.MarshalProfiles(baseProfiles)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		profiles, err := unmarshaler.UnmarshalProfiles(buf)
		require.NoError(b, err)
		assert.Equal(b, baseProfiles.ResourceProfiles().Len(), profiles.ResourceProfiles().Len())
	}
}

func generateBenchmarkProfiles(profilesCount int) Profiles {
	endTime := pcommon.NewTimestampFromTime(time.Now())

	md := NewProfiles()
	ilm := md.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty()
	ilm.Profiles().EnsureCapacity(profilesCount)
	for i := 0; i < profilesCount; i++ {
		im := ilm.Profiles().AppendEmpty()
		im.SetTimestamp(endTime)
	}
	return md
}
