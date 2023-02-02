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

package pmetric

import (
	"sort"

	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

// ResourceMetricsSlice logically represents a slice of ResourceMetrics.
type ResourceMetricsSlice interface {
	commonResourceMetricsSlice
	At(ix int) ResourceMetrics
}

type MutableResourceMetricsSlice interface {
	commonResourceMetricsSlice
	At(ix int) MutableResourceMetrics
	EnsureCapacity(newCap int)
	AppendEmpty() MutableResourceMetrics
	Sort(less func(a, b MutableResourceMetrics) bool)
}

type commonResourceMetricsSlice interface {
	Len() int
	CopyTo(dest MutableResourceMetricsSlice)
	getOrig() *[]*otlpmetrics.ResourceMetrics
}

type immutableResourceMetricsSlice struct {
	orig *[]*otlpmetrics.ResourceMetrics
}

type mutableResourceMetricsSlice struct {
	immutableResourceMetricsSlice
}

func (es immutableResourceMetricsSlice) getOrig() *[]*otlpmetrics.ResourceMetrics {
	return es.orig
}

func newImmutableResourceMetricsSlice(orig *[]*otlpmetrics.ResourceMetrics) immutableResourceMetricsSlice {
	return immutableResourceMetricsSlice{orig}
}

func newMutableResourceMetricsSlice(orig *[]*otlpmetrics.ResourceMetrics) mutableResourceMetricsSlice {
	return mutableResourceMetricsSlice{immutableResourceMetricsSlice{orig}}
}

// NewResourceMetricsSlice creates a ResourceMetricsSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewResourceMetricsSlice() MutableResourceMetricsSlice {
	orig := []*otlpmetrics.ResourceMetrics(nil)
	return newMutableResourceMetricsSlice(&orig)
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewResourceMetricsSlice()".
func (es immutableResourceMetricsSlice) Len() int {
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
func (es immutableResourceMetricsSlice) At(ix int) ResourceMetrics {
	return newImmutableResourceMetrics((*es.getOrig())[ix])
}

func (es mutableResourceMetricsSlice) At(ix int) MutableResourceMetrics {
	return newMutableResourceMetrics((*es.getOrig())[ix])
}

// CopyTo copies all elements from the current slice overriding the destination.
func (es immutableResourceMetricsSlice) CopyTo(dest MutableResourceMetricsSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.getOrig())
	if srcLen <= destCap {
		(*dest.getOrig()) = (*dest.getOrig())[:srcLen:destCap]
		for i := range *es.getOrig() {
			newImmutableResourceMetrics((*es.getOrig())[i]).CopyTo(newMutableResourceMetrics((*dest.getOrig())[i]))
		}
		return
	}
	origs := make([]otlpmetrics.ResourceMetrics, srcLen)
	wrappers := make([]*otlpmetrics.ResourceMetrics, srcLen)
	for i := range *es.getOrig() {
		wrappers[i] = &origs[i]
		newImmutableResourceMetrics((*es.getOrig())[i]).CopyTo(newMutableResourceMetrics(wrappers[i]))
	}
	*dest.getOrig() = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new ResourceMetricsSlice can be initialized:
//
//	es := NewResourceMetricsSlice()
//	es.EnsureCapacity(4)
//	for i := 0; i < 4; i++ {
//	    e := es.AppendEmpty()
//	    // Here should set all the values for e.
//	}
func (es mutableResourceMetricsSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.getOrig())
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlpmetrics.ResourceMetrics, len(*es.getOrig()), newCap)
	copy(newOrig, *es.getOrig())
	*es.getOrig() = newOrig
}

// AppendEmpty will append to the end of the slice an empty ResourceMetrics.
// It returns the newly added ResourceMetrics.
func (es mutableResourceMetricsSlice) AppendEmpty() MutableResourceMetrics {
	*es.getOrig() = append(*es.getOrig(), &otlpmetrics.ResourceMetrics{})
	return es.At(es.Len() - 1)
}

// Sort sorts the ResourceMetrics elements within ResourceMetricsSlice given the
// provided less function so that two instances of ResourceMetricsSlice
// can be compared.
func (es mutableResourceMetricsSlice) Sort(less func(a, b MutableResourceMetrics) bool) {
	sort.SliceStable(*es.getOrig(), func(i, j int) bool { return less(es.At(i), es.At(j)) })
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es mutableResourceMetricsSlice) MoveAndAppendTo(dest mutableResourceMetricsSlice) {
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
func (es mutableResourceMetricsSlice) RemoveIf(f func(MutableResourceMetrics) bool) {
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

func generateTestResourceMetricsSlice() MutableResourceMetricsSlice {
	tv := NewResourceMetricsSlice()
	fillTestResourceMetricsSlice(tv)
	return tv
}

func fillTestResourceMetricsSlice(tv MutableResourceMetricsSlice) {
	*tv.orig = make([]*otlpmetrics.ResourceMetrics, 7)
	for i := 0; i < 7; i++ {
		(*tv.orig)[i] = &otlpmetrics.ResourceMetrics{}
		fillTestResourceMetrics(newResourceMetrics((*tv.orig)[i]))
	}
}
