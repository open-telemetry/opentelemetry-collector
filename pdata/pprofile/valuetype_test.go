// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestValueTypeSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name      string
		valueType ValueType

		src ProfilesDictionary
		dst ProfilesDictionary

		wantValueType  ValueType
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:      "with an empty value type",
			valueType: NewValueType(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantValueType:  NewValueType(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing type",
			valueType: func() ValueType {
				vt := NewValueType()
				vt.SetTypeStrindex(1)
				return vt
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

			wantValueType: func() ValueType {
				vt := NewValueType()
				vt.SetTypeStrindex(2)
				return vt
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a type index that does not match anything",
			valueType: func() ValueType {
				vt := NewValueType()
				vt.SetTypeStrindex(1)
				return vt
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantValueType: func() ValueType {
				vt := NewValueType()
				vt.SetTypeStrindex(1)
				return vt
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid type index 1"),
		},
		{
			name: "with an existing unit",
			valueType: func() ValueType {
				vt := NewValueType()
				vt.SetUnitStrindex(1)
				return vt
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

			wantValueType: func() ValueType {
				vt := NewValueType()
				vt.SetUnitStrindex(2)
				return vt
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a unit index that does not match anything",
			valueType: func() ValueType {
				vt := NewValueType()
				vt.SetUnitStrindex(1)
				return vt
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantValueType: func() ValueType {
				vt := NewValueType()
				vt.SetUnitStrindex(1)
				return vt
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid unit index 1"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			vt := tt.valueType
			dst := tt.dst
			err := vt.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantValueType, vt)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkValueTypeSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	vt := NewValueType()
	vt.SetTypeStrindex(1)
	vt.SetUnitStrindex(2)

	src := NewProfilesDictionary()
	src.StringTable().Append("", "test", "foo")

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		dst.StringTable().Append("", "foo")
		b.StartTimer()

		_ = vt.switchDictionary(src, dst)
	}
}
