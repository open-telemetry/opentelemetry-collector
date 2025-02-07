// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pcommon

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

// Int64Slice represents a []int64 slice.
// The instance of Int64Slice can be assigned to multiple objects since it's immutable.
//
// Must use NewInt64Slice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type Int64Slice internal.Int64Slice

func (ms Int64Slice) getOrig() *[]int64 {
	return internal.GetOrigInt64Slice(internal.Int64Slice(ms))
}

func (ms Int64Slice) getState() *internal.State {
	return internal.GetInt64SliceState(internal.Int64Slice(ms))
}

// NewInt64Slice creates a new empty Int64Slice.
func NewInt64Slice() Int64Slice {
	orig := []int64(nil)
	state := internal.StateMutable
	return Int64Slice(internal.NewInt64Slice(&orig, &state))
}

// AsRaw returns a copy of the []int64 slice.
func (ms Int64Slice) AsRaw() []int64 {
	return copyInt64Slice(nil, *ms.getOrig())
}

// FromRaw copies raw []int64 into the slice Int64Slice.
func (ms Int64Slice) FromRaw(val []int64) {
	ms.getState().AssertMutable()
	*ms.getOrig() = copyInt64Slice(*ms.getOrig(), val)
}

// Len returns length of the []int64 slice value.
// Equivalent of len(int64Slice).
func (ms Int64Slice) Len() int {
	return len(*ms.getOrig())
}

// At returns an item from particular index.
// Equivalent of int64Slice[i].
func (ms Int64Slice) At(i int) int64 {
	return (*ms.getOrig())[i]
}

// SetAt sets int64 item at particular index.
// Equivalent of int64Slice[i] = val
func (ms Int64Slice) SetAt(i int, val int64) {
	ms.getState().AssertMutable()
	(*ms.getOrig())[i] = val
}

// EnsureCapacity ensures Int64Slice has at least the specified capacity.
//  1. If the newCap <= cap, then is no change in capacity.
//  2. If the newCap > cap, then the slice capacity will be expanded to the provided value which will be equivalent of:
//     buf := make([]int64, len(int64Slice), newCap)
//     copy(buf, int64Slice)
//     int64Slice = buf
func (ms Int64Slice) EnsureCapacity(newCap int) {
	ms.getState().AssertMutable()
	oldCap := cap(*ms.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]int64, len(*ms.getOrig()), newCap)
	copy(newOrig, *ms.getOrig())
	*ms.getOrig() = newOrig
}

// Append appends extra elements to Int64Slice.
// Equivalent of int64Slice = append(int64Slice, elms...)
func (ms Int64Slice) Append(elms ...int64) {
	ms.getState().AssertMutable()
	*ms.getOrig() = append(*ms.getOrig(), elms...)
}

// MoveTo moves all elements from the current slice overriding the destination and
// resetting the current instance to its zero value.
func (ms Int64Slice) MoveTo(dest Int64Slice) {
	ms.getState().AssertMutable()
	dest.getState().AssertMutable()
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = nil
}

// CopyTo copies all elements from the current slice overriding the destination.
func (ms Int64Slice) CopyTo(dest Int64Slice) {
	dest.getState().AssertMutable()
	*dest.getOrig() = copyInt64Slice(*dest.getOrig(), *ms.getOrig())
}

func copyInt64Slice(dst, src []int64) []int64 {
	dst = dst[:0]
	return append(dst, src...)
}

// IncrementFrom increments all elements by the elements from another slice.
func (ms Int64Slice) IncrementFrom(other Int64Slice, offset int) bool {
	if offset < 0 {
		return false
	}
	ms.getState().AssertMutable()
	newLen := max(ms.Len(), other.Len()+offset)
	ours := *ms.getOrig()
	if cap(ours) < newLen {
		return false
	}
	ours = ours[:newLen]
	theirs := *other.getOrig()
	for i := 0; i < len(theirs); i++ {
		ours[i+offset] += theirs[i]
	}
	*ms.getOrig() = ours
	return true
}
