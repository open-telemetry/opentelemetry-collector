// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pdata

import (
	"encoding/hex"

	"github.com/gogo/protobuf/proto"

	otlpcollectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
)

// This file defines in-memory data structures to represent traces (spans).

// Traces is the top-level struct that is propagated through the traces pipeline.
// This is the newer version of consumerdata.Traces, but uses more efficient
// in-memory representation.
type Traces struct {
	orig *[]*otlptrace.ResourceSpans
}

// TracesFromOtlp creates the internal Traces representation from the OTLP.
func TracesFromOtlp(orig []*otlptrace.ResourceSpans) Traces {
	return Traces{&orig}
}

// TracesToOtlp converts the internal Traces to the OTLP.
func TracesToOtlp(td Traces) []*otlptrace.ResourceSpans {
	return *td.orig
}

// ToOtlpProtoBytes returns the internal Traces to OTLP Collector
// ExportTraceServiceRequest ProtoBuf bytes. This is intended to export OTLP
// Protobuf bytes for OTLP/HTTP transports.
func (td Traces) ToOtlpProtoBytes() ([]byte, error) {
	return proto.Marshal(&otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: *td.orig,
	})
}

// NewTraces creates a new Traces.
func NewTraces() Traces {
	orig := []*otlptrace.ResourceSpans(nil)
	return Traces{&orig}
}

// Clone returns a copy of Traces.
func (td Traces) Clone() Traces {
	otlp := TracesToOtlp(td)
	resourceSpansClones := make([]*otlptrace.ResourceSpans, 0, len(otlp))
	for _, resourceSpans := range otlp {
		resourceSpansClones = append(resourceSpansClones,
			proto.Clone(resourceSpans).(*otlptrace.ResourceSpans))
	}
	return TracesFromOtlp(resourceSpansClones)
}

// SpanCount calculates the total number of spans.
func (td Traces) SpanCount() int {
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

// Size returns size in bytes.
func (td Traces) Size() int {
	size := 0
	for i := 0; i < len(*td.orig); i++ {
		if (*td.orig)[i] == nil {
			continue
		}
		size += (*td.orig)[i].Size()
	}
	return size
}

func (td Traces) ResourceSpans() ResourceSpansSlice {
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
	TraceStateEmpty TraceState = ""
)

const (
	SpanKindUNSPECIFIED = SpanKind(0)
	SpanKindINTERNAL    = SpanKind(otlptrace.Span_SPAN_KIND_INTERNAL)
	SpanKindSERVER      = SpanKind(otlptrace.Span_SPAN_KIND_SERVER)
	SpanKindCLIENT      = SpanKind(otlptrace.Span_SPAN_KIND_CLIENT)
	SpanKindPRODUCER    = SpanKind(otlptrace.Span_SPAN_KIND_PRODUCER)
	SpanKindCONSUMER    = SpanKind(otlptrace.Span_SPAN_KIND_CONSUMER)
)

// StatusCode mirrors the codes defined at
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/api-tracing.md#statuscanonicalcode
// and is numerically equal to Standard GRPC codes https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
type StatusCode otlptrace.Status_StatusCode

const (
	StatusCodeOk                 = StatusCode(otlptrace.Status_STATUS_CODE_OK)
	StatusCodeCancelled          = StatusCode(otlptrace.Status_STATUS_CODE_CANCELLED)
	StatusCodeUnknownError       = StatusCode(otlptrace.Status_STATUS_CODE_UNKNOWN_ERROR)
	StatusCodeInvalidArgument    = StatusCode(otlptrace.Status_STATUS_CODE_INVALID_ARGUMENT)
	StatusCodeDeadlineExceeded   = StatusCode(otlptrace.Status_STATUS_CODE_DEADLINE_EXCEEDED)
	StatusCodeNotFound           = StatusCode(otlptrace.Status_STATUS_CODE_NOT_FOUND)
	StatusCodeAlreadyExists      = StatusCode(otlptrace.Status_STATUS_CODE_ALREADY_EXISTS)
	StatusCodePermissionDenied   = StatusCode(otlptrace.Status_STATUS_CODE_PERMISSION_DENIED)
	StatusCodeResourceExhausted  = StatusCode(otlptrace.Status_STATUS_CODE_RESOURCE_EXHAUSTED)
	StatusCodeFailedPrecondition = StatusCode(otlptrace.Status_STATUS_CODE_FAILED_PRECONDITION)
	StatusCodeAborted            = StatusCode(otlptrace.Status_STATUS_CODE_ABORTED)
	StatusCodeOutOfRange         = StatusCode(otlptrace.Status_STATUS_CODE_OUT_OF_RANGE)
	StatusCodeUnimplemented      = StatusCode(otlptrace.Status_STATUS_CODE_UNIMPLEMENTED)
	StatusCodeInternalError      = StatusCode(otlptrace.Status_STATUS_CODE_INTERNAL_ERROR)
	StatusCodeUnavailable        = StatusCode(otlptrace.Status_STATUS_CODE_UNAVAILABLE)
	StatusCodeDataLoss           = StatusCode(otlptrace.Status_STATUS_CODE_DATA_LOSS)
	StatusCodeUnauthenticated    = StatusCode(otlptrace.Status_STATUS_CODE_UNAUTHENTICATED)
)

func (sc StatusCode) String() string { return otlptrace.Status_StatusCode(sc).String() }
