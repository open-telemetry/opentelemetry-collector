// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestSlice(t *testing.T) {
	es := NewSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newSlice(&[]otlpcommon.AnyValue{}, &state)
	assert.Equal(t, 0, es.Len())

	es.EnsureCapacity(7)
	emptyVal := newValue(&otlpcommon.AnyValue{}, &state)
	testVal := Value(internal.GenerateTestValue())
	assert.Equal(t, 7, cap(*es.getOrig()))
	for i := 0; i < es.Len(); i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, el)
		internal.FillTestValue(internal.Value(el))
		assert.Equal(t, testVal, el)
	}
}

func TestSliceReadOnly(t *testing.T) {
	state := internal.StateReadOnly
	es := newSlice(&[]otlpcommon.AnyValue{{Value: &otlpcommon.AnyValue_IntValue{IntValue: 3}}}, &state)

	assert.Equal(t, 1, es.Len())
	assert.Equal(t, int64(3), es.At(0).Int())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })

	es2 := NewSlice()
	es.CopyTo(es2)
	assert.Equal(t, es.AsRaw(), es2.AsRaw())
	assert.Panics(t, func() { es2.CopyTo(es) })

	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })

	assert.Panics(t, func() { es.RemoveIf(func(Value) bool { return false }) })

	assert.Equal(t, []any{int64(3)}, es.AsRaw())
	assert.Panics(t, func() { _ = es.FromRaw([]any{3}) })
}

func TestSlice_CopyTo(t *testing.T) {
	dest := NewSlice()
	// Test CopyTo to empty
	NewSlice().CopyTo(dest)
	assert.Equal(t, NewSlice(), dest)

	// Test CopyTo larger slice
	Slice(internal.GenerateTestSlice()).CopyTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)

	// Test CopyTo same size slice
	Slice(internal.GenerateTestSlice()).CopyTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)
}

func TestSlice_EnsureCapacity(t *testing.T) {
	es := Slice(internal.GenerateTestSlice())
	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	expectedEs := make(map[*otlpcommon.AnyValue]bool)
	for i := 0; i < es.Len(); i++ {
		expectedEs[es.At(i).getOrig()] = true
	}
	assert.Len(t, expectedEs, es.Len())
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	foundEs := make(map[*otlpcommon.AnyValue]bool, es.Len())
	for i := 0; i < es.Len(); i++ {
		foundEs[es.At(i).getOrig()] = true
	}
	assert.Equal(t, expectedEs, foundEs)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	oldLen := es.Len()
	assert.Len(t, expectedEs, oldLen)
	es.EnsureCapacity(ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.getOrig()))
}

func TestSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := Slice(internal.GenerateTestSlice())
	dest := NewSlice()
	src := Slice(internal.GenerateTestSlice())
	src.MoveAndAppendTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, Slice(internal.GenerateTestSlice()), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	Slice(internal.GenerateTestSlice()).MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewSlice()
	emptySlice.RemoveIf(func(Value) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := Slice(internal.GenerateTestSlice())
	pos := 0
	filtered.RemoveIf(func(Value) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestSliceAll(t *testing.T) {
	es := Slice(internal.GenerateTestSlice())
	assert.NotEmpty(t, es.Len())

	var c int
	for i, v := range es.All() {
		assert.Equal(t, es.At(i), v, "element should match")
		c++
	}
	assert.Equal(t, es.Len(), c, "All elements should have been visited")
}

func TestInvalidSlice(t *testing.T) {
	es := Slice{}

	assert.Panics(t, func() { es.Len() })
	assert.Panics(t, func() { es.At(0) })
	assert.Panics(t, func() { es.CopyTo(Slice{}) })
	assert.Panics(t, func() { es.EnsureCapacity(1) })
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.MoveAndAppendTo(Slice{}) })
	assert.Panics(t, func() { es.RemoveIf(func(Value) bool { return false }) })
	assert.Panics(t, func() { es.AsRaw() })
	assert.Panics(t, func() { _ = es.FromRaw([]any{3}) })
}

func TestSliceEqual(t *testing.T) {
	es := NewSlice()
	es2 := NewSlice()
	assert.True(t, es.Equal(es2))

	v := es.AppendEmpty()
	v.SetStr("test")
	assert.False(t, es.Equal(es2))

	v = es2.AppendEmpty()
	v.SetStr("test")
	assert.True(t, es.Equal(es2))
}

func BenchmarkSliceEqual(b *testing.B) {
	es := NewSlice()
	v := es.AppendEmpty()
	v.SetStr("test")
	cmp := NewSlice()
	v = cmp.AppendEmpty()
	v.SetStr("test")

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		_ = es.Equal(cmp)
	}
}
