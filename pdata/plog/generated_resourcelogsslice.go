// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package plog

import (
	"sort"

	"go.opentelemetry.io/collector/pdata/internal"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

// ResourceLogsSlice logically represents a slice of ResourceLogs.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewResourceLogsSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceLogsSlice struct {
	orig  *[]*otlplogs.ResourceLogs
	state *internal.State
}

func newResourceLogsSlice(orig *[]*otlplogs.ResourceLogs, state *internal.State) ResourceLogsSlice {
	return ResourceLogsSlice{orig: orig, state: state}
}

// NewResourceLogsSlice creates a ResourceLogsSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewResourceLogsSlice() ResourceLogsSlice {
	orig := []*otlplogs.ResourceLogs(nil)
	state := internal.StateMutable
	return newResourceLogsSlice(&orig, &state)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewResourceLogsSlice()".
func (es ResourceLogsSlice) Len() int {
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
func (es ResourceLogsSlice) At(i int) ResourceLogs {
	return newResourceLogs((*es.orig)[i], es.state)
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new ResourceLogsSlice can be initialized:
//
//	es := NewResourceLogsSlice()
//	es.EnsureCapacity(4)
//	for i := 0; i < 4; i++ {
//	    e := es.AppendEmpty()
//	    // Here should set all the values for e.
//	}
func (es ResourceLogsSlice) EnsureCapacity(newCap int) {
	es.state.AssertMutable()
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlplogs.ResourceLogs, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty ResourceLogs.
// It returns the newly added ResourceLogs.
func (es ResourceLogsSlice) AppendEmpty() ResourceLogs {
	es.state.AssertMutable()
	*es.orig = append(*es.orig, &otlplogs.ResourceLogs{})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es ResourceLogsSlice) MoveAndAppendTo(dest ResourceLogsSlice) {
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
func (es ResourceLogsSlice) RemoveIf(f func(ResourceLogs) bool) {
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
func (es ResourceLogsSlice) CopyTo(dest ResourceLogsSlice) {
	dest.state.AssertMutable()
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			newResourceLogs((*es.orig)[i], es.state).CopyTo(newResourceLogs((*dest.orig)[i], dest.state))
		}
		return
	}
	origs := make([]otlplogs.ResourceLogs, srcLen)
	wrappers := make([]*otlplogs.ResourceLogs, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		newResourceLogs((*es.orig)[i], es.state).CopyTo(newResourceLogs(wrappers[i], dest.state))
	}
	*dest.orig = wrappers
}

// Sort sorts the ResourceLogs elements within ResourceLogsSlice given the
// provided less function so that two instances of ResourceLogsSlice
// can be compared.
func (es ResourceLogsSlice) Sort(less func(a, b ResourceLogs) bool) {
	es.state.AssertMutable()
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
}

// ForEach iterates over ResourceLogsSlice and executes f against each element.
func (es ResourceLogsSlice) ForEach(f func(ResourceLogs)) {
	for i := 0; i < es.Len(); i++ {
		f(es.At(i))
	}
}

// ForEachIndex iterates over ResourceLogsSlice and executes f against each element.
// The function also passes the iteration index.
func (es ResourceLogsSlice) ForEachIndex(f func(int, ResourceLogs)) {
	for i := 0; i < es.Len(); i++ {
		f(i, es.At(i))
	}
}
