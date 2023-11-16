// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pmetric

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

func TestSummaryDataPointSlice(t *testing.T) {
	es := NewSummaryDataPointSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newSummaryDataPointSlice(&[]*otlpmetrics.SummaryDataPoint{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewSummaryDataPoint()
	testVal := generateTestSummaryDataPoint()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestSummaryDataPoint(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestSummaryDataPointSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newSummaryDataPointSlice(&[]*otlpmetrics.SummaryDataPoint{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewSummaryDataPointSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestSummaryDataPointSlice_CopyTo(t *testing.T) {
	dest := NewSummaryDataPointSlice()
	// Test CopyTo to empty
	NewSummaryDataPointSlice().CopyTo(dest)
	assert.Equal(t, NewSummaryDataPointSlice(), dest)

	// Test CopyTo larger slice
	generateTestSummaryDataPointSlice().CopyTo(dest)
	assert.Equal(t, generateTestSummaryDataPointSlice(), dest)

	// Test CopyTo same size slice
	generateTestSummaryDataPointSlice().CopyTo(dest)
	assert.Equal(t, generateTestSummaryDataPointSlice(), dest)
}

func TestSummaryDataPointSlice_EnsureCapacity(t *testing.T) {
	es := generateTestSummaryDataPointSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestSummaryDataPointSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestSummaryDataPointSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestSummaryDataPointSlice(), es)
}

func TestSummaryDataPointSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestSummaryDataPointSlice()
	dest := NewSummaryDataPointSlice()
	src := generateTestSummaryDataPointSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestSummaryDataPointSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestSummaryDataPointSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestSummaryDataPointSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestSummaryDataPointSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewSummaryDataPointSlice()
	emptySlice.RemoveIf(func(el SummaryDataPoint) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestSummaryDataPointSlice()
	pos := 0
	filtered.RemoveIf(func(el SummaryDataPoint) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestSummaryDataPointSlice_Sort(t *testing.T) {
	es := generateTestSummaryDataPointSlice()
	es.Sort(func(a, b SummaryDataPoint) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) < uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b SummaryDataPoint) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.True(t, uintptr(unsafe.Pointer(es.At(i-1).orig)) > uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func TestSummaryDataPointSlice_ForEach(t *testing.T) {
	// Test ForEach on empty slice
	emptySlice := NewSummaryDataPointSlice()
	emptySlice.ForEach(func(el SummaryDataPoint) {
		t.Fail()
	})

	// Test ForEach
	slice := generateTestSummaryDataPointSlice()
	count := 0
	slice.ForEach(func(el SummaryDataPoint) {
		count++
	})
	assert.Equal(t, 7, count)
}

func TestSummaryDataPointSlice_ForEachIndex(t *testing.T) {
	// Test ForEach on empty slice
	emptySlice := NewSummaryDataPointSlice()
	emptySlice.ForEachIndex(func(i int, el SummaryDataPoint) {
		t.Fail()
	})

	// Test ForEach
	slice := generateTestSummaryDataPointSlice()
	total := 0
	slice.ForEachIndex(func(i int, el SummaryDataPoint) {
		total += i
	})
	assert.Equal(t, 0+1+2+3+4+5+6, total)
}

func generateTestSummaryDataPointSlice() SummaryDataPointSlice {
	es := NewSummaryDataPointSlice()
	fillTestSummaryDataPointSlice(es)
	return es
}

func fillTestSummaryDataPointSlice(es SummaryDataPointSlice) {
	*es.orig = make([]*otlpmetrics.SummaryDataPoint, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlpmetrics.SummaryDataPoint{}
		fillTestSummaryDataPoint(newSummaryDataPoint((*es.orig)[i], es.state))
	}
}
