// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	attrs := FromAttributeIndices(table, NewProfile(), dic)
	assert.Equal(t, attrs, pcommon.NewMap())

	// A Location with a single attribute
	loc := NewLocation()
	loc.AttributeIndices().Append(0)

	attrs = FromAttributeIndices(table, loc, dic)

	m := map[string]any{"hello": "world"}
	assert.Equal(t, attrs.AsRaw(), m)

	// A Mapping with two attributes
	mapp := NewLocation()
	mapp.AttributeIndices().Append(0, 1)

	attrs = FromAttributeIndices(table, mapp, dic)

	m = map[string]any{"hello": "world", "bonjour": "monde"}
	assert.Equal(t, attrs.AsRaw(), m)
}

func testPutAttribute(t *testing.T, record attributable) {
	dic := NewProfilesDictionary()
	dic.StringTable().Append("")
	table := NewKeyValueAndUnitSlice()

	// Put a first attribute.
	require.NoError(t, PutAttribute(table, record, dic, "hello", pcommon.NewValueStr("world")))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, record.AttributeIndices().AsRaw())

	// Put an attribute, same key, same value.
	// This should be a no-op.
	require.NoError(t, PutAttribute(table, record, dic, "hello", pcommon.NewValueStr("world")))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, record.AttributeIndices().AsRaw())

	// Special case: removing and adding again should not change the table as
	// this can lead to multiple identical attributes in the table.
	record.AttributeIndices().FromRaw([]int32{})
	require.NoError(t, PutAttribute(table, record, dic, "hello", pcommon.NewValueStr("world")))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, record.AttributeIndices().AsRaw())

	// Put an attribute, same key, different value.
	// This updates the index and adds to the table.
	require.NoError(t, PutAttribute(table, record, dic, "hello", pcommon.NewValueStr("world2")))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{1}, record.AttributeIndices().AsRaw())

	// Put an attribute that already exists in the table.
	// This updates the index and does not add to the table.
	require.NoError(t, PutAttribute(table, record, dic, "hello", pcommon.NewValueStr("world")))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{0}, record.AttributeIndices().AsRaw())

	// Put a new attribute.
	// This adds an index and adds to the table.
	require.NoError(t, PutAttribute(table, record, dic, "good", pcommon.NewValueStr("day")))
	assert.Equal(t, 3, table.Len())
	assert.Equal(t, []int32{0, 2}, record.AttributeIndices().AsRaw())

	// Add multiple distinct attributes.
	for i := range 100 {
		require.NoError(t, PutAttribute(table, record, dic, fmt.Sprintf("key_%d", i), pcommon.NewValueStr("day")))
		assert.Equal(t, i+4, table.Len())
		assert.Equal(t, i+3, record.AttributeIndices().Len())
	}

	// Add a negative index to the record.
	record.AttributeIndices().Append(-1)
	tableLen := table.Len()
	indicesLen := record.AttributeIndices().Len()
	// Try putting a new attribute, make sure it fails, and that table/indices didn't change.
	require.Error(t, PutAttribute(table, record, dic, "newKey", pcommon.NewValueStr("value")))
	require.Equal(t, tableLen, table.Len())
	require.Equal(t, indicesLen, record.AttributeIndices().Len())

	// Set the last index to the table length, which is out of range.
	record.AttributeIndices().SetAt(indicesLen-1, int32(tableLen)) //nolint:gosec
	// Try putting a new attribute, make sure it fails, and that table/indices didn't change.
	require.Error(t, PutAttribute(table, record, dic, "newKey", pcommon.NewValueStr("value")))
	require.Equal(t, tableLen, table.Len())
	require.Equal(t, indicesLen, record.AttributeIndices().Len())
}

func TestPutAttribute(t *testing.T) {
	// Test every existing record type.
	for _, tt := range []struct {
		name string
		attr attributable
	}{
		{"Profile", NewProfile()},
		{"Sample", NewSample()},
		{"Mapping", NewMapping()},
		{"Location", NewLocation()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testPutAttribute(t, tt.attr)
		})
	}
}

func BenchmarkFromAttributeIndices(b *testing.B) {
	dic := NewProfilesDictionary()
	table := NewKeyValueAndUnitSlice()

	for i := range 10 {
		dic.StringTable().Append(fmt.Sprintf("key_%d", i))

		att := table.AppendEmpty()
		att.SetKeyStrindex(int32(dic.StringTable().Len())) //nolint:gosec // overflow impossible in test
		att.Value().SetStr(fmt.Sprintf("value_%d", i))
	}

	obj := NewLocation()
	obj.AttributeIndices().Append(1, 3, 7)

	b.ReportAllocs()

	for b.Loop() {
		_ = FromAttributeIndices(table, obj, dic)
	}
}

func BenchmarkPutAttribute(b *testing.B) {
	for _, bb := range []struct {
		name  string
		key   string
		value pcommon.Value

		runBefore func(*testing.B, KeyValueAndUnitSlice, attributable, ProfilesDictionary)
	}{
		{
			name:  "with a new string attribute",
			key:   "attribute",
			value: pcommon.NewValueStr("test"),
		},
		{
			name:  "with an existing attribute",
			key:   "attribute",
			value: pcommon.NewValueStr("test"),

			runBefore: func(_ *testing.B, table KeyValueAndUnitSlice, _ attributable, dic ProfilesDictionary) {
				dic.StringTable().Append("attribute")
				entry := table.AppendEmpty()
				entry.SetKeyStrindex(int32(dic.StringTable().Len())) //nolint:gosec // overflow impossible in test
				entry.Value().SetStr("test")
			},
		},
		{
			name:  "with a duplicate attribute",
			key:   "attribute",
			value: pcommon.NewValueStr("test"),

			runBefore: func(_ *testing.B, table KeyValueAndUnitSlice, obj attributable, dic ProfilesDictionary) {
				require.NoError(b, PutAttribute(table, obj, dic, "attribute", pcommon.NewValueStr("test")))
			},
		},
		{
			name:  "with a hundred attributes to loop through",
			key:   "attribute",
			value: pcommon.NewValueStr("test"),

			runBefore: func(_ *testing.B, table KeyValueAndUnitSlice, _ attributable, dic ProfilesDictionary) {
				for i := range 100 {
					dic.StringTable().Append(fmt.Sprintf("attr_%d", i))

					entry := table.AppendEmpty()
					entry.SetKeyStrindex(int32(dic.StringTable().Len())) //nolint:gosec // overflow impossible in test
					entry.Value().SetStr("test")
				}

				dic.StringTable().Append("attribute")
				entry := table.AppendEmpty()
				entry.SetKeyStrindex(int32(dic.StringTable().Len())) //nolint:gosec // overflow impossible in test
				entry.Value().SetStr("test")
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			dic := NewProfilesDictionary()
			table := NewKeyValueAndUnitSlice()
			obj := NewLocation()

			if bb.runBefore != nil {
				bb.runBefore(b, table, obj, dic)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = PutAttribute(table, obj, dic, bb.key, bb.value)
			}
		})
	}
}
