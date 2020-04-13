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
	"encoding/hex"

	"github.com/golang/protobuf/proto"
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
func TraceDataToOtlp(td TraceData) []*otlptrace.ResourceSpans {
	return *td.orig
}

// NewTraceData creates a new TraceData.
func NewTraceData() TraceData {
	orig := []*otlptrace.ResourceSpans(nil)
	return TraceData{&orig}
}

// Clone returns a copy of TraceData.
func (td TraceData) Clone() TraceData {
	otlp := TraceDataToOtlp(td)
	resourceSpansClones := make([]*otlptrace.ResourceSpans, 0, len(otlp))
	for _, resourceSpans := range otlp {
		resourceSpansClones = append(resourceSpansClones,
			proto.Clone(resourceSpans).(*otlptrace.ResourceSpans))
	}
	return TraceDataFromOtlp(resourceSpansClones)
}

// SpanCount calculates the total number of spans.
func (td TraceData) SpanCount() int {
	spanCount := 0
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		if rs.IsNil() {
			continue
		}
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			if ils.IsNil() {
				continue
			}
			spanCount += ilss.At(j).Spans().Len()
		}
	}
	return spanCount
}

func (td TraceData) ResourceSpans() ResourceSpansSlice {
	return newResourceSpansSlice(td.orig)
}

type TraceID []byte

// NewTraceID returns a new TraceID.
func NewTraceID(bytes []byte) TraceID { return bytes }

func (t TraceID) Bytes() []byte {
	return t
}

func (t TraceID) String() string { return hex.EncodeToString(t) }

type SpanID []byte

// NewSpanID returns a new SpanID.
func NewSpanID(bytes []byte) SpanID { return bytes }

func (s SpanID) Bytes() []byte {
	return s
}

func (s SpanID) String() string { return hex.EncodeToString(s) }

// TraceState in w3c-trace-context format: https://www.w3.org/TR/trace-context/#tracestate-header
type TraceState string

type SpanKind otlptrace.Span_SpanKind

func (sk SpanKind) String() string { return otlptrace.Span_SpanKind(sk).String() }

const (
	SpanKindUNSPECIFIED = SpanKind(0)
	SpanKindINTERNAL    = SpanKind(otlptrace.Span_INTERNAL)
	SpanKindSERVER      = SpanKind(otlptrace.Span_SERVER)
	SpanKindCLIENT      = SpanKind(otlptrace.Span_CLIENT)
	SpanKindPRODUCER    = SpanKind(otlptrace.Span_PRODUCER)
	SpanKindCONSUMER    = SpanKind(otlptrace.Span_CONSUMER)
)

// StatusCode mirrors the codes defined at
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-tracing.md#statuscanonicalcode
// and is numerically equal to Standard GRPC codes https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
type StatusCode otlptrace.Status_StatusCode

func (sc StatusCode) String() string { return otlptrace.Status_StatusCode(sc).String() }
