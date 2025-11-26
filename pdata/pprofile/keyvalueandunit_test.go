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

func TestKeyValueAndUnitEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig KeyValueAndUnit
		dest KeyValueAndUnit
		want bool
	}{
		{
			name: "empty keyvalueandunit",
			orig: NewKeyValueAndUnit(),
			dest: NewKeyValueAndUnit(),
			want: true,
		},
		{
			name: "non-empty identical keyvalueandunit",
			orig: buildKeyValueAndUnit(1, 2, pcommon.NewValueStr("test")),
			dest: buildKeyValueAndUnit(1, 2, pcommon.NewValueStr("test")),
			want: true,
		},
		{
			name: "with different key index",
			orig: buildKeyValueAndUnit(1, 2, pcommon.NewValueStr("test")),
			dest: buildKeyValueAndUnit(2, 2, pcommon.NewValueStr("test")),
			want: false,
		},
		{
			name: "with different unit index",
			orig: buildKeyValueAndUnit(1, 2, pcommon.NewValueStr("test")),
			dest: buildKeyValueAndUnit(1, 3, pcommon.NewValueStr("test")),
			want: false,
		},
		{
			name: "with different value",
			orig: buildKeyValueAndUnit(1, 2, pcommon.NewValueStr("test")),
			dest: buildKeyValueAndUnit(1, 2, pcommon.NewValueStr("hello")),
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				assert.True(t, tt.orig.Equal(tt.dest))
			} else {
				assert.False(t, tt.orig.Equal(tt.dest))
			}
		})
	}
}

func TestKeyValueAndUnitSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name            string
		keyValueAndUnit KeyValueAndUnit

		src ProfilesDictionary
		dst ProfilesDictionary

		wantKeyValueAndUnit KeyValueAndUnit
		wantDictionary      ProfilesDictionary
		wantErr             error
	}{
		{
			name:            "with an empty key value and unit",
			keyValueAndUnit: NewKeyValueAndUnit(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantKeyValueAndUnit: NewKeyValueAndUnit(),
			wantDictionary:      NewProfilesDictionary(),
		},
		{
			name: "with an existing key",
			keyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetKeyStrindex(1)
				return kvu
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

			wantKeyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetKeyStrindex(2)
				return kvu
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a key index that does not match anything",
			keyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetKeyStrindex(1)
				return kvu
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantKeyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetKeyStrindex(1)
				return kvu
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid key index 1"),
		},
		{
			name: "with an existing unit",
			keyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetUnitStrindex(1)
				return kvu
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

			wantKeyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetUnitStrindex(2)
				return kvu
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a unit index that does not match anything",
			keyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetUnitStrindex(1)
				return kvu
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantKeyValueAndUnit: func() KeyValueAndUnit {
				kvu := NewKeyValueAndUnit()
				kvu.SetUnitStrindex(1)
				return kvu
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid unit index 1"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			kvu := tt.keyValueAndUnit
			dst := tt.dst
			err := kvu.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantKeyValueAndUnit, kvu)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkKeyValueAndUnitSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	kvu := NewKeyValueAndUnit()
	kvu.SetKeyStrindex(1)

	src := NewProfilesDictionary()
	src.StringTable().Append("", "test")

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		b.StartTimer()

		_ = kvu.switchDictionary(src, dst)
	}
}

func buildKeyValueAndUnit(keyIdx, unitIdx int32, val pcommon.Value) KeyValueAndUnit {
	kvu := NewKeyValueAndUnit()
	kvu.SetKeyStrindex(keyIdx)
	kvu.SetUnitStrindex(unitIdx)
	val.CopyTo(kvu.Value())
	return kvu
}
