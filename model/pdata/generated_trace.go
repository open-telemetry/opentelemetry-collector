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

// Code generated by "cmd/pdatagen/main.go". DO NOT EDIT.
// To regenerate this file run "go run cmd/pdatagen/main.go".

package pdata

import (
	"sort"
	
	otlptrace "go.opentelemetry.io/collector/model/internal/data/protogen/trace/v1"
)

// ResourceSpansSlice logically represents a slice of ResourceSpans.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewResourceSpansSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceSpansSlice struct {
	// orig points to the slice otlptrace.ResourceSpans field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like EnsureCapacity.
	orig *[]*otlptrace.ResourceSpans
}

func newResourceSpansSlice(orig *[]*otlptrace.ResourceSpans) ResourceSpansSlice {
	return ResourceSpansSlice{orig}
}

// NewResourceSpansSlice creates a ResourceSpansSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewResourceSpansSlice() ResourceSpansSlice {
	orig := []*otlptrace.ResourceSpans(nil)
	return ResourceSpansSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewResourceSpansSlice()".
func (es ResourceSpansSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es ResourceSpansSlice) At(ix int) ResourceSpans {
	return newResourceSpans((*es.orig)[ix])
}

// CopyTo copies all elements from the current slice to the dest.
func (es ResourceSpansSlice) CopyTo(dest ResourceSpansSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			newResourceSpans((*es.orig)[i]).CopyTo(newResourceSpans((*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlptrace.ResourceSpans, srcLen)
	wrappers := make([]*otlptrace.ResourceSpans, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		newResourceSpans((*es.orig)[i]).CopyTo(newResourceSpans(wrappers[i]))
	}
	*dest.orig = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new ResourceSpansSlice can be initialized:
//   es := NewResourceSpansSlice()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es ResourceSpansSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlptrace.ResourceSpans, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty ResourceSpans.
// It returns the newly added ResourceSpans.
func (es ResourceSpansSlice) AppendEmpty() ResourceSpans {
	*es.orig = append(*es.orig, &otlptrace.ResourceSpans{})
	return es.At(es.Len() - 1)
}

// Sort sorts the ResourceSpans elements within ResourceSpansSlice given the
// provided less function so that two instances of ResourceSpansSlice
// can be compared.
//
// Returns the same instance to allow nicer code like:
//   lessFunc := func(a, b ResourceSpans) bool {
//     return a.Name() < b.Name() // choose any comparison here
//   }
//   assert.EqualValues(t, expected.Sort(lessFunc), actual.Sort(lessFunc))
func (es ResourceSpansSlice) Sort(less func(a, b ResourceSpans) bool) ResourceSpansSlice {
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
	return es
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es ResourceSpansSlice) MoveAndAppendTo(dest ResourceSpansSlice) {
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
func (es ResourceSpansSlice) RemoveIf(f func(ResourceSpans) bool) {
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

// ResourceSpans is a collection of spans from a Resource.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewResourceSpans function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceSpans struct {
	orig *otlptrace.ResourceSpans
}

func newResourceSpans(orig *otlptrace.ResourceSpans) ResourceSpans {
	return ResourceSpans{orig: orig}
}

// NewResourceSpans creates a new empty ResourceSpans.
//
// This must be used only in testing code since no "Set" method available.
func NewResourceSpans() ResourceSpans {
	return newResourceSpans(&otlptrace.ResourceSpans{})
}

// Resource returns the resource associated with this ResourceSpans.
func (ms ResourceSpans) Resource() Resource {
	return newResource(&(*ms.orig).Resource)
}

// InstrumentationLibrarySpans returns the InstrumentationLibrarySpans associated with this ResourceSpans.
func (ms ResourceSpans) InstrumentationLibrarySpans() InstrumentationLibrarySpansSlice {
	return newInstrumentationLibrarySpansSlice(&(*ms.orig).InstrumentationLibrarySpans)
}

// CopyTo copies all properties from the current struct to the dest.
func (ms ResourceSpans) CopyTo(dest ResourceSpans) {
	ms.Resource().CopyTo(dest.Resource())
	ms.InstrumentationLibrarySpans().CopyTo(dest.InstrumentationLibrarySpans())
}

// InstrumentationLibrarySpansSlice logically represents a slice of InstrumentationLibrarySpans.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewInstrumentationLibrarySpansSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type InstrumentationLibrarySpansSlice struct {
	// orig points to the slice otlptrace.InstrumentationLibrarySpans field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like EnsureCapacity.
	orig *[]*otlptrace.InstrumentationLibrarySpans
}

func newInstrumentationLibrarySpansSlice(orig *[]*otlptrace.InstrumentationLibrarySpans) InstrumentationLibrarySpansSlice {
	return InstrumentationLibrarySpansSlice{orig}
}

// NewInstrumentationLibrarySpansSlice creates a InstrumentationLibrarySpansSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewInstrumentationLibrarySpansSlice() InstrumentationLibrarySpansSlice {
	orig := []*otlptrace.InstrumentationLibrarySpans(nil)
	return InstrumentationLibrarySpansSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewInstrumentationLibrarySpansSlice()".
func (es InstrumentationLibrarySpansSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es InstrumentationLibrarySpansSlice) At(ix int) InstrumentationLibrarySpans {
	return newInstrumentationLibrarySpans((*es.orig)[ix])
}

// CopyTo copies all elements from the current slice to the dest.
func (es InstrumentationLibrarySpansSlice) CopyTo(dest InstrumentationLibrarySpansSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			newInstrumentationLibrarySpans((*es.orig)[i]).CopyTo(newInstrumentationLibrarySpans((*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlptrace.InstrumentationLibrarySpans, srcLen)
	wrappers := make([]*otlptrace.InstrumentationLibrarySpans, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		newInstrumentationLibrarySpans((*es.orig)[i]).CopyTo(newInstrumentationLibrarySpans(wrappers[i]))
	}
	*dest.orig = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new InstrumentationLibrarySpansSlice can be initialized:
//   es := NewInstrumentationLibrarySpansSlice()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es InstrumentationLibrarySpansSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlptrace.InstrumentationLibrarySpans, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty InstrumentationLibrarySpans.
// It returns the newly added InstrumentationLibrarySpans.
func (es InstrumentationLibrarySpansSlice) AppendEmpty() InstrumentationLibrarySpans {
	*es.orig = append(*es.orig, &otlptrace.InstrumentationLibrarySpans{})
	return es.At(es.Len() - 1)
}

// Sort sorts the InstrumentationLibrarySpans elements within InstrumentationLibrarySpansSlice given the
// provided less function so that two instances of InstrumentationLibrarySpansSlice
// can be compared.
//
// Returns the same instance to allow nicer code like:
//   lessFunc := func(a, b InstrumentationLibrarySpans) bool {
//     return a.Name() < b.Name() // choose any comparison here
//   }
//   assert.EqualValues(t, expected.Sort(lessFunc), actual.Sort(lessFunc))
func (es InstrumentationLibrarySpansSlice) Sort(less func(a, b InstrumentationLibrarySpans) bool) InstrumentationLibrarySpansSlice {
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
	return es
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es InstrumentationLibrarySpansSlice) MoveAndAppendTo(dest InstrumentationLibrarySpansSlice) {
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
func (es InstrumentationLibrarySpansSlice) RemoveIf(f func(InstrumentationLibrarySpans) bool) {
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

// InstrumentationLibrarySpans is a collection of spans from a LibraryInstrumentation.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewInstrumentationLibrarySpans function to create new instances.
// Important: zero-initialized instance is not valid for use.
type InstrumentationLibrarySpans struct {
	orig *otlptrace.InstrumentationLibrarySpans
}

func newInstrumentationLibrarySpans(orig *otlptrace.InstrumentationLibrarySpans) InstrumentationLibrarySpans {
	return InstrumentationLibrarySpans{orig: orig}
}

// NewInstrumentationLibrarySpans creates a new empty InstrumentationLibrarySpans.
//
// This must be used only in testing code since no "Set" method available.
func NewInstrumentationLibrarySpans() InstrumentationLibrarySpans {
	return newInstrumentationLibrarySpans(&otlptrace.InstrumentationLibrarySpans{})
}

// InstrumentationLibrary returns the instrumentationlibrary associated with this InstrumentationLibrarySpans.
func (ms InstrumentationLibrarySpans) InstrumentationLibrary() InstrumentationLibrary {
	return newInstrumentationLibrary(&(*ms.orig).InstrumentationLibrary)
}

// Spans returns the Spans associated with this InstrumentationLibrarySpans.
func (ms InstrumentationLibrarySpans) Spans() SpanSlice {
	return newSpanSlice(&(*ms.orig).Spans)
}

// CopyTo copies all properties from the current struct to the dest.
func (ms InstrumentationLibrarySpans) CopyTo(dest InstrumentationLibrarySpans) {
	ms.InstrumentationLibrary().CopyTo(dest.InstrumentationLibrary())
	ms.Spans().CopyTo(dest.Spans())
}

// SpanSlice logically represents a slice of Span.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewSpanSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type SpanSlice struct {
	// orig points to the slice otlptrace.Span field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like EnsureCapacity.
	orig *[]*otlptrace.Span
}

func newSpanSlice(orig *[]*otlptrace.Span) SpanSlice {
	return SpanSlice{orig}
}

// NewSpanSlice creates a SpanSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewSpanSlice() SpanSlice {
	orig := []*otlptrace.Span(nil)
	return SpanSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewSpanSlice()".
func (es SpanSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es SpanSlice) At(ix int) Span {
	return newSpan((*es.orig)[ix])
}

// CopyTo copies all elements from the current slice to the dest.
func (es SpanSlice) CopyTo(dest SpanSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			newSpan((*es.orig)[i]).CopyTo(newSpan((*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlptrace.Span, srcLen)
	wrappers := make([]*otlptrace.Span, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		newSpan((*es.orig)[i]).CopyTo(newSpan(wrappers[i]))
	}
	*dest.orig = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new SpanSlice can be initialized:
//   es := NewSpanSlice()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es SpanSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlptrace.Span, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty Span.
// It returns the newly added Span.
func (es SpanSlice) AppendEmpty() Span {
	*es.orig = append(*es.orig, &otlptrace.Span{})
	return es.At(es.Len() - 1)
}

// Sort sorts the Span elements within SpanSlice given the
// provided less function so that two instances of SpanSlice
// can be compared.
//
// Returns the same instance to allow nicer code like:
//   lessFunc := func(a, b Span) bool {
//     return a.Name() < b.Name() // choose any comparison here
//   }
//   assert.EqualValues(t, expected.Sort(lessFunc), actual.Sort(lessFunc))
func (es SpanSlice) Sort(less func(a, b Span) bool) SpanSlice {
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
	return es
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es SpanSlice) MoveAndAppendTo(dest SpanSlice) {
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
func (es SpanSlice) RemoveIf(f func(Span) bool) {
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

// Span represents a single operation within a trace.
// See Span definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewSpan function to create new instances.
// Important: zero-initialized instance is not valid for use.
type Span struct {
	orig *otlptrace.Span
}

func newSpan(orig *otlptrace.Span) Span {
	return Span{orig: orig}
}

// NewSpan creates a new empty Span.
//
// This must be used only in testing code since no "Set" method available.
func NewSpan() Span {
	return newSpan(&otlptrace.Span{})
}

// TraceID returns the traceid associated with this Span.
func (ms Span) TraceID() TraceID {
	return TraceID{orig: ((*ms.orig).TraceId)}
}

// SetTraceID replaces the traceid associated with this Span.
func (ms Span) SetTraceID(v TraceID) {
	(*ms.orig).TraceId = v.orig
}

// SpanID returns the spanid associated with this Span.
func (ms Span) SpanID() SpanID {
	return SpanID{orig: ((*ms.orig).SpanId)}
}

// SetSpanID replaces the spanid associated with this Span.
func (ms Span) SetSpanID(v SpanID) {
	(*ms.orig).SpanId = v.orig
}

// TraceState returns the tracestate associated with this Span.
func (ms Span) TraceState() TraceState {
	return TraceState((*ms.orig).TraceState)
}

// SetTraceState replaces the tracestate associated with this Span.
func (ms Span) SetTraceState(v TraceState) {
	(*ms.orig).TraceState = string(v)
}

// ParentSpanID returns the parentspanid associated with this Span.
func (ms Span) ParentSpanID() SpanID {
	return SpanID{orig: ((*ms.orig).ParentSpanId)}
}

// SetParentSpanID replaces the parentspanid associated with this Span.
func (ms Span) SetParentSpanID(v SpanID) {
	(*ms.orig).ParentSpanId = v.orig
}

// Name returns the name associated with this Span.
func (ms Span) Name() string {
	return (*ms.orig).Name
}

// SetName replaces the name associated with this Span.
func (ms Span) SetName(v string) {
	(*ms.orig).Name = v
}

// Kind returns the kind associated with this Span.
func (ms Span) Kind() SpanKind {
	return SpanKind((*ms.orig).Kind)
}

// SetKind replaces the kind associated with this Span.
func (ms Span) SetKind(v SpanKind) {
	(*ms.orig).Kind = otlptrace.Span_SpanKind(v)
}

// StartTimestamp returns the starttimestamp associated with this Span.
func (ms Span) StartTimestamp() Timestamp {
	return Timestamp((*ms.orig).StartTimeUnixNano)
}

// SetStartTimestamp replaces the starttimestamp associated with this Span.
func (ms Span) SetStartTimestamp(v Timestamp) {
	(*ms.orig).StartTimeUnixNano = uint64(v)
}

// EndTimestamp returns the endtimestamp associated with this Span.
func (ms Span) EndTimestamp() Timestamp {
	return Timestamp((*ms.orig).EndTimeUnixNano)
}

// SetEndTimestamp replaces the endtimestamp associated with this Span.
func (ms Span) SetEndTimestamp(v Timestamp) {
	(*ms.orig).EndTimeUnixNano = uint64(v)
}

// Attributes returns the Attributes associated with this Span.
func (ms Span) Attributes() AttributeMap {
	return newAttributeMap(&(*ms.orig).Attributes)
}

// DroppedAttributesCount returns the droppedattributescount associated with this Span.
func (ms Span) DroppedAttributesCount() uint32 {
	return (*ms.orig).DroppedAttributesCount
}

// SetDroppedAttributesCount replaces the droppedattributescount associated with this Span.
func (ms Span) SetDroppedAttributesCount(v uint32) {
	(*ms.orig).DroppedAttributesCount = v
}

// Events returns the Events associated with this Span.
func (ms Span) Events() SpanEventSlice {
	return newSpanEventSlice(&(*ms.orig).Events)
}

// DroppedEventsCount returns the droppedeventscount associated with this Span.
func (ms Span) DroppedEventsCount() uint32 {
	return (*ms.orig).DroppedEventsCount
}

// SetDroppedEventsCount replaces the droppedeventscount associated with this Span.
func (ms Span) SetDroppedEventsCount(v uint32) {
	(*ms.orig).DroppedEventsCount = v
}

// Links returns the Links associated with this Span.
func (ms Span) Links() SpanLinkSlice {
	return newSpanLinkSlice(&(*ms.orig).Links)
}

// DroppedLinksCount returns the droppedlinkscount associated with this Span.
func (ms Span) DroppedLinksCount() uint32 {
	return (*ms.orig).DroppedLinksCount
}

// SetDroppedLinksCount replaces the droppedlinkscount associated with this Span.
func (ms Span) SetDroppedLinksCount(v uint32) {
	(*ms.orig).DroppedLinksCount = v
}

// Status returns the status associated with this Span.
func (ms Span) Status() SpanStatus {
	return newSpanStatus(&(*ms.orig).Status)
}

// CopyTo copies all properties from the current struct to the dest.
func (ms Span) CopyTo(dest Span) {
	dest.SetTraceID(ms.TraceID())
	dest.SetSpanID(ms.SpanID())
	dest.SetTraceState(ms.TraceState())
	dest.SetParentSpanID(ms.ParentSpanID())
	dest.SetName(ms.Name())
	dest.SetKind(ms.Kind())
	dest.SetStartTimestamp(ms.StartTimestamp())
	dest.SetEndTimestamp(ms.EndTimestamp())
	ms.Attributes().CopyTo(dest.Attributes())
	dest.SetDroppedAttributesCount(ms.DroppedAttributesCount())
	ms.Events().CopyTo(dest.Events())
	dest.SetDroppedEventsCount(ms.DroppedEventsCount())
	ms.Links().CopyTo(dest.Links())
	dest.SetDroppedLinksCount(ms.DroppedLinksCount())
	ms.Status().CopyTo(dest.Status())
}

// SpanEventSlice logically represents a slice of SpanEvent.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewSpanEventSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type SpanEventSlice struct {
	// orig points to the slice otlptrace.Span_Event field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like EnsureCapacity.
	orig *[]*otlptrace.Span_Event
}

func newSpanEventSlice(orig *[]*otlptrace.Span_Event) SpanEventSlice {
	return SpanEventSlice{orig}
}

// NewSpanEventSlice creates a SpanEventSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewSpanEventSlice() SpanEventSlice {
	orig := []*otlptrace.Span_Event(nil)
	return SpanEventSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewSpanEventSlice()".
func (es SpanEventSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es SpanEventSlice) At(ix int) SpanEvent {
	return newSpanEvent((*es.orig)[ix])
}

// CopyTo copies all elements from the current slice to the dest.
func (es SpanEventSlice) CopyTo(dest SpanEventSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			newSpanEvent((*es.orig)[i]).CopyTo(newSpanEvent((*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlptrace.Span_Event, srcLen)
	wrappers := make([]*otlptrace.Span_Event, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		newSpanEvent((*es.orig)[i]).CopyTo(newSpanEvent(wrappers[i]))
	}
	*dest.orig = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new SpanEventSlice can be initialized:
//   es := NewSpanEventSlice()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es SpanEventSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlptrace.Span_Event, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty SpanEvent.
// It returns the newly added SpanEvent.
func (es SpanEventSlice) AppendEmpty() SpanEvent {
	*es.orig = append(*es.orig, &otlptrace.Span_Event{})
	return es.At(es.Len() - 1)
}

// Sort sorts the SpanEvent elements within SpanEventSlice given the
// provided less function so that two instances of SpanEventSlice
// can be compared.
//
// Returns the same instance to allow nicer code like:
//   lessFunc := func(a, b SpanEvent) bool {
//     return a.Name() < b.Name() // choose any comparison here
//   }
//   assert.EqualValues(t, expected.Sort(lessFunc), actual.Sort(lessFunc))
func (es SpanEventSlice) Sort(less func(a, b SpanEvent) bool) SpanEventSlice {
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
	return es
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es SpanEventSlice) MoveAndAppendTo(dest SpanEventSlice) {
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
func (es SpanEventSlice) RemoveIf(f func(SpanEvent) bool) {
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

// SpanEvent is a time-stamped annotation of the span, consisting of user-supplied
// text description and key-value pairs. See OTLP for event definition.
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewSpanEvent function to create new instances.
// Important: zero-initialized instance is not valid for use.
type SpanEvent struct {
	orig *otlptrace.Span_Event
}

func newSpanEvent(orig *otlptrace.Span_Event) SpanEvent {
	return SpanEvent{orig: orig}
}

// NewSpanEvent creates a new empty SpanEvent.
//
// This must be used only in testing code since no "Set" method available.
func NewSpanEvent() SpanEvent {
	return newSpanEvent(&otlptrace.Span_Event{})
}

// Timestamp returns the timestamp associated with this SpanEvent.
func (ms SpanEvent) Timestamp() Timestamp {
	return Timestamp((*ms.orig).TimeUnixNano)
}

// SetTimestamp replaces the timestamp associated with this SpanEvent.
func (ms SpanEvent) SetTimestamp(v Timestamp) {
	(*ms.orig).TimeUnixNano = uint64(v)
}

// Name returns the name associated with this SpanEvent.
func (ms SpanEvent) Name() string {
	return (*ms.orig).Name
}

// SetName replaces the name associated with this SpanEvent.
func (ms SpanEvent) SetName(v string) {
	(*ms.orig).Name = v
}

// Attributes returns the Attributes associated with this SpanEvent.
func (ms SpanEvent) Attributes() AttributeMap {
	return newAttributeMap(&(*ms.orig).Attributes)
}

// DroppedAttributesCount returns the droppedattributescount associated with this SpanEvent.
func (ms SpanEvent) DroppedAttributesCount() uint32 {
	return (*ms.orig).DroppedAttributesCount
}

// SetDroppedAttributesCount replaces the droppedattributescount associated with this SpanEvent.
func (ms SpanEvent) SetDroppedAttributesCount(v uint32) {
	(*ms.orig).DroppedAttributesCount = v
}

// CopyTo copies all properties from the current struct to the dest.
func (ms SpanEvent) CopyTo(dest SpanEvent) {
	dest.SetTimestamp(ms.Timestamp())
	dest.SetName(ms.Name())
	ms.Attributes().CopyTo(dest.Attributes())
	dest.SetDroppedAttributesCount(ms.DroppedAttributesCount())
}

// SpanLinkSlice logically represents a slice of SpanLink.
//
// This is a reference type. If passed by value and callee modifies it, the
// caller will see the modification.
//
// Must use NewSpanLinkSlice function to create new instances.
// Important: zero-initialized instance is not valid for use.
type SpanLinkSlice struct {
	// orig points to the slice otlptrace.Span_Link field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like EnsureCapacity.
	orig *[]*otlptrace.Span_Link
}

func newSpanLinkSlice(orig *[]*otlptrace.Span_Link) SpanLinkSlice {
	return SpanLinkSlice{orig}
}

// NewSpanLinkSlice creates a SpanLinkSlice with 0 elements.
// Can use "EnsureCapacity" to initialize with a given capacity.
func NewSpanLinkSlice() SpanLinkSlice {
	orig := []*otlptrace.Span_Link(nil)
	return SpanLinkSlice{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "NewSpanLinkSlice()".
func (es SpanLinkSlice) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
//   for i := 0; i < es.Len(); i++ {
//       e := es.At(i)
//       ... // Do something with the element
//   }
func (es SpanLinkSlice) At(ix int) SpanLink {
	return newSpanLink((*es.orig)[ix])
}

// CopyTo copies all elements from the current slice to the dest.
func (es SpanLinkSlice) CopyTo(dest SpanLinkSlice) {
	srcLen := es.Len()
	destCap := cap(*dest.orig)
	if srcLen <= destCap {
		(*dest.orig) = (*dest.orig)[:srcLen:destCap]
		for i := range *es.orig {
			newSpanLink((*es.orig)[i]).CopyTo(newSpanLink((*dest.orig)[i]))
		}
		return
	}
	origs := make([]otlptrace.Span_Link, srcLen)
	wrappers := make([]*otlptrace.Span_Link, srcLen)
	for i := range *es.orig {
		wrappers[i] = &origs[i]
		newSpanLink((*es.orig)[i]).CopyTo(newSpanLink(wrappers[i]))
	}
	*dest.orig = wrappers
}

// EnsureCapacity is an operation that ensures the slice has at least the specified capacity.
// 1. If the newCap <= cap then no change in capacity.
// 2. If the newCap > cap then the slice capacity will be expanded to equal newCap.
//
// Here is how a new SpanLinkSlice can be initialized:
//   es := NewSpanLinkSlice()
//   es.EnsureCapacity(4)
//   for i := 0; i < 4; i++ {
//       e := es.AppendEmpty()
//       // Here should set all the values for e.
//   }
func (es SpanLinkSlice) EnsureCapacity(newCap int) {
	oldCap := cap(*es.orig)
	if newCap <= oldCap {
		return
	}

	newOrig := make([]*otlptrace.Span_Link, len(*es.orig), newCap)
	copy(newOrig, *es.orig)
	*es.orig = newOrig
}

// AppendEmpty will append to the end of the slice an empty SpanLink.
// It returns the newly added SpanLink.
func (es SpanLinkSlice) AppendEmpty() SpanLink {
	*es.orig = append(*es.orig, &otlptrace.Span_Link{})
	return es.At(es.Len() - 1)
}

// Sort sorts the SpanLink elements within SpanLinkSlice given the
// provided less function so that two instances of SpanLinkSlice
// can be compared.
//
// Returns the same instance to allow nicer code like:
//   lessFunc := func(a, b SpanLink) bool {
//     return a.Name() < b.Name() // choose any comparison here
//   }
//   assert.EqualValues(t, expected.Sort(lessFunc), actual.Sort(lessFunc))
func (es SpanLinkSlice) Sort(less func(a, b SpanLink) bool) SpanLinkSlice {
	sort.SliceStable(*es.orig, func(i, j int) bool { return less(es.At(i), es.At(j)) })
	return es
}

// MoveAndAppendTo moves all elements from the current slice and appends them to the dest.
// The current slice will be cleared.
func (es SpanLinkSlice) MoveAndAppendTo(dest SpanLinkSlice) {
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
func (es SpanLinkSlice) RemoveIf(f func(SpanLink) bool) {
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

// SpanLink is a pointer from the current span to another span in the same trace or in a
// different trace.
// See Link definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewSpanLink function to create new instances.
// Important: zero-initialized instance is not valid for use.
type SpanLink struct {
	orig *otlptrace.Span_Link
}

func newSpanLink(orig *otlptrace.Span_Link) SpanLink {
	return SpanLink{orig: orig}
}

// NewSpanLink creates a new empty SpanLink.
//
// This must be used only in testing code since no "Set" method available.
func NewSpanLink() SpanLink {
	return newSpanLink(&otlptrace.Span_Link{})
}

// TraceID returns the traceid associated with this SpanLink.
func (ms SpanLink) TraceID() TraceID {
	return TraceID{orig: ((*ms.orig).TraceId)}
}

// SetTraceID replaces the traceid associated with this SpanLink.
func (ms SpanLink) SetTraceID(v TraceID) {
	(*ms.orig).TraceId = v.orig
}

// SpanID returns the spanid associated with this SpanLink.
func (ms SpanLink) SpanID() SpanID {
	return SpanID{orig: ((*ms.orig).SpanId)}
}

// SetSpanID replaces the spanid associated with this SpanLink.
func (ms SpanLink) SetSpanID(v SpanID) {
	(*ms.orig).SpanId = v.orig
}

// TraceState returns the tracestate associated with this SpanLink.
func (ms SpanLink) TraceState() TraceState {
	return TraceState((*ms.orig).TraceState)
}

// SetTraceState replaces the tracestate associated with this SpanLink.
func (ms SpanLink) SetTraceState(v TraceState) {
	(*ms.orig).TraceState = string(v)
}

// Attributes returns the Attributes associated with this SpanLink.
func (ms SpanLink) Attributes() AttributeMap {
	return newAttributeMap(&(*ms.orig).Attributes)
}

// DroppedAttributesCount returns the droppedattributescount associated with this SpanLink.
func (ms SpanLink) DroppedAttributesCount() uint32 {
	return (*ms.orig).DroppedAttributesCount
}

// SetDroppedAttributesCount replaces the droppedattributescount associated with this SpanLink.
func (ms SpanLink) SetDroppedAttributesCount(v uint32) {
	(*ms.orig).DroppedAttributesCount = v
}

// CopyTo copies all properties from the current struct to the dest.
func (ms SpanLink) CopyTo(dest SpanLink) {
	dest.SetTraceID(ms.TraceID())
	dest.SetSpanID(ms.SpanID())
	dest.SetTraceState(ms.TraceState())
	ms.Attributes().CopyTo(dest.Attributes())
	dest.SetDroppedAttributesCount(ms.DroppedAttributesCount())
}

// SpanStatus is an optional final status for this span. Semantically, when Status was not
// set, that means the span ended without errors and to assume Status.Ok (code = 0).
//
// This is a reference type, if passed by value and callee modifies it the
// caller will see the modification.
//
// Must use NewSpanStatus function to create new instances.
// Important: zero-initialized instance is not valid for use.
type SpanStatus struct {
	orig *otlptrace.Status
}

func newSpanStatus(orig *otlptrace.Status) SpanStatus {
	return SpanStatus{orig: orig}
}

// NewSpanStatus creates a new empty SpanStatus.
//
// This must be used only in testing code since no "Set" method available.
func NewSpanStatus() SpanStatus {
	return newSpanStatus(&otlptrace.Status{})
}

// Code returns the code associated with this SpanStatus.
func (ms SpanStatus) Code() StatusCode {
	return StatusCode((*ms.orig).Code)
}

// Message returns the message associated with this SpanStatus.
func (ms SpanStatus) Message() string {
	return (*ms.orig).Message
}

// SetMessage replaces the message associated with this SpanStatus.
func (ms SpanStatus) SetMessage(v string) {
	(*ms.orig).Message = v
}

// CopyTo copies all properties from the current struct to the dest.
func (ms SpanStatus) CopyTo(dest SpanStatus) {
	dest.SetCode(ms.Code())
	dest.SetMessage(ms.Message())
}
