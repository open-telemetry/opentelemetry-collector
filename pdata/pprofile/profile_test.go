// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProfileSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name    string
		profile Profile

		src ProfilesDictionary
		dst ProfilesDictionary

		wantProfile    Profile
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:    "with an empty profile",
			profile: NewProfile(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantProfile:    NewProfile(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing attribute",
			profile: func() Profile {
				p := NewProfile()
				p.AttributeIndices().Append(1)
				return p
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")

				d.AttributeTable().AppendEmpty()
				a := d.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(1)

				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")

				d.AttributeTable().AppendEmpty()
				d.AttributeTable().AppendEmpty()
				return d
			}(),

			wantProfile: func() Profile {
				p := NewProfile()
				p.AttributeIndices().Append(2)
				return p
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")

				d.AttributeTable().AppendEmpty()
				d.AttributeTable().AppendEmpty()
				a := d.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(2)
				return d
			}(),
		},
		{
			name: "with an attribute index that does not match anything",
			profile: func() Profile {
				p := NewProfile()
				p.AttributeIndices().Append(1)
				return p
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantProfile: func() Profile {
				p := NewProfile()
				p.AttributeIndices().Append(1)
				return p
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid attribute index 1"),
		},
		{
			name: "with a profile that has a sample",
			profile: func() Profile {
				p := NewProfile()
				p.Samples().AppendEmpty().SetLinkIndex(1)
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

			wantProfile: func() Profile {
				p := NewProfile()
				p.Samples().AppendEmpty().SetLinkIndex(2)
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
		{
			name: "with a profile that has a period type",
			profile: func() Profile {
				p := NewProfile()
				p.PeriodType().SetTypeStrindex(1)
				return p
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")
				return d
			}(),

			wantProfile: func() Profile {
				p := NewProfile()
				p.PeriodType().SetTypeStrindex(2)
				return p
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a profile that has a sample type",
			profile: func() Profile {
				p := NewProfile()
				p.SampleType().SetTypeStrindex(1)
				return p
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")
				return d
			}(),

			wantProfile: func() Profile {
				p := NewProfile()
				p.SampleType().SetTypeStrindex(2)
				return p
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			profile := tt.profile
			dst := tt.dst
			err := profile.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantProfile, profile)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}
