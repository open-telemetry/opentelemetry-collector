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

// NewTraces creates a new Traces.
func NewTraces() Traces {
	orig := []*otlptrace.ResourceSpans(nil)
	return Traces{&orig}
}

// TracesFromOtlp creates the internal Traces representation from the OTLP.
func TracesFromOtlp(orig []*otlptrace.ResourceSpans) Traces {
	return Traces{&orig}
}

// TracesToOtlp converts the internal Traces to the OTLP.
func TracesToOtlp(td Traces) []*otlptrace.ResourceSpans {
	return *td.orig
}

// ToOtlpProtoBytes converts the internal Traces to OTLP Collector
// ExportTraceServiceRequest ProtoBuf bytes.
func (td Traces) ToOtlpProtoBytes() ([]byte, error) {
	traces := otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: *td.orig,
	}
	return traces.Marshal()
}

// FromOtlpProtoBytes converts OTLP Collector ExportTraceServiceRequest
// ProtoBuf bytes to the internal Traces. Overrides current data.
// Calling this function on zero-initialized structure causes panic.
// Use it with NewTraces or on existing initialized Traces.
func (td Traces) FromOtlpProtoBytes(data []byte) error {
	traces := otlpcollectortrace.ExportTraceServiceRequest{}
	if err := traces.Unmarshal(data); err != nil {
		return err
	}
	*td.orig = traces.ResourceSpans
	return nil
}

// Clone returns a copy of Traces.
func (td Traces) Clone() Traces {
	rss := NewResourceSpansSlice()
	td.ResourceSpans().CopyTo(rss)
	return Traces(rss)
}

// SpanCount calculates the total number of spans.
func (td Traces) SpanCount() int {
	spanCount := 0
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
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

// DeprecatedStatusCode is the deprecated status code used previously.
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/api.md#set-status
// Deprecated: use StatusCode instead.
type DeprecatedStatusCode otlptrace.Status_DeprecatedStatusCode

const (
	DeprecatedStatusCodeOk                 = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_OK)
	DeprecatedStatusCodeCancelled          = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_CANCELLED)
	DeprecatedStatusCodeUnknownError       = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR)
	DeprecatedStatusCodeInvalidArgument    = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_INVALID_ARGUMENT)
	DeprecatedStatusCodeDeadlineExceeded   = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_DEADLINE_EXCEEDED)
	DeprecatedStatusCodeNotFound           = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_NOT_FOUND)
	DeprecatedStatusCodeAlreadyExists      = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_ALREADY_EXISTS)
	DeprecatedStatusCodePermissionDenied   = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_PERMISSION_DENIED)
	DeprecatedStatusCodeResourceExhausted  = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_RESOURCE_EXHAUSTED)
	DeprecatedStatusCodeFailedPrecondition = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_FAILED_PRECONDITION)
	DeprecatedStatusCodeAborted            = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_ABORTED)
	DeprecatedStatusCodeOutOfRange         = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_OUT_OF_RANGE)
	DeprecatedStatusCodeUnimplemented      = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_UNIMPLEMENTED)
	DeprecatedStatusCodeInternalError      = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_INTERNAL_ERROR)
	DeprecatedStatusCodeUnavailable        = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_UNAVAILABLE)
	DeprecatedStatusCodeDataLoss           = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_DATA_LOSS)
	DeprecatedStatusCodeUnauthenticated    = DeprecatedStatusCode(otlptrace.Status_DEPRECATED_STATUS_CODE_UNAUTHENTICATED)
)

func (sc DeprecatedStatusCode) String() string {
	return otlptrace.Status_DeprecatedStatusCode(sc).String()
}

// StatusCode mirrors the codes defined at
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/api.md#set-status
type StatusCode otlptrace.Status_StatusCode

const (
	StatusCodeUnset = StatusCode(otlptrace.Status_STATUS_CODE_UNSET)
	StatusCodeOk    = StatusCode(otlptrace.Status_STATUS_CODE_OK)
	StatusCodeError = StatusCode(otlptrace.Status_STATUS_CODE_ERROR)
)

func (sc StatusCode) String() string { return otlptrace.Status_StatusCode(sc).String() }

// SetCode replaces the code associated with this SpanStatus.
func (ms SpanStatus) SetCode(v StatusCode) {
	ms.orig.Code = otlptrace.Status_StatusCode(v)

	// According to OTLP spec we also need to set the deprecated_code field.
	// See https://github.com/open-telemetry/opentelemetry-proto/blob/59c488bfb8fb6d0458ad6425758b70259ff4a2bd/opentelemetry/proto/trace/v1/trace.proto#L231
	//
	//   if code==STATUS_CODE_UNSET then `deprecated_code` MUST be
	//   set to DEPRECATED_STATUS_CODE_OK.
	//
	//   if code==STATUS_CODE_OK then `deprecated_code` MUST be
	//   set to DEPRECATED_STATUS_CODE_OK.
	//
	//   if code==STATUS_CODE_ERROR then `deprecated_code` MUST be
	//   set to DEPRECATED_STATUS_CODE_UNKNOWN_ERROR.
	switch v {
	case StatusCodeUnset, StatusCodeOk:
		ms.SetDeprecatedCode(DeprecatedStatusCodeOk)
	case StatusCodeError:
		ms.SetDeprecatedCode(DeprecatedStatusCodeUnknownError)
	}
}
