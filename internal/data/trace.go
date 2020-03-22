// Copyright 2020 OpenTelemetry Authors
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

package data

import (
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

// This file defines in-memory data structures to represent traces (spans).

// TraceData is the top-level struct that is propagated through the traces pipeline.
// This is the newer version of consumerdata.TraceData, but uses more efficient
// in-memory representation.
type TraceData struct {
	resourceSpans []*ResourceSpans
}

func NewTraceData(resourceSpans []*ResourceSpans) TraceData {
	return TraceData{resourceSpans}
}

// SpanCount calculates the total number of spans.
func (td TraceData) SpanCount() int {
	spanCount := 0
	for _, rs := range td.resourceSpans {
		for _, ils := range rs.ils {
			spanCount += len(ils.spans)
		}
	}
	return spanCount
}

func (td TraceData) ResourceSpans() []*ResourceSpans {
	return td.resourceSpans
}

// A collection of spans from a Resource.
//
// Must use NewResourceSpans functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type ResourceSpans struct {
	// The resource for the spans in this message.
	// If this field is not set then no resource info is known.
	resource Resource

	// A list of Spans that originate from a resource.
	ils []*InstrumentationLibrarySpans
}

func NewResourceSpans(resource Resource, ils []*InstrumentationLibrarySpans) *ResourceSpans {
	return &ResourceSpans{resource, ils}
}

func (m *ResourceSpans) Resource() Resource {
	return m.resource
}

func (m *ResourceSpans) SetResource(r Resource) {
	m.resource = r
}

func (m *ResourceSpans) InstrumentationLibrarySpans() []*InstrumentationLibrarySpans {
	return m.ils
}

func (m *ResourceSpans) SetInstrumentationLibrarySpans(s []*InstrumentationLibrarySpans) {
	m.ils = s
}

// InstrumentationLibrarySpans represents a collection of spans from a InstrumentationLibrary.
//
// Must use NewInstrumentationLibrarySpans functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type InstrumentationLibrarySpans struct {
	// The InstrumentationLibrary for the spans in this message.
	// If this field is not set then no resource info is known.
	instrumentationLibrary InstrumentationLibrary

	// A list of Spans that originate from a resource.
	spans []Span
}

func NewInstrumentationLibrarySpans(il InstrumentationLibrary, spans []Span) *InstrumentationLibrarySpans {
	return &InstrumentationLibrarySpans{il, spans}
}

func (ils *InstrumentationLibrarySpans) InstrumentationLibrary() InstrumentationLibrary {
	return ils.instrumentationLibrary
}

func (ils *InstrumentationLibrarySpans) SetInstrumentationLibrary(il InstrumentationLibrary) {
	ils.instrumentationLibrary = il
}

func (ils *InstrumentationLibrarySpans) Spans() []Span {
	return ils.spans
}

func (ils *InstrumentationLibrarySpans) SetSpans(s []Span) {
	ils.spans = s
}

type TraceID []byte

func (t TraceID) Bytes() []byte {
	return t
}

func NewTraceID(bytes []byte) TraceID { return TraceID(bytes) }

type SpanID []byte

func (s SpanID) Bytes() []byte {
	return s
}

func NewSpanID(bytes []byte) SpanID { return SpanID(bytes) }

// TraceState in w3c-trace-context format: https://www.w3.org/TR/trace-context/#tracestate-header
type TraceState string

type SpanKind otlptrace.Span_SpanKind

func (sk SpanKind) String() string { return otlptrace.Span_SpanKind(sk).String() }

const (
	SpanKindUNSPECIFIED SpanKind = 0
	SpanKindINTERNAL    SpanKind = SpanKind(otlptrace.Span_INTERNAL)
	SpanKindSERVER      SpanKind = SpanKind(otlptrace.Span_SERVER)
	SpanKindCLIENT      SpanKind = SpanKind(otlptrace.Span_CLIENT)
	SpanKindPRODUCER    SpanKind = SpanKind(otlptrace.Span_PRODUCER)
	SpanKindCONSUMER    SpanKind = SpanKind(otlptrace.Span_CONSUMER)
)

// StatusCode mirrors the codes defined at
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-tracing.md#statuscanonicalcode
// and is numerically equal to Standard GRPC codes https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
type StatusCode otlptrace.Status_StatusCode

// NewSpanSlice creates a slice of pointers to Spans that are correctly initialized.
func NewSpanSlice(len int) []Span {
	// Slice for underlying data.
	origs := make([]otlptrace.Span, len)

	// Slice for wrappers.
	wrappers := make([]Span, len)

	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}
