// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestResourceProfilesSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name             string
		resourceProfiles ResourceProfiles

		src ProfilesDictionary
		dst ProfilesDictionary

		wantResourceProfiles ResourceProfiles
		wantDictionary       ProfilesDictionary
		wantErr              error
	}{
		{
			name:             "with an empty resource profile",
			resourceProfiles: NewResourceProfiles(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantResourceProfiles: NewResourceProfiles(),
			wantDictionary:       NewProfilesDictionary(),
		},
		{
			name: "with a resource profiles that has a profile",
			resourceProfiles: func() ResourceProfiles {
				r := NewResourceProfiles()
				profile := r.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
				profile.Samples().AppendEmpty().SetLinkIndex(1)
				return r
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

			wantResourceProfiles: func() ResourceProfiles {
				r := NewResourceProfiles()
				profile := r.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
				profile.Samples().AppendEmpty().SetLinkIndex(2)
				return r
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
			rp := tt.resourceProfiles
			dst := tt.dst
			err := rp.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantResourceProfiles, rp)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkResourceProfilesSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	r := NewResourceProfiles()
	profile := r.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	profile.Samples().AppendEmpty().SetLinkIndex(1)

	src := NewProfilesDictionary()
	src.LinkTable().AppendEmpty()
	src.LinkTable().AppendEmpty().SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		b.StartTimer()

		_ = r.switchDictionary(src, dst)
	}
}
