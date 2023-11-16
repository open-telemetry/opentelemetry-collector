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

func TestLogRecordSlice(t *testing.T) {
	es := NewLogRecordSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newLogRecordSlice(&[]*otlplogs.LogRecord{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewLogRecord()
	testVal := generateTestLogRecord()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestLogRecord(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestLogRecordSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newLogRecordSlice(&[]*otlplogs.LogRecord{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewLogRecordSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestLogRecordSlice_CopyTo(t *testing.T) {
	dest := NewLogRecordSlice()
	// Test CopyTo to empty
	NewLogRecordSlice().CopyTo(dest)
	assert.Equal(t, NewLogRecordSlice(), dest)

	// Test CopyTo larger slice
	generateTestLogRecordSlice().CopyTo(dest)
	assert.Equal(t, generateTestLogRecordSlice(), dest)

	// Test CopyTo same size slice
	generateTestLogRecordSlice().CopyTo(dest)
	assert.Equal(t, generateTestLogRecordSlice(), dest)
}

func TestLogRecordSlice_EnsureCapacity(t *testing.T) {
	es := generateTestLogRecordSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestLogRecordSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestLogRecordSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestLogRecordSlice(), es)
}

func TestLogRecordSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestLogRecordSlice()
	dest := NewLogRecordSlice()
	src := generateTestLogRecordSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestLogRecordSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestLogRecordSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestLogRecordSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestLogRecordSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewLogRecordSlice()
	emptySlice.RemoveIf(func(el LogRecord) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestLogRecordSlice()
	pos := 0
	filtered.RemoveIf(func(el LogRecord) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestLogRecordSlice_Sort(t *testing.T) {
	es := generateTestLogRecordSlice()
	es.Sort(func(a, b LogRecord) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) < uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b LogRecord) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) > uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func TestLogRecordSlice_ForEach(t *testing.T) {
	// Test ForEach on empty slice
	emptySlice := NewLogRecordSlice()
	emptySlice.ForEach(func(el LogRecord) {
		t.Fail()
	})

	// Test ForEach
	slice := generateTestLogRecordSlice()
	count := 0
	slice.ForEach(func(el LogRecord) {
		count++
	})
	assert.Equal(t, 7, count)
}

func TestLogRecordSlice_ForEachIndex(t *testing.T) {
	// Test ForEach on empty slice
	emptySlice := NewLogRecordSlice()
	emptySlice.ForEachIndex(func(i int, el LogRecord) {
		t.Fail()
	})

	// Test ForEach
	slice := generateTestLogRecordSlice()
	total := 0
	slice.ForEachIndex(func(i int, el LogRecord) {
		total += i
	})
	assert.Equal(t, 0+1+2+3+4+5+6, total)
}

func generateTestLogRecordSlice() LogRecordSlice {
	es := NewLogRecordSlice()
	fillTestLogRecordSlice(es)
	return es
}

func fillTestLogRecordSlice(es LogRecordSlice) {
	*es.orig = make([]*otlplogs.LogRecord, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlplogs.LogRecord{}
		fillTestLogRecord(newLogRecord((*es.orig)[i], es.state))
	}
}
