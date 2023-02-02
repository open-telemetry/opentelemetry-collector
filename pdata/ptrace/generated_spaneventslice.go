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

// Code generated by "pdata/internal/cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "make genpdata".

package ptrace

import (
	"sort"

	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

// SpanEventSlice logically represents a slice of SpanEvent.
type SpanEventSlice interface {
	commonSpanEventSlice
	At(ix int) SpanEvent
}

type MutableSpanEventSlice interface {
	commonSpanEventSlice
	At(ix int) MutableSpanEvent
	EnsureCapacity(newCap int)
	AppendEmpty() MutableSpanEvent
	Sort(less func(a, b MutableSpanEvent) bool)
}

type commonSpanEventSlice interface {
	Len() int
	CopyTo(dest MutableSpanEventSlice)
	getOrig() *[]*otlptrace.Span_Event
}

type immutableSpanEventSlice struct {
	orig *[]*otlptrace.Span_Event
}

type mutableSpanEventSlice struct {
	immutableSpanEventSlice
}

func (es immutableSpanEventSlice) getOrig() *[]*otlptrace.Span_Event {
	return es.orig
}

func newImmutableSpanEventSlice(orig *[]*otlptrace.Span_Event) immutableSpanEventSlice {
	return immutableSpanEventSlice{orig}
}

func newMutableSpanEventSlice(orig *[]*otlptrace.Span_Event) mutableSpanEventSlice {
	return mutableSpanEventSlice{immutableSpanEventSlice{orig}}
}

// NewSpanEventSlice creates a SpanEventSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewSpanEventSlice() MutableSpanEventSlice {
	orig := []*otlptrace.Span_Event(nil)
	return newMutableSpanEventSlice(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewSpanEventSlice()".
func (es immutableSpanEventSlice) Len() int {
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
func (es immutableSpanEventSlice) At(ix int) SpanEvent {
	return newImmutableSpanEvent((*es.getOrig())[ix])
}

func (es mutableSpanEventSlice) At(ix int) MutableSpanEvent {
	return newMutableSpanEvent((*es.getOrig())[ix])
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es immutableSpanEventSlice) CopyTo(dest MutableSpanEventSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]
		for i := range *es.getOrig() {
			newImmutableSpanEvent((*es.getOrig())[i]).CopyTo(newMutableSpanEvent((*dest.getOrig())[i]))
		}
		return
	}
	origs := make([]otlptrace.Span_Event, srcLen)
	wrappers := make([]*otlptrace.Span_Event, srcLen)
	for i := range *es.getOrig() {
		wrappers[i] = &origs[i]
		newImmutableSpanEvent((*es.getOrig())[i]).CopyTo(newMutableSpanEvent(wrappers[i]))
	}
	*dest.getOrig() = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new SpanEventSlice can be initialized:
//
//	es := NewSpanEventSlice()
//	es.EnsureCapacity(4)
//	for i := 0; i < 4; i++ {
//	    e := es.AppendEmpty()
//	    // Here should set all the values for e.
//	}
func (es mutableSpanEventSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlptrace.Span_Event, len(*es.getOrig()), newCap)
	copy(newOrig, *es.getOrig())
	*es.getOrig() = newOrig
}

// AppendEmpty will append to the end of the slice an empty SpanEvent.
// It returns the newly added SpanEvent.
func (es mutableSpanEventSlice) AppendEmpty() MutableSpanEvent {
	*es.getOrig() = append(*es.getOrig(), &otlptrace.Span_Event{})
	return es.At(es.Len() - 1)
}

// Sort sorts the SpanEvent elements within SpanEventSlice given the
// provided less function so that two instances of SpanEventSlice
// can be compared.
func (es mutableSpanEventSlice) Sort(less func(a, b MutableSpanEvent) bool) {
	sort.SliceStable(*es.getOrig(), func(i, j int) bool { return less(es.At(i), es.At(j)) })
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es mutableSpanEventSlice) MoveAndAppendTo(dest mutableSpanEventSlice) {
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
func (es mutableSpanEventSlice) RemoveIf(f func(MutableSpanEvent) bool) {
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
	*es.orig = (*es.orig)[:newLen]
}

func generateTestSpanEventSlice() MutableSpanEventSlice {
	tv := NewSpanEventSlice()
	fillTestSpanEventSlice(tv)
	return tv
}

func fillTestSpanEventSlice(tv MutableSpanEventSlice) {
	*tv.orig = make([]*otlptrace.Span_Event, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlptrace.Span_Event{}
		fillTestSpanEvent(newSpanEvent((*tv.orig)[i]))
	}
}
