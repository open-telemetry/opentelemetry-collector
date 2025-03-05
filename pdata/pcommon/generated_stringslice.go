// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package pcommon

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
)

// StringSlice represents a []string slice.
// The instance of StringSlice can be assigned to multiple objects since it's immutable.
//
// Must use NewStringSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type StringSlice internal.StringSlice

func (ms StringSlice) getOrig() *[]string {
	return internal.GetOrigStringSlice(internal.StringSlice(ms))
}

func (ms StringSlice) getState() *internal.State {
	return internal.GetStringSliceState(internal.StringSlice(ms))
}

// NewStringSlice creates a new empty StringSlice.
func NewStringSlice() StringSlice {
	orig := []string(nil)
	state := internal.StateMutable
	return StringSlice(internal.NewStringSlice(&orig, &state))
}

// AsRaw returns a copy of the []string slice.
func (ms StringSlice) AsRaw() []string {
	return copyStringSlice(nil, *ms.getOrig())
}

// FromRaw copies raw []string into the slice StringSlice.
func (ms StringSlice) FromRaw(val []string) {
	ms.getState().AssertMutable()
	*ms.getOrig() = copyStringSlice(*ms.getOrig(), val)
}

// Len returns length of the []string slice value.
// Equivalent of len(stringSlice).
func (ms StringSlice) Len() int {
	return len(*ms.getOrig())
}

// At returns an item from particular index.
// Equivalent of stringSlice[i].
func (ms StringSlice) At(i int) string {
	return (*ms.getOrig())[i]
}

// SetAt sets string item at particular index.
// Equivalent of stringSlice[i] = val
func (ms StringSlice) SetAt(i int, val string) {
	ms.getState().AssertMutable()
	(*ms.getOrig())[i] = val
}

// EnsureCapacity ensures StringSlice has at least the specified capacity.
//  1. If the newCap <= cap, then is no change in capacity.
//  2. If the newCap > cap, then the slice capacity will be expanded to the provided value which will be equivalent of:
//     buf := make([]string, len(stringSlice), newCap)
//     copy(buf, stringSlice)
//     stringSlice = buf
func (ms StringSlice) EnsureCapacity(newCap int) {
	ms.getState().AssertMutable()
	oldCap := cap(*ms.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]string, len(*ms.getOrig()), newCap)
	copy(newOrig, *ms.getOrig())
	*ms.getOrig() = newOrig
}

// Append appends extra elements to StringSlice.
// Equivalent of stringSlice = append(stringSlice, elms...)
func (ms StringSlice) Append(elms ...string) {
	ms.getState().AssertMutable()
	*ms.getOrig() = append(*ms.getOrig(), elms...)
}

// MoveTo moves all elements from the current slice overriding the destination and
// resetting the current instance to its zero value.
func (ms StringSlice) MoveTo(dest StringSlice) {
	ms.getState().AssertMutable()
	dest.getState().AssertMutable()
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = nil
}

// CopyTo copies all elements from the current slice overriding the destination.
func (ms StringSlice) CopyTo(dest StringSlice) {
	dest.getState().AssertMutable()
	*dest.getOrig() = copyStringSlice(*dest.getOrig(), *ms.getOrig())
}

// Equal checks equality with a raw []string
func (ms StringSlice) Equal(val []string) bool {
	return slices.Equal(*ms.getOrig(), val)
}

func copyStringSlice(dst, src []string) []string {
	dst = dst[:0]
	return append(dst, src...)
}
