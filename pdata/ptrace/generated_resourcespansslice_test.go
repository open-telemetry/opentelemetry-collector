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

func TestResourceSpansSlice(t *testing.T) {
	es := NewResourceSpansSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newResourceSpansSlice(&[]*otlptrace.ResourceSpans{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewResourceSpans()
	testVal := generateTestResourceSpans()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestResourceSpans(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestResourceSpansSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newResourceSpansSlice(&[]*otlptrace.ResourceSpans{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewResourceSpansSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestResourceSpansSlice_CopyTo(t *testing.T) {
	dest := NewResourceSpansSlice()
	// Test CopyTo to empty
	NewResourceSpansSlice().CopyTo(dest)
	assert.Equal(t, NewResourceSpansSlice(), dest)

	// Test CopyTo larger slice
	generateTestResourceSpansSlice().CopyTo(dest)
	assert.Equal(t, generateTestResourceSpansSlice(), dest)

	// Test CopyTo same size slice
	generateTestResourceSpansSlice().CopyTo(dest)
	assert.Equal(t, generateTestResourceSpansSlice(), dest)
}

func TestResourceSpansSlice_EnsureCapacity(t *testing.T) {
	es := generateTestResourceSpansSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestResourceSpansSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestResourceSpansSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestResourceSpansSlice(), es)
}

func TestResourceSpansSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestResourceSpansSlice()
	dest := NewResourceSpansSlice()
	src := generateTestResourceSpansSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestResourceSpansSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestResourceSpansSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestResourceSpansSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestResourceSpansSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewResourceSpansSlice()
	emptySlice.RemoveIf(func(el ResourceSpans) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestResourceSpansSlice()
	pos := 0
	filtered.RemoveIf(func(el ResourceSpans) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestResourceSpansSliceAll(t *testing.T) {
	ms := generateTestResourceSpansSlice()
	assert.NotEmpty(t, ms.Len())

	var c int
	for i, v := range ms.All() {
		assert.Equal(t, ms.At(i), v, "element should match")
		c++
	}
	assert.Equal(t, ms.Len(), c, "All elements should have been visited")
}

func TestResourceSpansSlice_Sort(t *testing.T) {
	es := generateTestResourceSpansSlice()
	es.Sort(func(a, b ResourceSpans) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Less(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b ResourceSpans) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Greater(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func generateTestResourceSpansSlice() ResourceSpansSlice {
	es := NewResourceSpansSlice()
	fillTestResourceSpansSlice(es)
	return es
}

func fillTestResourceSpansSlice(es ResourceSpansSlice) {
	*es.orig = make([]*otlptrace.ResourceSpans, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlptrace.ResourceSpans{}
		fillTestResourceSpans(newResourceSpans((*es.orig)[i], es.state))
	}
}
