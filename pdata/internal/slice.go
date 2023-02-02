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

type commonSlice interface {
	Len() int
	CopyTo(dest MutableSlice)
	AsRaw() []any
	getOrig() *[]otlpcommon.AnyValue
}

type Slice interface {
	commonSlice
	At(ix int) Value
}

type MutableSlice interface {
	commonSlice
	At(ix int) MutableValue
	EnsureCapacity(newCap int)
	AppendEmpty() MutableValue
	MoveAndAppendTo(dest MutableSlice)
	RemoveIf(f func(MutableValue) bool)
	FromRaw(rawSlice []any) error
}

type immutableSlice struct {
	orig *[]otlpcommon.AnyValue
}

type mutableSlice struct {
	immutableSlice
}

func NewImmutableSlice(orig *[]otlpcommon.AnyValue) Slice {
	return immutableSlice{orig}
}

func NewMutableSlice(orig *[]otlpcommon.AnyValue) MutableSlice {
	return mutableSlice{immutableSlice{orig}}
}

func GenerateTestSlice() MutableSlice {
	orig := []otlpcommon.AnyValue{}
	tv := NewMutableSlice(&orig)
	FillTestSlice(tv)
	return tv
}

func FillTestSlice(tv MutableSlice) {
	*tv.getOrig() = make([]otlpcommon.AnyValue, 7)
	for i := 0; i < 7; i++ {
		FillTestValue(NewMutableValue(&(*tv.getOrig())[i]))
	}
}

func (es immutableSlice) getOrig() *[]otlpcommon.AnyValue {
	return es.orig
}

// NewSlice creates a Slice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewSlice() MutableSlice {
	orig := []otlpcommon.AnyValue(nil)
	return NewMutableSlice(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewSlice()".
func (es immutableSlice) Len() int {
	return len(*es.getOrig())
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//
//	for i := 0; i < es.Len(); i++ {
//	    e := es.At(i)
//	    ... // Do something with the element
//	}
func (es immutableSlice) At(ix int) Value {
	return immutableValue{&(*es.getOrig())[ix]}
}

func (es mutableSlice) At(ix int) MutableValue {
	return mutableValue{es.immutableSlice.At(ix).(immutableValue)}
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es immutableSlice) CopyTo(dest MutableSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]
	} else {
		(*dest.getOrig()) = make([]otlpcommon.AnyValue, srcLen)
	}

	for i := range *es.getOrig() {
		NewImmutableValue(&(*es.getOrig())[i]).CopyTo(NewMutableValue(&(*dest.getOrig())[i]))
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
func (es mutableSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]otlpcommon.AnyValue, len(*es.getOrig()), newCap)
	copy(newOrig, *es.getOrig())
	*es.getOrig() = newOrig
}

// AppendEmpty will append to the end of the slice an empty Value.
// It returns the newly added Value.
func (es mutableSlice) AppendEmpty() MutableValue {
	*es.getOrig() = append(*es.getOrig(), otlpcommon.AnyValue{})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es mutableSlice) MoveAndAppendTo(dest MutableSlice) {
	if *dest.getOrig() == nil {
		// We can simply move the entire vector and avoid any allocations.
		*dest.getOrig() = *es.getOrig()
	} else {
		*dest.getOrig() = append(*dest.getOrig(), *es.getOrig()...)
	}
	*es.getOrig() = nil
}

// RemoveIf calls f sequentially for each element present in the slice.
// If f returns true, the element is removed from the slice.
func (es mutableSlice) RemoveIf(f func(MutableValue) bool) {
	newLen := 0
	for i := 0; i < len(*es.getOrig()); i++ {
		if f(es.At(i)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*es.getOrig())[newLen] = (*es.getOrig())[i]
		newLen++
	}
	// TODO: Prevent memory leak by erasing truncated values.
	*es.getOrig() = (*es.getOrig())[:newLen]
}

// AsRaw return []any copy of the Slice.
func (es immutableSlice) AsRaw() []any {
	rawSlice := make([]any, 0, es.Len())
	for i := 0; i < es.Len(); i++ {
		rawSlice = append(rawSlice, es.At(i).AsRaw())
	}
	return rawSlice
}

// FromRaw copies []any into the Slice.
func (es mutableSlice) FromRaw(rawSlice []any) error {
	if len(rawSlice) == 0 {
		*es.getOrig() = nil
		return nil
	}
	var errs error
	origs := make([]otlpcommon.AnyValue, len(rawSlice))
	for ix, iv := range rawSlice {
		errs = multierr.Append(errs, NewMutableValue(&origs[ix]).FromRaw(iv))
	}
	*es.getOrig() = origs
	return errs
}
