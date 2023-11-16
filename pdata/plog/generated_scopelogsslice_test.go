// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package plog

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

func TestScopeLogsSlice(t *testing.T) {
	es := NewScopeLogsSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newScopeLogsSlice(&[]*otlplogs.ScopeLogs{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewScopeLogs()
	testVal := generateTestScopeLogs()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestScopeLogs(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestScopeLogsSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newScopeLogsSlice(&[]*otlplogs.ScopeLogs{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewScopeLogsSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestScopeLogsSlice_CopyTo(t *testing.T) {
	dest := NewScopeLogsSlice()
	// Test CopyTo to empty
	NewScopeLogsSlice().CopyTo(dest)
	assert.Equal(t, NewScopeLogsSlice(), dest)

	// Test CopyTo larger slice
	generateTestScopeLogsSlice().CopyTo(dest)
	assert.Equal(t, generateTestScopeLogsSlice(), dest)

	// Test CopyTo same size slice
	generateTestScopeLogsSlice().CopyTo(dest)
	assert.Equal(t, generateTestScopeLogsSlice(), dest)
}

func TestScopeLogsSlice_EnsureCapacity(t *testing.T) {
	es := generateTestScopeLogsSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestScopeLogsSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestScopeLogsSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestScopeLogsSlice(), es)
}

func TestScopeLogsSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestScopeLogsSlice()
	dest := NewScopeLogsSlice()
	src := generateTestScopeLogsSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestScopeLogsSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestScopeLogsSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestScopeLogsSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestScopeLogsSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewScopeLogsSlice()
	emptySlice.RemoveIf(func(el ScopeLogs) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestScopeLogsSlice()
	pos := 0
	filtered.RemoveIf(func(el ScopeLogs) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestScopeLogsSlice_Sort(t *testing.T) {
	es := generateTestScopeLogsSlice()
	es.Sort(func(a, b ScopeLogs) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) < uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b ScopeLogs) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) > uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func TestScopeLogsSlice_ForEach(t *testing.T) {
	// Test ForEach on empty slice
	emptySlice := NewScopeLogsSlice()
	emptySlice.ForEach(func(el ScopeLogs) {
		t.Fail()
	})

	// Test ForEach
	slice := generateTestScopeLogsSlice()
	count := 0
	slice.ForEach(func(el ScopeLogs) {
		count++
	})
	assert.Equal(t, 7, count)
}

func TestScopeLogsSlice_ForEachIndex(t *testing.T) {
	// Test ForEach on empty slice
	emptySlice := NewScopeLogsSlice()
	emptySlice.ForEachIndex(func(i int, el ScopeLogs) {
		t.Fail()
	})

	// Test ForEach
	slice := generateTestScopeLogsSlice()
	total := 0
	slice.ForEachIndex(func(i int, el ScopeLogs) {
		total += i
	})
	assert.Equal(t, 0+1+2+3+4+5+6, total)
}

func generateTestScopeLogsSlice() ScopeLogsSlice {
	es := NewScopeLogsSlice()
	fillTestScopeLogsSlice(es)
	return es
}

func fillTestScopeLogsSlice(es ScopeLogsSlice) {
	*es.orig = make([]*otlplogs.ScopeLogs, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlplogs.ScopeLogs{}
		fillTestScopeLogs(newScopeLogs((*es.orig)[i], es.state))
	}
}
