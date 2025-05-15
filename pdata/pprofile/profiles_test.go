// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal/data"
	otlpcollectorprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1development"
	otlpprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestReadOnlyProfilesInvalidUsage(t *testing.T) {
	profiles := NewProfiles()
	assert.False(t, profiles.IsReadOnly())
	res := profiles.ResourceProfiles().AppendEmpty().Resource()
	res.Attributes().PutStr("k1", "v1")
	profiles.MarkReadOnly()
	assert.True(t, profiles.IsReadOnly())
	assert.Panics(t, func() { res.Attributes().PutStr("k2", "v2") })
}

func TestSampleCount(t *testing.T) {
	profiles := NewProfiles()
	assert.Equal(t, 0, profiles.SampleCount())

	rs := profiles.ResourceProfiles().AppendEmpty()
	assert.Equal(t, 0, profiles.SampleCount())

	ils := rs.ScopeProfiles().AppendEmpty()
	assert.Equal(t, 0, profiles.SampleCount())

	ps := ils.Profiles().AppendEmpty()
	assert.Equal(t, 0, profiles.SampleCount())

	ps.Sample().AppendEmpty()
	assert.Equal(t, 1, profiles.SampleCount())

	ils2 := rs.ScopeProfiles().AppendEmpty()
	assert.Equal(t, 1, profiles.SampleCount())

	ps2 := ils2.Profiles().AppendEmpty()
	assert.Equal(t, 1, profiles.SampleCount())

	ps2.Sample().AppendEmpty()
	assert.Equal(t, 2, profiles.SampleCount())

	rms := profiles.ResourceProfiles()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeProfiles().AppendEmpty()
	ilss := rms.AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Sample()
	for i := 0; i < 5; i++ {
		ilss.AppendEmpty()
	}
	// 5 + 2 (from rms.At(0) and rms.At(1) initialized first)
	assert.Equal(t, 7, profiles.SampleCount())
}

func TestSampleCountWithEmpty(t *testing.T) {
	assert.Equal(t, 0, newProfiles(&otlpcollectorprofile.ExportProfilesServiceRequest{
		ResourceProfiles: []*otlpprofile.ResourceProfiles{{}},
	}).SampleCount())
	assert.Equal(t, 0, newProfiles(&otlpcollectorprofile.ExportProfilesServiceRequest{
		ResourceProfiles: []*otlpprofile.ResourceProfiles{
			{
				ScopeProfiles: []*otlpprofile.ScopeProfiles{{}},
			},
		},
	}).SampleCount())
	assert.Equal(t, 1, newProfiles(&otlpcollectorprofile.ExportProfilesServiceRequest{
		ResourceProfiles: []*otlpprofile.ResourceProfiles{
			{
				ScopeProfiles: []*otlpprofile.ScopeProfiles{
					{
						Profiles: []*otlpprofile.Profile{
							{
								Sample: []*otlpprofile.Sample{
									{},
								},
							},
						},
					},
				},
			},
		},
	}).SampleCount())
}

func BenchmarkProfilesUsage(b *testing.B) {
	profiles := NewProfiles()
	fillTestResourceProfilesSlice(profiles.ResourceProfiles())
	ts := pcommon.NewTimestampFromTime(time.Now())
	testValProfileID := ProfileID(data.ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
	testSecondValProfileID := ProfileID(data.ProfileID([16]byte{2, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))

	b.ReportAllocs()
	b.ResetTimer()

	for bb := 0; bb < b.N; bb++ {
		for i := 0; i < profiles.ResourceProfiles().Len(); i++ {
			rs := profiles.ResourceProfiles().At(i)
			res := rs.Resource()
			res.Attributes().PutStr("foo", "bar")
			v, ok := res.Attributes().Get("foo")
			assert.True(b, ok)
			assert.Equal(b, "bar", v.Str())
			v.SetStr("new-bar")
			assert.Equal(b, "new-bar", v.Str())
			res.Attributes().Remove("foo")
			for j := 0; j < rs.ScopeProfiles().Len(); j++ {
				iss := rs.ScopeProfiles().At(j)
				iss.Scope().SetName("new_test_name")
				assert.Equal(b, "new_test_name", iss.Scope().Name())
				for k := 0; k < iss.Profiles().Len(); k++ {
					s := iss.Profiles().At(k)
					s.SetProfileID(testValProfileID)
					assert.Equal(b, testValProfileID, s.ProfileID())
					s.SetTime(ts)
					assert.Equal(b, ts, s.Time())
					s.SetDuration(ts)
					assert.Equal(b, ts, s.Duration())
				}
				s := iss.Profiles().AppendEmpty()
				s.SetProfileID(testSecondValProfileID)
				s.SetTime(ts)
				s.SetDuration(ts)
				s.AttributeIndices().Append(1)
				iss.Profiles().RemoveIf(func(lr Profile) bool {
					return lr.ProfileID() == testSecondValProfileID
				})
			}
		}
	}
}
