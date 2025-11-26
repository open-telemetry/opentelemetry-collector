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

func TestSampleSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name   string
		sample Sample

		src ProfilesDictionary
		dst ProfilesDictionary

		wantSample     Sample
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:   "with an empty sample",
			sample: NewSample(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantSample:     NewSample(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing attribute",
			sample: func() Sample {
				s := NewSample()
				s.AttributeIndices().Append(1)
				return s
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

			wantSample: func() Sample {
				s := NewSample()
				s.AttributeIndices().Append(2)
				return s
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
			sample: func() Sample {
				s := NewSample()
				s.AttributeIndices().Append(1)
				return s
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantSample: func() Sample {
				s := NewSample()
				s.AttributeIndices().Append(1)
				return s
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid attribute index 1"),
		},
		{
			name: "with an existing link",
			sample: func() Sample {
				s := NewSample()
				s.SetLinkIndex(1)
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

			wantSample: func() Sample {
				s := NewSample()
				s.SetLinkIndex(2)
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
		{
			name: "with a link index that does not match anything",
			sample: func() Sample {
				s := NewSample()
				s.SetLinkIndex(1)
				return s
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantSample: func() Sample {
				s := NewSample()
				s.SetLinkIndex(1)
				return s
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid link index 1"),
		},
		{
			name: "with an existing stack",
			sample: func() Sample {
				s := NewSample()
				s.SetStackIndex(1)
				return s
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.LocationTable().AppendEmpty().SetAddress(1)
				d.LocationTable().AppendEmpty().SetAddress(2)

				d.StackTable().AppendEmpty()
				s := d.StackTable().AppendEmpty()
				s.LocationIndices().Append(1)
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StackTable().AppendEmpty()
				d.StackTable().AppendEmpty()
				return d
			}(),

			wantSample: func() Sample {
				s := NewSample()
				s.SetStackIndex(2)
				return s
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.LocationTable().AppendEmpty().SetAddress(2)

				d.StackTable().AppendEmpty()
				d.StackTable().AppendEmpty()
				s := d.StackTable().AppendEmpty()
				s.LocationIndices().Append(0)
				return d
			}(),
		},
		{
			name: "with a stack index that does not match anything",
			sample: func() Sample {
				s := NewSample()
				s.SetStackIndex(1)
				return s
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantSample: func() Sample {
				s := NewSample()
				s.SetStackIndex(1)
				return s
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid stack index 1"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sample := tt.sample
			dst := tt.dst
			err := sample.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantSample, sample)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkSampleSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	s := NewSample()
	s.SetLinkIndex(1)
	s.SetStackIndex(1)

	src := NewProfilesDictionary()
	src.LocationTable().AppendEmpty()
	src.LocationTable().AppendEmpty().SetAddress(2)
	src.LinkTable().AppendEmpty()
	src.LinkTable().AppendEmpty().SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	src.StackTable().AppendEmpty()
	src.StackTable().AppendEmpty().LocationIndices().Append(1)

	dst := NewProfilesDictionary()
	src.LinkTable().AppendEmpty()
	src.LinkTable().AppendEmpty().SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	b.ReportAllocs()

	for b.Loop() {
		_ = s.switchDictionary(src, dst)
	}
}
