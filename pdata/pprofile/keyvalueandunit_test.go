// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

func buildKeyValueAndUnit(keyIdx, unitIdx int32, val pcommon.Value) KeyValueAndUnit {
	kvu := NewKeyValueAndUnit()
	kvu.SetKeyStrindex(keyIdx)
	kvu.SetUnitStrindex(unitIdx)
	val.CopyTo(kvu.Value())
	return kvu
}
