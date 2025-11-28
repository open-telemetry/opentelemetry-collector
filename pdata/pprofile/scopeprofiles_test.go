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

func TestScopeProfilesSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name          string
		scopeProfiles ScopeProfiles

		src ProfilesDictionary
		dst ProfilesDictionary

		wantScopeProfiles ScopeProfiles
		wantDictionary    ProfilesDictionary
		wantErr           error
	}{
		{
			name:          "with an empty scope profile",
			scopeProfiles: NewScopeProfiles(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantScopeProfiles: NewScopeProfiles(),
			wantDictionary:    NewProfilesDictionary(),
		},
		{
			name: "with a scope profiles that has a profile",
			scopeProfiles: func() ScopeProfiles {
				s := NewScopeProfiles()
				profile := s.Profiles().AppendEmpty()
				profile.Samples().AppendEmpty().SetLinkIndex(1)
				return s
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

			wantScopeProfiles: func() ScopeProfiles {
				s := NewScopeProfiles()
				profile := s.Profiles().AppendEmpty()
				profile.Samples().AppendEmpty().SetLinkIndex(2)
				return s
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
			sp := tt.scopeProfiles
			dst := tt.dst
			err := sp.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantScopeProfiles, sp)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkScopeProfilesSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	s := NewScopeProfiles()
	profile := s.Profiles().AppendEmpty()
	profile.Samples().AppendEmpty().SetLinkIndex(1)

	src := NewProfilesDictionary()
	src.LinkTable().AppendEmpty()
	src.LinkTable().AppendEmpty().SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	dst := NewProfilesDictionary()

	b.ReportAllocs()

	for b.Loop() {
		_ = s.switchDictionary(src, dst)
	}
}
