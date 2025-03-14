// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pprofile

import (
	"iter"
	"sort"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
)

// AttributeUnitSlice logically represents a slice of AttributeUnit.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewAttributeUnitSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type AttributeUnitSlice struct {
	orig  *[]*otlpprofiles.AttributeUnit
	state *internal.State
}

func newAttributeUnitSlice(orig *[]*otlpprofiles.AttributeUnit, state *internal.State) AttributeUnitSlice {
	return AttributeUnitSlice{orig: orig, state: state}
}

// NewAttributeUnitSlice creates a AttributeUnitSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewAttributeUnitSlice() AttributeUnitSlice {
	orig := []*otlpprofiles.AttributeUnit(nil)
	state := internal.StateMutable
	return newAttributeUnitSlice(&orig, &state)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewAttributeUnitSlice()".
func (es AttributeUnitSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//
//	for i := 0; i < es.Len(); i++ {
//	    e := es.At(i)
//	    ... // Do something with the element
//	}
func (es AttributeUnitSlice) At(i int) AttributeUnit {
	return newAttributeUnit((*es.orig)[i], es.state)
}

// All returns an iterator over index-value pairs in the slice.
//
//	for i, v := range es.All() {
//	    ... // Do something with index-value pair
//	}
func (es AttributeUnitSlice) All() iter.Seq2[int, AttributeUnit] {
	return func(yield func(int, AttributeUnit) bool) {
		for i := 0; i < es.Len(); i++ {
			if !yield(i, es.At(i)) {
				return
			}
		}
	}
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new AttributeUnitSlice can be initialized:
//
//	es := NewAttributeUnitSlice()
//	es.EnsureCapacity(4)
//	for i := 0; i < 4; i++ {
//	    e := es.AppendEmpty()
//	    // Here should set all the values for e.
//	}
func (es AttributeUnitSlice) EnsureCapacity(newCap int) {
	es.state.AssertMutable()
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlpprofiles.AttributeUnit, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty AttributeUnit.
// It returns the newly added AttributeUnit.
func (es AttributeUnitSlice) AppendEmpty() AttributeUnit {
	es.state.AssertMutable()
	*es.orig = append(*es.orig, &otlpprofiles.AttributeUnit{})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es AttributeUnitSlice) MoveAndAppendTo(dest AttributeUnitSlice) {
	es.state.AssertMutable()
	dest.state.AssertMutable()
	if *dest.orig == nil {
		// We can simply move the entire vector and avoid any allocations.
		*dest.orig = *es.orig
	} else {
		*dest.orig = append(*dest.orig, *es.orig...)
	}
	*es.orig = nil
}

// RemoveIf calls f sequentially for each element present in the slice.
// If f returns true, the element is removed from the slice.
func (es AttributeUnitSlice) RemoveIf(f func(AttributeUnit) bool) {
	es.state.AssertMutable()
	newLen := 0
	for i := 0; i < len(*es.orig); i++ {
		if f(es.At(i)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*es.orig)[newLen] = (*es.orig)[i]
		newLen++
	}
	*es.orig = (*es.orig)[:newLen]
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es AttributeUnitSlice) CopyTo(dest AttributeUnitSlice) {
	dest.state.AssertMutable()
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			newAttributeUnit((*es.orig)[i], es.state).CopyTo(newAttributeUnit((*dest.orig)[i], dest.state))
		}
		return
	}
	origs := make([]otlpprofiles.AttributeUnit, srcLen)
	wrappers := make([]*otlpprofiles.AttributeUnit, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		newAttributeUnit((*es.orig)[i], es.state).CopyTo(newAttributeUnit(wrappers[i], dest.state))
	}
	*dest.orig = wrappers
}

// Sort sorts the AttributeUnit elements within AttributeUnitSlice given the
// provided less function so that two instances of AttributeUnitSlice
// can be compared.
func (es AttributeUnitSlice) Sort(less func(a, b AttributeUnit) bool) {
	es.state.AssertMutable()
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
}
