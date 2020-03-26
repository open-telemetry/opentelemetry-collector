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
	orig *[]*otlptrace.ResourceSpans
}

// TraceDataFromOtlp creates the internal TraceData representation from the OTLP.
func TraceDataFromOtlp(orig []*otlptrace.ResourceSpans) TraceData {
	return TraceData{&orig}
}

// TraceDataToOtlp converts the internal TraceData to the OTLP.
func TraceDataToOtlp(md TraceData) []*otlptrace.ResourceSpans {
	return *md.orig
}

// NewTraceData creates a new TraceData.
func NewTraceData() TraceData {
	orig := []*otlptrace.ResourceSpans(nil)
	return TraceData{&orig}
}

// SpanCount calculates the total number of spans.
func (td TraceData) SpanCount() int {
	spanCount := 0
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		ils := rss.Get(i).InstrumentationLibrarySpans()
		for j := 0; j < ils.Len(); j++ {
			spanCount += ils.Get(j).Spans().Len()
		}
	}
	return spanCount
}

func (td TraceData) ResourceSpans() ResourceSpansSlice {
	return newResourceSpansSlice(td.orig)
}

func (td TraceData) SetResourceSpans(v ResourceSpansSlice) {
	*td.orig = *v.orig
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
