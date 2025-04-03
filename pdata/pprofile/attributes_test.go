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
	table := NewAttributeTableSlice()
	att := table.AppendEmpty()
	att.SetKey("hello")
	att.Value().SetStr("world")
	att2 := table.AppendEmpty()
	att2.SetKey("bonjour")
	att2.Value().SetStr("monde")

	attrs := FromAttributeIndices(table, NewProfile())
	assert.Equal(t, attrs, pcommon.NewMap())

	// A Location with a single attribute
	loc := NewLocation()
	loc.AttributeIndices().Append(0)

	attrs = FromAttributeIndices(table, loc)

	m := map[string]any{"hello": "world"}
	assert.Equal(t, attrs.AsRaw(), m)

	// A Mapping with two attributes
	mapp := NewLocation()
	mapp.AttributeIndices().Append(0, 1)

	attrs = FromAttributeIndices(table, mapp)

	m = map[string]any{"hello": "world", "bonjour": "monde"}
	assert.Equal(t, attrs.AsRaw(), m)
}

func TestPutAttribute(t *testing.T) {
	table := NewAttributeTableSlice()
	indices := NewProfile()

	// Put a first attribute.
	require.NoError(t, PutAttribute(table, indices, "hello", pcommon.NewValueStr("world")))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, indices.AttributeIndices().AsRaw())

	// Put an attribute, same key, same value.
	// This should be a no-op.
	require.NoError(t, PutAttribute(table, indices, "hello", pcommon.NewValueStr("world")))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, indices.AttributeIndices().AsRaw())

	// Put an attribute, same key, different value.
	// This updates the index and adds to the table.
	fmt.Printf("test\n")
	require.NoError(t, PutAttribute(table, indices, "hello", pcommon.NewValueStr("world2")))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{1}, indices.AttributeIndices().AsRaw())

	// Put an attribute that already exists in the table.
	// This updates the index and does not add to the table.
	require.NoError(t, PutAttribute(table, indices, "hello", pcommon.NewValueStr("world")))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{0}, indices.AttributeIndices().AsRaw())

	// Put a new attribute.
	// This adds an index and adds to the table.
	require.NoError(t, PutAttribute(table, indices, "good", pcommon.NewValueStr("day")))
	assert.Equal(t, 3, table.Len())
	assert.Equal(t, []int32{0, 2}, indices.AttributeIndices().AsRaw())

	// Add multiple distinct attributes.
	for i := range 100 {
		require.NoError(t, PutAttribute(table, indices, fmt.Sprintf("key_%d", i), pcommon.NewValueStr("day")))
		assert.Equal(t, i+4, table.Len())
		assert.Equal(t, i+3, indices.AttributeIndices().Len())
	}
}

func BenchmarkFromAttributeIndices(b *testing.B) {
	table := NewAttributeTableSlice()

	for i := range 10 {
		att := table.AppendEmpty()
		att.SetKey(fmt.Sprintf("key_%d", i))
		att.Value().SetStr(fmt.Sprintf("value_%d", i))
	}

	obj := NewLocation()
	obj.AttributeIndices().Append(1, 3, 7)

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		_ = FromAttributeIndices(table, obj)
	}
}

func BenchmarkPutAttribute(b *testing.B) {
	for _, bb := range []struct {
		name  string
		key   string
		value pcommon.Value

		runBefore func(*testing.B, AttributeTableSlice, attributable)
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

			runBefore: func(_ *testing.B, table AttributeTableSlice, _ attributable) {
				entry := table.AppendEmpty()
				entry.SetKey("attribute")
				entry.Value().SetStr("test")
			},
		},
		{
			name:  "with a duplicate attribute",
			key:   "attribute",
			value: pcommon.NewValueStr("test"),

			runBefore: func(_ *testing.B, table AttributeTableSlice, obj attributable) {
				require.NoError(b, PutAttribute(table, obj, "attribute", pcommon.NewValueStr("test")))
			},
		},
		{
			name:  "with a hundred attributes to loop through",
			key:   "attribute",
			value: pcommon.NewValueStr("test"),

			runBefore: func(_ *testing.B, table AttributeTableSlice, _ attributable) {
				for i := range 100 {
					entry := table.AppendEmpty()
					entry.SetKey(fmt.Sprintf("attr_%d", i))
					entry.Value().SetStr("test")
				}

				entry := table.AppendEmpty()
				entry.SetKey("attribute")
				entry.Value().SetStr("test")
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewAttributeTableSlice()
			obj := NewLocation()

			if bb.runBefore != nil {
				bb.runBefore(b, table, obj)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for n := 0; n < b.N; n++ {
				_ = PutAttribute(table, obj, bb.key, bb.value)
			}
		})
	}
}
