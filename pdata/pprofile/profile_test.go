// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
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
				d.StringTable().Append("", "test")

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
				d.StringTable().Append("", "test")

				d.AttributeTable().AppendEmpty()
				d.AttributeTable().AppendEmpty()
				a := d.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(1)
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
			name: "with an attribute index equal to the source table length (boundary condition)",
			profile: func() Profile {
				p := NewProfile()
				p.AttributeIndices().Append(2)
				return p
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.AttributeTable().AppendEmpty()
				d.AttributeTable().AppendEmpty()
				return d
			}(),
			dst: NewProfilesDictionary(),

			wantProfile: func() Profile {
				p := NewProfile()
				p.AttributeIndices().Append(2)
				return p
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid attribute index 2"),
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
		{
			name: "Profile with various elements",
			profile: func() Profile {
				p := NewProfile()

				p.SampleType().SetTypeStrindex(1)
				p.SampleType().SetUnitStrindex(2)

				p.PeriodType().SetTypeStrindex(3)
				p.PeriodType().SetUnitStrindex(4)

				p.AttributeIndices().Append(1)

				return p
			}(),
			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				// Make sure we are conform with the protocol
				d.MappingTable().AppendEmpty()
				d.LocationTable().AppendEmpty()
				d.FunctionTable().AppendEmpty()
				d.LinkTable().AppendEmpty()
				d.StringTable().Append("")
				d.AttributeTable().AppendEmpty()
				d.StackTable().AppendEmpty()

				d.StringTable().Append("sample-type")    // 1
				d.StringTable().Append("sample-unit")    // 2
				d.StringTable().Append("period-type")    // 3
				d.StringTable().Append("period-unit")    // 4
				d.StringTable().Append("unrelated-1")    // 5
				d.StringTable().Append("unrelated-2")    // 6
				d.StringTable().Append("attribute-key")  // 7
				d.StringTable().Append("attribute-unit") // 8

				a := d.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(7)
				a.Value().SetStr("AnyValue")
				a.SetUnitStrindex(8)

				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				// Make sure we are conform with the protocol
				d.MappingTable().AppendEmpty()
				d.LocationTable().AppendEmpty()
				d.FunctionTable().AppendEmpty()
				d.LinkTable().AppendEmpty()
				d.StringTable().Append("")
				d.AttributeTable().AppendEmpty()
				d.StackTable().AppendEmpty()

				return d
			}(),
			wantProfile: func() Profile {
				p := NewProfile()

				p.AttributeIndices().Append(1)

				// Order of entries depend on the order of
				// processing in switchDictionary()
				p.SampleType().SetTypeStrindex(3)
				p.SampleType().SetUnitStrindex(4)

				p.PeriodType().SetTypeStrindex(1)
				p.PeriodType().SetUnitStrindex(2)

				return p
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				// Make sure we are conform with the protocol
				d.MappingTable().AppendEmpty()
				d.LocationTable().AppendEmpty()
				d.FunctionTable().AppendEmpty()
				d.LinkTable().AppendEmpty()
				d.StringTable().Append("")
				d.AttributeTable().AppendEmpty()
				d.StackTable().AppendEmpty()

				a := d.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(7)
				a.SetUnitStrindex(8)
				a.Value().SetStr("AnyValue")

				// Order of entries depend on the order of
				// processing in switchDictionary()
				d.StringTable().Append("period-type") // 1
				d.StringTable().Append("period-unit") // 2
				d.StringTable().Append("sample-type") // 3
				d.StringTable().Append("sample-unit") // 4
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

func BenchmarkProfileSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	p := NewProfile()
	p.AttributeIndices().Append(1, 2)

	src := NewProfilesDictionary()
	src.StringTable().Append("", "test", "foo")
	src.AttributeTable().AppendEmpty()
	src.AttributeTable().AppendEmpty().SetKeyStrindex(1)
	src.AttributeTable().AppendEmpty().SetKeyStrindex(2)

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		dst.StringTable().Append("", "foo")
		dst.AttributeTable().AppendEmpty()
		dst.AttributeTable().AppendEmpty().SetKeyStrindex(1)
		b.StartTimer()

		_ = p.switchDictionary(src, dst)
	}
}

func TestProfile_Duration(_ *testing.T) {
	ms := NewProfile()
	ms.SetDuration(0)

	ts := ms.Duration()
	_ = ts
}
