// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestFromAttributeIndices(t *testing.T) {
	dic := NewProfilesDictionary()
	dic.StringTable().Append("")
	dic.StringTable().Append("hello")
	dic.StringTable().Append("bonjour")

	table := NewKeyValueAndUnitSlice()
	att := table.AppendEmpty()
	att.SetKeyStrindex(1)
	att.Value().SetStr("world")
	att2 := table.AppendEmpty()
	att2.SetKeyStrindex(2)
	att2.Value().SetStr("monde")

	attrs, err := FromAttributeIndices(table, NewProfile(), dic)
	require.NoError(t, err)
	assert.Equal(t, attrs, pcommon.NewMap())

	// A Location with a single attribute
	loc := NewLocation()
	loc.AttributeIndices().Append(0)

	attrs, err = FromAttributeIndices(table, loc, dic)
	require.NoError(t, err)

	m := map[string]any{"hello": "world"}
	assert.Equal(t, attrs.AsRaw(), m)

	// A Mapping with two attributes
	mapp := NewLocation()
	mapp.AttributeIndices().Append(0, 1)

	attrs, err = FromAttributeIndices(table, mapp, dic)
	require.NoError(t, err)

	m = map[string]any{"hello": "world", "bonjour": "monde"}
	assert.Equal(t, attrs.AsRaw(), m)

	// Out-of-bounds attribute index
	oob := NewLocation()
	oob.AttributeIndices().Append(int32(table.Len()))
	_, err = FromAttributeIndices(table, oob, dic)
	assert.ErrorContains(t, err, "out of bounds")

	// Negative attribute index
	neg := NewLocation()
	neg.AttributeIndices().Append(-1)
	_, err = FromAttributeIndices(table, neg, dic)
	assert.ErrorContains(t, err, "out of bounds")

	// Out-of-bounds key string index
	badKey := NewKeyValueAndUnitSlice()
	bk := badKey.AppendEmpty()
	bk.SetKeyStrindex(int32(dic.StringTable().Len()))
	bk.Value().SetStr("x")
	badKeyRec := NewLocation()
	badKeyRec.AttributeIndices().Append(0)
	_, err = FromAttributeIndices(badKey, badKeyRec, dic)
	assert.ErrorContains(t, err, "out of bounds")
}

func BenchmarkFromAttributeIndices(b *testing.B) {
	dic := NewProfilesDictionary()
	table := NewKeyValueAndUnitSlice()

	for i := range 10 {
		dic.StringTable().Append(fmt.Sprintf("key_%d", i))

		att := table.AppendEmpty()
		att.SetKeyStrindex(int32(dic.StringTable().Len()))
		att.Value().SetStr(fmt.Sprintf("value_%d", i))
	}

	obj := NewLocation()
	obj.AttributeIndices().Append(1, 3, 7)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = FromAttributeIndices(table, obj, dic)
	}
}

func TestSetAttribute(t *testing.T) {
	table := NewKeyValueAndUnitSlice()
	attr := NewKeyValueAndUnit()
	attr.SetKeyStrindex(1)
	attr.SetUnitStrindex(2)
	require.NoError(t, attr.Value().FromRaw("test"))
	attr2 := NewKeyValueAndUnit()
	attr2.SetKeyStrindex(3)
	attr2.SetUnitStrindex(4)
	require.NoError(t, attr.Value().FromRaw("test2"))

	// Put a first value
	idx, err := SetAttribute(table, attr)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Put the same attribute
	// This should be a no-op.
	idx, err = SetAttribute(table, attr)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Set a new value
	// This sets the index and adds to the table.
	idx, err = SetAttribute(table, attr2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)

	// Set an existing value
	idx, err = SetAttribute(table, attr)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), idx)
	// Set another existing value
	idx, err = SetAttribute(table, attr2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)
}

func BenchmarkSetAttribute(b *testing.B) {
	testutil.SkipMemoryBench(b)
	for _, bb := range []struct {
		name string
		attr KeyValueAndUnit

		runBefore func(*testing.B, KeyValueAndUnitSlice)
	}{
		{
			name: "with a new attribute",
			attr: NewKeyValueAndUnit(),
		},
		{
			name: "with an existing attribute",
			attr: func() KeyValueAndUnit {
				a := NewKeyValueAndUnit()
				a.SetKeyStrindex(1)
				return a
			}(),

			runBefore: func(_ *testing.B, table KeyValueAndUnitSlice) {
				a := table.AppendEmpty()
				a.SetKeyStrindex(1)
			},
		},
		{
			name: "with a duplicate attribute",
			attr: NewKeyValueAndUnit(),

			runBefore: func(_ *testing.B, table KeyValueAndUnitSlice) {
				_, err := SetAttribute(table, NewKeyValueAndUnit())
				require.NoError(b, err)
			},
		},
		{
			name: "with a hundred locations to loop through",
			attr: func() KeyValueAndUnit {
				a := NewKeyValueAndUnit()
				a.SetKeyStrindex(1)
				return a
			}(),

			runBefore: func(_ *testing.B, table KeyValueAndUnitSlice) {
				for i := range 100 {
					l := table.AppendEmpty()
					l.SetKeyStrindex(int32(i))
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewKeyValueAndUnitSlice()

			if bb.runBefore != nil {
				bb.runBefore(b, table)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = SetAttribute(table, bb.attr)
			}
		})
	}
}
