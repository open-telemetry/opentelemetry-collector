// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pcommon

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/internal"
)

// Int32Slice represents a []int32 slice.
// The instance of Int32Slice can be assigned to multiple objects since it's immutable.
//
// Must use NewInt32Slice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type Int32Slice internal.Int32Slice

func (ms Int32Slice) getOrig() *[]int32 {
	return internal.GetOrigInt32Slice(internal.Int32Slice(ms))
}

func (ms Int32Slice) getState() *internal.State {
	return internal.GetInt32SliceState(internal.Int32Slice(ms))
}

// NewInt32Slice creates a new empty Int32Slice.
func NewInt32Slice() Int32Slice {
	orig := []int32(nil)
	state := internal.StateMutable
	return Int32Slice(internal.NewInt32Slice(&orig, &state))
}

// AsRaw returns a copy of the []int32 slice.
func (ms Int32Slice) AsRaw() []int32 {
	return copyInt32Slice(nil, *ms.getOrig())
}

// FromRaw copies raw []int32 into the slice Int32Slice.
func (ms Int32Slice) FromRaw(val []int32) {
	ms.getState().AssertMutable()
	*ms.getOrig() = copyInt32Slice(*ms.getOrig(), val)
}

// Len returns length of the []int32 slice value.
// Equivalent of len(int32Slice).
func (ms Int32Slice) Len() int {
	return len(*ms.getOrig())
}

// At returns an item from particular index.
// Equivalent of int32Slice[i].
func (ms Int32Slice) At(i int) int32 {
	return (*ms.getOrig())[i]
}

// SetAt sets int32 item at particular index.
// Equivalent of int32Slice[i] = val
func (ms Int32Slice) SetAt(i int, val int32) {
	ms.getState().AssertMutable()
	(*ms.getOrig())[i] = val
}

// EnsureCapacity ensures Int32Slice has at least the specified capacity.
//  1. If the newCap <= cap, then is no change in capacity.
//  2. If the newCap > cap, then the slice capacity will be expanded to the provided value which will be equivalent of:
//     buf := make([]int32, len(int32Slice), newCap)
//     copy(buf, int32Slice)
//     int32Slice = buf
func (ms Int32Slice) EnsureCapacity(newCap int) {
	ms.getState().AssertMutable()
	oldCap := cap(*ms.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]int32, len(*ms.getOrig()), newCap)
	copy(newOrig, *ms.getOrig())
	*ms.getOrig() = newOrig
}

// Append appends extra elements to Int32Slice.
// Equivalent of int32Slice = append(int32Slice, elms...)
func (ms Int32Slice) Append(elms ...int32) {
	ms.getState().AssertMutable()
	*ms.getOrig() = append(*ms.getOrig(), elms...)
}

// MoveTo moves all elements from the current slice overriding the destination and
// resetting the current instance to its zero value.
func (ms Int32Slice) MoveTo(dest Int32Slice) {
	ms.getState().AssertMutable()
	dest.getState().AssertMutable()
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = nil
}

// CopyTo copies all elements from the current slice overriding the destination.
func (ms Int32Slice) CopyTo(dest Int32Slice) {
	dest.getState().AssertMutable()
	*dest.getOrig() = copyInt32Slice(*dest.getOrig(), *ms.getOrig())
}

func copyInt32Slice(dst, src []int32) []int32 {
	dst = dst[:0]
	return append(dst, src...)
}

// IncrementFrom increments all elements by the elements from another slice.
func (ms Int32Slice) IncrementFrom(other Int32Slice, offset int) bool {
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

// Collapse merges (sums) n adjacent buckets and reslices to account for the decreased length
//
//	n=2 offset=1
//	before:  1  1 1  1 1  1 1  1
//	        V    V    V    V    V
//	after:  1    2    2    2    1
func (ms Int32Slice) Collapse(n int, offset int) {
	ms.getState().AssertMutable()
	if offset >= n || offset < 0 {
		panic(fmt.Sprintf("offset %d must be positive and smaller than n %d", offset, n))
	}
	if n < 2 {
		return
	}
	orig := *ms.getOrig()
	newLen := (len(orig) + offset) / n
	if (len(orig)+offset)%n != 0 {
		newLen++
	}

	for i := 0; i < newLen; i++ {
		if offset == 0 || i > 0 {
			orig[i] = orig[i*n-offset]
		}
		for j := i*n + 1 - offset; j < i*n+n-offset && j < len(orig); j++ {
			if j > 0 {
				orig[i] += orig[j]
			}
		}
	}

	*ms.getOrig() = orig[:newLen]
}
