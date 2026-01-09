// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestReadOnlyProfilesInvalidUsage(t *testing.T) {
	pd := NewProfiles()
	assert.False(t, pd.IsReadOnly())
	res := pd.ResourceProfiles().AppendEmpty().Resource()
	res.Attributes().PutStr("k1", "v1")
	pd.MarkReadOnly()
	assert.True(t, pd.IsReadOnly())
	assert.Panics(t, func() { res.Attributes().PutStr("k2", "v2") })
}

func TestSampleCount(t *testing.T) {
	pd := NewProfiles()
	assert.Equal(t, 0, pd.SampleCount())

	rs := pd.ResourceProfiles().AppendEmpty()
	assert.Equal(t, 0, pd.SampleCount())

	ils := rs.ScopeProfiles().AppendEmpty()
	assert.Equal(t, 0, pd.SampleCount())

	ps := ils.Profiles().AppendEmpty()
	assert.Equal(t, 0, pd.SampleCount())

	ps.Samples().AppendEmpty()
	assert.Equal(t, 1, pd.SampleCount())

	ils2 := rs.ScopeProfiles().AppendEmpty()
	assert.Equal(t, 1, pd.SampleCount())

	ps2 := ils2.Profiles().AppendEmpty()
	assert.Equal(t, 1, pd.SampleCount())

	ps2.Samples().AppendEmpty()
	assert.Equal(t, 2, pd.SampleCount())

	rms := pd.ResourceProfiles()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeProfiles().AppendEmpty()
	ilss := rms.AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples()
	for range 5 {
		ilss.AppendEmpty()
	}
	// 5 + 2 (from rms.At(0) and rms.At(1) initialized first)
	assert.Equal(t, 7, pd.SampleCount())
}

func TestProfileCount(t *testing.T) {
	pd := NewProfiles()
	assert.Equal(t, 0, pd.ProfileCount())

	rs := pd.ResourceProfiles().AppendEmpty()
	assert.Equal(t, 0, pd.ProfileCount())

	ils := rs.ScopeProfiles().AppendEmpty()
	assert.Equal(t, 0, pd.ProfileCount())

	ps := ils.Profiles().AppendEmpty()
	assert.Equal(t, 1, pd.ProfileCount())

	ps.Samples().AppendEmpty()
	assert.Equal(t, 1, pd.ProfileCount())

	ils2 := rs.ScopeProfiles().AppendEmpty()
	assert.Equal(t, 1, pd.ProfileCount())

	ps2 := ils2.Profiles().AppendEmpty()
	assert.Equal(t, 2, pd.ProfileCount())

	ps2.Samples().AppendEmpty()
	assert.Equal(t, 2, pd.ProfileCount())

	rms := pd.ResourceProfiles()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeProfiles().AppendEmpty()
	ilss := rms.AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples()
	for range 5 {
		ilss.AppendEmpty()
	}
	// 5 + 2 (from rms.At(0) and rms.At(1) initialized first)
	assert.Equal(t, 3, pd.ProfileCount())
}

func TestSampleCountWithEmpty(t *testing.T) {
	assert.Equal(t, 0, newProfiles(&internal.ExportProfilesServiceRequest{
		ResourceProfiles: []*internal.ResourceProfiles{{}},
	}, new(internal.State)).SampleCount())
	assert.Equal(t, 0, newProfiles(&internal.ExportProfilesServiceRequest{
		ResourceProfiles: []*internal.ResourceProfiles{
			{
				ScopeProfiles: []*internal.ScopeProfiles{{}},
			},
		},
	}, new(internal.State)).SampleCount())
	assert.Equal(t, 1, newProfiles(&internal.ExportProfilesServiceRequest{
		ResourceProfiles: []*internal.ResourceProfiles{
			{
				ScopeProfiles: []*internal.ScopeProfiles{
					{
						Profiles: []*internal.Profile{
							{
								Samples: []*internal.Sample{
									{},
								},
							},
						},
					},
				},
			},
		},
	}, new(internal.State)).SampleCount())
}

func TestProfilesSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name     string
		profiles Profiles

		src ProfilesDictionary
		dst ProfilesDictionary

		wantProfiles   Profiles
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:     "with an empty profiles",
			profiles: NewProfiles(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantProfiles:   NewProfiles(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with a profiles that has a profile",
			profiles: func() Profiles {
				p := NewProfiles()
				profile := p.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
				profile.Samples().AppendEmpty().SetLinkIndex(1)
				return p
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.LinkTable().AppendEmpty()
				l := d.LinkTable().AppendEmpty()
				l.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.LinkTable().AppendEmpty()
				d.LinkTable().AppendEmpty()
				return d
			}(),

			wantProfiles: func() Profiles {
				p := NewProfiles()
				profile := p.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
				profile.Samples().AppendEmpty().SetLinkIndex(2)
				return p
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.LinkTable().AppendEmpty()
				d.LinkTable().AppendEmpty()
				l := d.LinkTable().AppendEmpty()
				l.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				return d
			}(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.profiles
			dst := tt.dst
			err := p.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantProfiles, p)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkProfilesSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	p := NewProfiles()
	profile := p.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	profile.Samples().AppendEmpty().SetLinkIndex(1)

	src := NewProfilesDictionary()
	src.LinkTable().AppendEmpty()
	src.LinkTable().AppendEmpty().SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		b.StartTimer()

		_ = p.switchDictionary(src, dst)
	}
}

func BenchmarkProfilesUsage(b *testing.B) {
	pd := generateTestProfiles()
	ts := pcommon.NewTimestampFromTime(time.Now())
	dur := uint64(1_000_000_000)
	testValProfileID := ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	testSecondValProfileID := ProfileID([16]byte{2, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})

	b.ReportAllocs()

	for b.Loop() {
		for i := 0; i < pd.ResourceProfiles().Len(); i++ {
			rs := pd.ResourceProfiles().At(i)
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
					s.SetDurationNano(dur)
					assert.Equal(b, dur, s.DurationNano())
				}
				s := iss.Profiles().AppendEmpty()
				s.SetProfileID(testSecondValProfileID)
				s.SetTime(ts)
				s.SetDurationNano(dur)
				s.AttributeIndices().Append(1)
				iss.Profiles().RemoveIf(func(lr Profile) bool {
					return lr.ProfileID() == testSecondValProfileID
				})
			}
		}
	}
}

func BenchmarkProfilesMarshalJSON(b *testing.B) {
	pd := generateTestProfiles()
	encoder := &JSONMarshaler{}

	b.ReportAllocs()

	for b.Loop() {
		jsonBuf, err := encoder.MarshalProfiles(pd)
		require.NoError(b, err)
		require.NotNil(b, jsonBuf)
	}
}
