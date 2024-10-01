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
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1experimental"
)

func TestProfilesContainersSlice(t *testing.T) {
	es := NewProfilesContainersSlice()
	assert.Equal(t, 0, es.Len())
	state := internal.StateMutable
	es = newProfilesContainersSlice(&[]*otlpprofiles.ProfileContainer{}, &state)
	assert.Equal(t, 0, es.Len())

	emptyVal := NewProfileContainer()
	testVal := generateTestProfileContainer()
	for i := 0; i < 7; i++ {
		el := es.AppendEmpty()
		assert.Equal(t, emptyVal, es.At(i))
		fillTestProfileContainer(el)
		assert.Equal(t, testVal, es.At(i))
	}
	assert.Equal(t, 7, es.Len())
}

func TestProfilesContainersSliceReadOnly(t *testing.T) {
	sharedState := internal.StateReadOnly
	es := newProfilesContainersSlice(&[]*otlpprofiles.ProfileContainer{}, &sharedState)
	assert.Equal(t, 0, es.Len())
	assert.Panics(t, func() { es.AppendEmpty() })
	assert.Panics(t, func() { es.EnsureCapacity(2) })
	es2 := NewProfilesContainersSlice()
	es.CopyTo(es2)
	assert.Panics(t, func() { es2.CopyTo(es) })
	assert.Panics(t, func() { es.MoveAndAppendTo(es2) })
	assert.Panics(t, func() { es2.MoveAndAppendTo(es) })
}

func TestProfilesContainersSlice_CopyTo(t *testing.T) {
	dest := NewProfilesContainersSlice()
	// Test CopyTo to empty
	NewProfilesContainersSlice().CopyTo(dest)
	assert.Equal(t, NewProfilesContainersSlice(), dest)

	// Test CopyTo larger slice
	generateTestProfilesContainersSlice().CopyTo(dest)
	assert.Equal(t, generateTestProfilesContainersSlice(), dest)

	// Test CopyTo same size slice
	generateTestProfilesContainersSlice().CopyTo(dest)
	assert.Equal(t, generateTestProfilesContainersSlice(), dest)
}

func TestProfilesContainersSlice_EnsureCapacity(t *testing.T) {
	es := generateTestProfilesContainersSlice()

	// Test ensure smaller capacity.
	const ensureSmallLen = 4
	es.EnsureCapacity(ensureSmallLen)
	assert.Less(t, ensureSmallLen, es.Len())
	assert.Equal(t, es.Len(), cap(*es.orig))
	assert.Equal(t, generateTestProfilesContainersSlice(), es)

	// Test ensure larger capacity
	const ensureLargeLen = 9
	es.EnsureCapacity(ensureLargeLen)
	assert.Less(t, generateTestProfilesContainersSlice().Len(), ensureLargeLen)
	assert.Equal(t, ensureLargeLen, cap(*es.orig))
	assert.Equal(t, generateTestProfilesContainersSlice(), es)
}

func TestProfilesContainersSlice_MoveAndAppendTo(t *testing.T) {
	// Test MoveAndAppendTo to empty
	expectedSlice := generateTestProfilesContainersSlice()
	dest := NewProfilesContainersSlice()
	src := generateTestProfilesContainersSlice()
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestProfilesContainersSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo empty slice
	src.MoveAndAppendTo(dest)
	assert.Equal(t, generateTestProfilesContainersSlice(), dest)
	assert.Equal(t, 0, src.Len())
	assert.Equal(t, expectedSlice.Len(), dest.Len())

	// Test MoveAndAppendTo not empty slice
	generateTestProfilesContainersSlice().MoveAndAppendTo(dest)
	assert.Equal(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.Equal(t, expectedSlice.At(i), dest.At(i))
		assert.Equal(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func TestProfilesContainersSlice_RemoveIf(t *testing.T) {
	// Test RemoveIf on empty slice
	emptySlice := NewProfilesContainersSlice()
	emptySlice.RemoveIf(func(el ProfileContainer) bool {
		t.Fail()
		return false
	})

	// Test RemoveIf
	filtered := generateTestProfilesContainersSlice()
	pos := 0
	filtered.RemoveIf(func(el ProfileContainer) bool {
		pos++
		return pos%3 == 0
	})
	assert.Equal(t, 5, filtered.Len())
}

func TestProfilesContainersSlice_Sort(t *testing.T) {
	es := generateTestProfilesContainersSlice()
	es.Sort(func(a, b ProfileContainer) bool {
		return uintptr(unsafe.Pointer(a.orig)) < uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Less(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
	es.Sort(func(a, b ProfileContainer) bool {
		return uintptr(unsafe.Pointer(a.orig)) > uintptr(unsafe.Pointer(b.orig))
	})
	for i := 1; i < es.Len(); i++ {
		assert.Greater(t, uintptr(unsafe.Pointer(es.At(i-1).orig)), uintptr(unsafe.Pointer(es.At(i).orig)))
	}
}

func generateTestProfilesContainersSlice() ProfilesContainersSlice {
	es := NewProfilesContainersSlice()
	fillTestProfilesContainersSlice(es)
	return es
}

func fillTestProfilesContainersSlice(es ProfilesContainersSlice) {
	*es.orig = make([]*otlpprofiles.ProfileContainer, 7)
	for i := 0; i < 7; i++ {
		(*es.orig)[i] = &otlpprofiles.ProfileContainer{}
		fillTestProfileContainer(newProfileContainer((*es.orig)[i], es.state))
	}
}
