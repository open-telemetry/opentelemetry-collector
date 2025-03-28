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

func TestAddAttribute(t *testing.T) {
	table := NewAttributeTableSlice()
	att := table.AppendEmpty()
	att.SetKey("hello")
	att.Value().SetStr("world")

	// Add a brand new attribute
	loc := NewLocation()
	err := AddAttribute(table, loc, "bonjour", pcommon.NewValueStr("monde"))
	require.NoError(t, err)

	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{1}, loc.AttributeIndices().AsRaw())

	// Add an already existing attribute
	mapp := NewMapping()
	err = AddAttribute(table, mapp, "hello", pcommon.NewValueStr("world"))
	require.NoError(t, err)

	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{0}, mapp.AttributeIndices().AsRaw())

	// Add a duplicate attribute
	err = AddAttribute(table, mapp, "hello", pcommon.NewValueStr("world"))
	require.NoError(t, err)

	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{0}, mapp.AttributeIndices().AsRaw())
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

func BenchmarkAddAttribute(b *testing.B) {
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
				require.NoError(b, AddAttribute(table, obj, "attribute", pcommon.NewValueStr("test")))
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
				_ = AddAttribute(table, obj, bb.key, bb.value)
			}
		})
	}
}
