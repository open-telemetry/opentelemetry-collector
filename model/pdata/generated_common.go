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

// Code generated by "model/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "go run model/internal/cmd/pdatagen/main.go".

package pdata

import (
	otlpcommon "go.opentelemetry.io/collector/model/internal/data/protogen/common/v1"
)

// InstrumentationLibrary is a message representing the instrumentation library information.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewInstrumentationLibrary function to create new instances.
// Important: zero-initialized instance is not valid for use.
//
type InstrumentationLibrary struct {
	orig *otlpcommon.InstrumentationLibrary
}

func newInstrumentationLibrary(orig *otlpcommon.InstrumentationLibrary) InstrumentationLibrary {
	if orig == nil {
		return NewInstrumentationLibrary()
	}
	return InstrumentationLibrary{orig: orig}
}

// NewInstrumentationLibrary creates a new empty InstrumentationLibrary.
//
// This must be used only in testing code. Users should use "AppendEmpty" when part of a Slice,
// OR directly access the member if this is embedded in another struct.
func NewInstrumentationLibrary() InstrumentationLibrary {
	return newInstrumentationLibrary(&otlpcommon.InstrumentationLibrary{})
}

// MoveTo moves all properties from the current struct to dest
// resetting the current instance to its zero value
func (ms InstrumentationLibrary) MoveTo(dest InstrumentationLibrary) {
	*dest.orig = *ms.orig
	*ms.orig = otlpcommon.InstrumentationLibrary{}
}

// Name returns the name associated with this InstrumentationLibrary.
func (ms InstrumentationLibrary) Name() string {
	return (*ms.orig).Name
}

// SetName replaces the name associated with this InstrumentationLibrary.
func (ms InstrumentationLibrary) SetName(v string) {
	(*ms.orig).Name = v
}

// Version returns the version associated with this InstrumentationLibrary.
func (ms InstrumentationLibrary) Version() string {
	return (*ms.orig).Version
}

// SetVersion replaces the version associated with this InstrumentationLibrary.
func (ms InstrumentationLibrary) SetVersion(v string) {
	(*ms.orig).Version = v
}

// CopyTo copies all properties from the current struct to the dest.
func (ms InstrumentationLibrary) CopyTo(dest InstrumentationLibrary) {
	dest.SetName(ms.Name())
	dest.SetVersion(ms.Version())
}

// AttributeValueSlice logically represents a slice of AttributeValue.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewAttributeValueSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type AttributeValueSlice struct {
	// orig points to the slice otlpcommon.AnyValue field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like EnsureCapacity.
	orig *[]*otlpcommon.AnyValue
}

func newAttributeValueSlice(orig *[]*otlpcommon.AnyValue) AttributeValueSlice {
	return AttributeValueSlice{orig}
}

// NewAttributeValueSlice creates a AttributeValueSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewAttributeValueSlice() AttributeValueSlice {
	orig := []*otlpcommon.AnyValue(nil)
	return AttributeValueSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewAttributeValueSlice()".
func (es AttributeValueSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es AttributeValueSlice) At(ix int) AttributeValue {
	return newAttributeValue((*es.orig)[ix])
}

// CopyTo copies all elements from the current slice to the dest.
func (es AttributeValueSlice) CopyTo(dest AttributeValueSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
	} else {
		(*dest.orig) = make([]*otlpcommon.AnyValue, srcLen)
		for i := 0; i < srcLen; i++ {
			(*dest.orig)[i] = &otlpcommon.AnyValue{}
		}
	}

	for i := range *es.orig {
		newAttributeValue((*es.orig)[i]).CopyTo(newAttributeValue((*dest.orig)[i]))
	}
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new AttributeValueSlice can be initialized:
//   es := NewAttributeValueSlice()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es AttributeValueSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlpcommon.AnyValue, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty AttributeValue.
// It returns the newly added AttributeValue.
func (es AttributeValueSlice) AppendEmpty() AttributeValue {
	*es.orig = append(*es.orig, &otlpcommon.AnyValue{})
	return es.At(es.Len() - 1)
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es AttributeValueSlice) MoveAndAppendTo(dest AttributeValueSlice) {
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
func (es AttributeValueSlice) RemoveIf(f func(AttributeValue) bool) {
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
