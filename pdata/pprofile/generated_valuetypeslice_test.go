// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pprofile

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
)

func TestValueTypeSlice(t *testing.T) {
	es := NewValueTypeSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newValueTypeSlice(&[]*otlpprofiles.ValueType{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewValueType()
	testVal := generateTestValueType()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestValueType(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestValueTypeSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newValueTypeSlice(&[]*otlpprofiles.ValueType{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewValueTypeSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestValueTypeSlice_CopyTo(t *testing.T) {
	dest := NewValueTypeSlice()
	// Test CopyTo to empty
	NewValueTypeSlice().CopyTo(dest)
	assert.Equal(t, NewValueTypeSlice(), dest)

	// Test CopyTo larger slice
	generateTestValueTypeSlice().CopyTo(dest)
	assert.Equal(t, generateTestValueTypeSlice(), dest)

	// Test CopyTo same size slice
	generateTestValueTypeSlice().CopyTo(dest)
	assert.Equal(t, generateTestValueTypeSlice(), dest)
}

func TestValueTypeSlice_EnsureCapacity(t *testing.T) {
	es := generateTestValueTypeSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestValueTypeSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestValueTypeSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestValueTypeSlice(), es)
}

func TestValueTypeSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestValueTypeSlice()
	dest := NewValueTypeSlice()
	src := generateTestValueTypeSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestValueTypeSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestValueTypeSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestValueTypeSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestValueTypeSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewValueTypeSlice()
	emptySlice.RemoveIf(func(el ValueType) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestValueTypeSlice()
	pos := 0
	filtered.RemoveIf(func(el ValueType) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestValueTypeSliceAll(t *testing.T) {
	ms := generateTestValueTypeSlice()
	assert.NotEmpty(t, ms.Len())

	var c int
	for i, v := range ms.All() {
		assert.Equal(t, ms.At(i), v, "element should match")
		c++
	}
	assert.Equal(t, ms.Len(), c, "All elements should have been visited")
}

func TestValueTypeSlice_Sort(t *testing.T) {
	es := generateTestValueTypeSlice()
	es.Sort(func(a, b ValueType) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Less(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b ValueType) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Greater(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func generateTestValueTypeSlice() ValueTypeSlice {
	es := NewValueTypeSlice()
	fillTestValueTypeSlice(es)
	return es
}

func fillTestValueTypeSlice(es ValueTypeSlice) {
	*es.orig = make([]*otlpprofiles.ValueType, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlpprofiles.ValueType{}
		fillTestValueType(newValueType((*es.orig)[i], es.state))
	}
}
