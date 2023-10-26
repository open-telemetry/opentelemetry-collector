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

func TestResourceLogsSlice(t *testing.T) {
	es := NewResourceLogsSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newResourceLogsSlice(&[]*otlplogs.ResourceLogs{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewResourceLogs()
	testVal := generateTestResourceLogs()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestResourceLogs(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestResourceLogsSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newResourceLogsSlice(&[]*otlplogs.ResourceLogs{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewResourceLogsSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestResourceLogsSlice_CopyTo(t *testing.T) {
	dest := NewResourceLogsSlice()
	// Test CopyTo to empty
	NewResourceLogsSlice().CopyTo(dest)
	assert.Equal(t, NewResourceLogsSlice(), dest)

	// Test CopyTo larger slice
	generateTestResourceLogsSlice().CopyTo(dest)
	assert.Equal(t, generateTestResourceLogsSlice(), dest)

	// Test CopyTo same size slice
	generateTestResourceLogsSlice().CopyTo(dest)
	assert.Equal(t, generateTestResourceLogsSlice(), dest)
}

func TestResourceLogsSlice_EnsureCapacity(t *testing.T) {
	es := generateTestResourceLogsSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestResourceLogsSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestResourceLogsSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestResourceLogsSlice(), es)
}

func TestResourceLogsSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestResourceLogsSlice()
	dest := NewResourceLogsSlice()
	src := generateTestResourceLogsSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestResourceLogsSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestResourceLogsSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestResourceLogsSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestResourceLogsSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewResourceLogsSlice()
	emptySlice.RemoveIf(func(el ResourceLogs) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestResourceLogsSlice()
	pos := 0
	filtered.RemoveIf(func(el ResourceLogs) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestResourceLogsSlice_Sort(t *testing.T) {
	es := generateTestResourceLogsSlice()
	es.Sort(func(a, b ResourceLogs) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) < uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b ResourceLogs) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) > uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func generateTestResourceLogsSlice() ResourceLogsSlice {
	es := NewResourceLogsSlice()
	fillTestResourceLogsSlice(es)
	return es
}

func fillTestResourceLogsSlice(es ResourceLogsSlice) {
	*es.orig = make([]*otlplogs.ResourceLogs, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlplogs.ResourceLogs{}
		fillTestResourceLogs(newResourceLogs((*es.orig)[i], es.state))
	}
}
