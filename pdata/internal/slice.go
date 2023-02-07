// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"go.uber.org/multierr"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type Slice struct {
	commonSlice
}

type MutableSlice struct {
	commonSlice
	preventConversion struct{} // nolint:unused
}

type commonSlice struct {
	orig *[]otlpcommon.AnyValue
}

func (es Slice) asMutable() MutableSlice {
	return MutableSlice{commonSlice: commonSlice{orig: es.orig}}
}

func (es MutableSlice) AsImmutable() Slice {
	return Slice{commonSlice{orig: es.orig}}
}

func NewSliceFromOrig(orig *[]otlpcommon.AnyValue) Slice {
	return Slice{commonSlice{orig}}
}

func NewMutableSliceFromOrig(orig *[]otlpcommon.AnyValue) MutableSlice {
	return MutableSlice{commonSlice: commonSlice{orig}}
}

func GenerateTestSlice() MutableSlice {
	orig := []otlpcommon.AnyValue{}
	tv := NewMutableSliceFromOrig(&orig)
	FillTestSlice(tv)
	return tv
}

func FillTestSlice(tv MutableSlice) {
	*tv.orig = make([]otlpcommon.AnyValue, 7)
	for i := 0; i < 7; i++ {
		FillTestValue(NewMutableValueFromOrig(&(*tv.orig)[i]))
	}
}

// NewMutableSlice creates a MutableSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewMutableSlice() MutableSlice {
	orig := []otlpcommon.AnyValue(nil)
	return NewMutableSliceFromOrig(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewSlice()".
func (es commonSlice) Len() int {
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
func (es Slice) At(ix int) Value {
	return NewValueFromOrig(&(*es.orig)[ix])
}

func (es MutableSlice) At(ix int) MutableValue {
	return NewMutableValueFromOrig(&(*es.orig)[ix])
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es commonSlice) CopyTo(dest MutableSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
	} else {
		(*dest.orig) = make([]otlpcommon.AnyValue, srcLen)
	}

	for i := range *es.orig {
		NewValueFromOrig(&(*es.orig)[i]).CopyTo(NewMutableValueFromOrig(&(*dest.orig)[i]))
	}
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new Slice can be initialized:
//
//	es := NewSlice()
//	es.EnsureCapacity(4)
//	for i := 0; i < 4; i++ {
//	    e := es.AppendEmpty()
//	    // Here should set all the values for e.
//	}
func (es MutableSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]otlpcommon.AnyValue, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty Value.
// It returns the newly added Value.
func (es MutableSlice) AppendEmpty() MutableValue {
	*es.orig = append(*es.orig, otlpcommon.AnyValue{})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es MutableSlice) MoveAndAppendTo(dest MutableSlice) {
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
func (es MutableSlice) RemoveIf(f func(MutableValue) bool) {
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
	// TODO: Prevent memory leak by erasing truncated values.
	*es.orig = (*es.orig)[:newLen]
}

// AsRaw return []any copy of the Slice.
func (es commonSlice) AsRaw() []any {
	rawSlice := make([]any, 0, es.Len())
	for i := 0; i < es.Len(); i++ {
		rawSlice = append(rawSlice, Slice{es}.At(i).AsRaw())
	}
	return rawSlice
}

// FromRaw copies []any into the Slice.
func (es MutableSlice) FromRaw(rawSlice []any) error {
	if len(rawSlice) == 0 {
		*es.orig = nil
		return nil
	}
	var errs error
	origs := make([]otlpcommon.AnyValue, len(rawSlice))
	for ix, iv := range rawSlice {
		errs = multierr.Append(errs, NewMutableValueFromOrig(&origs[ix]).FromRaw(iv))
	}
	*es.orig = origs
	return errs
}
