// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package ptrace

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

func TestSpanSlice(t *testing.T) {
	es := NewSpanSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newSpanSlice(&[]*otlptrace.Span{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewSpan()
	testVal := generateTestSpan()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestSpan(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestSpanSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newSpanSlice(&[]*otlptrace.Span{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewSpanSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestSpanSlice_CopyTo(t *testing.T) {
	dest := NewSpanSlice()
	// Test CopyTo to empty
	NewSpanSlice().CopyTo(dest)
	assert.Equal(t, NewSpanSlice(), dest)

	// Test CopyTo larger slice
	generateTestSpanSlice().CopyTo(dest)
	assert.Equal(t, generateTestSpanSlice(), dest)

	// Test CopyTo same size slice
	generateTestSpanSlice().CopyTo(dest)
	assert.Equal(t, generateTestSpanSlice(), dest)
}

func TestSpanSlice_EnsureCapacity(t *testing.T) {
	es := generateTestSpanSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestSpanSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestSpanSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestSpanSlice(), es)
}

func TestSpanSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestSpanSlice()
	dest := NewSpanSlice()
	src := generateTestSpanSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestSpanSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestSpanSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestSpanSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestSpanSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewSpanSlice()
	emptySlice.RemoveIf(func(el Span) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestSpanSlice()
	pos := 0
	filtered.RemoveIf(func(el Span) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestSpanSliceAll(t *testing.T) {
	ms := generateTestSpanSlice()
	assert.NotEmpty(t, ms.Len())

	var c int
	for i, v := range ms.All() {
		assert.Equalf(t, ms.At(i), v, "element should match")
		c++
	}
	assert.Equalf(t, ms.Len(), c, "All elements should have been visited")
}

func TestSpanSlice_Sort(t *testing.T) {
	es := generateTestSpanSlice()
	es.Sort(func(a, b Span) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Less(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b Span) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Greater(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func generateTestSpanSlice() SpanSlice {
	es := NewSpanSlice()
	fillTestSpanSlice(es)
	return es
}

func fillTestSpanSlice(es SpanSlice) {
	*es.orig = make([]*otlptrace.Span, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlptrace.Span{}
		fillTestSpan(newSpan((*es.orig)[i], es.state))
	}
}
