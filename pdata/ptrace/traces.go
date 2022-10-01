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

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Traces is the top-level struct that is propagated through the traces pipeline.
// Use NewTraces to create new instance, zero-initialized instance is not valid for use.
type Traces internal.Traces

func newTraces(orig *otlpcollectortrace.ExportTraceServiceRequest) Traces {
	return Traces(internal.NewTraces(orig))
}

func (ms Traces) getOrig() *otlpcollectortrace.ExportTraceServiceRequest {
	return internal.GetOrigTraces(internal.Traces(ms))
}

// NewTraces creates a new Traces struct.
func NewTraces() Traces {
	return newTraces(&otlpcollectortrace.ExportTraceServiceRequest{})
}

// MoveTo moves all properties from the current struct to dest
// resetting the current instance to its zero value.
func (ms Traces) MoveTo(dest Traces) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpcollectortrace.ExportTraceServiceRequest{}
}

// CopyTo copies all logs from ms to dest.
func (ms Traces) CopyTo(dest Traces) {
	ms.ResourceSpans().CopyTo(dest.ResourceSpans())
}

// Clone returns a copy of Traces.
// Deprecated: [0.61.0] Replace with:
//
//	newTraces := NewTraces()
//	ms.CopyTo(newTraces)
func (ms Traces) Clone() Traces {
	cloneTd := NewTraces()
	ms.ResourceSpans().CopyTo(cloneTd.ResourceSpans())
	return cloneTd
}

// SpanCount calculates the total number of spans.
func (ms Traces) SpanCount() int {
	spanCount := 0
	rss := ms.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			spanCount += ilss.At(j).Spans().Len()
		}
	}
	return spanCount
}

// ResourceSpans returns the ResourceSpansSlice associated with this Metrics.
func (ms Traces) ResourceSpans() ResourceSpansSlice {
	return newResourceSpansSlice(&ms.getOrig().ResourceSpans)
}

// Deprecated: [v0.61.0] use TraceState().AsRaw().
func (ms Span) TraceStateStruct() pcommon.TraceState {
	return ms.TraceState()
}

// Deprecated: [v0.61.0] use TraceState().AsRaw().
func (ms SpanLink) TraceStateStruct() pcommon.TraceState {
	return ms.TraceState()
}

// SpanKind is the type of span. Can be used to specify additional relationships between spans
// in addition to a parent/child relationship.
type SpanKind int32

// String returns the string representation of the SpanKind.
func (sk SpanKind) String() string { return otlptrace.Span_SpanKind(sk).String() }

const (
	// SpanKindUnspecified represents that the SpanKind is unspecified, it MUST NOT be used.
	SpanKindUnspecified = SpanKind(otlptrace.Span_SPAN_KIND_UNSPECIFIED)
	// SpanKindInternal indicates that the span represents an internal operation within an application,
	// as opposed to an operation happening at the boundaries. Default value.
	SpanKindInternal = SpanKind(otlptrace.Span_SPAN_KIND_INTERNAL)
	// SpanKindServer indicates that the span covers server-side handling of an RPC or other
	// remote network request.
	SpanKindServer = SpanKind(otlptrace.Span_SPAN_KIND_SERVER)
	// SpanKindClient indicates that the span describes a request to some remote service.
	SpanKindClient = SpanKind(otlptrace.Span_SPAN_KIND_CLIENT)
	// SpanKindProducer indicates that the span describes a producer sending a message to a broker.
	// Unlike CLIENT and SERVER, there is often no direct critical path latency relationship
	// between producer and consumer spans.
	// A PRODUCER span ends when the message was accepted by the broker while the logical processing of
	// the message might span a much longer time.
	SpanKindProducer = SpanKind(otlptrace.Span_SPAN_KIND_PRODUCER)
	// SpanKindConsumer indicates that the span describes consumer receiving a message from a broker.
	// Like the PRODUCER kind, there is often no direct critical path latency relationship between
	// producer and consumer spans.
	SpanKindConsumer = SpanKind(otlptrace.Span_SPAN_KIND_CONSUMER)
)

// SpanStatusCode mirrors the codes defined at
// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#set-status
type SpanStatusCode int32

// Deprecated: [0.62.0] Use SpanStatusCode instead
type StatusCode = SpanStatusCode

const (
	SpanStatusCodeUnset = SpanStatusCode(otlptrace.Status_STATUS_CODE_UNSET)
	SpanStatusCodeOk    = SpanStatusCode(otlptrace.Status_STATUS_CODE_OK)
	SpanStatusCodeError = SpanStatusCode(otlptrace.Status_STATUS_CODE_ERROR)

	// Deprecated: [0.62.0] Use SpanStatusCodeUnset instead
	StatusCodeUnset = SpanStatusCodeUnset

	// Deprecated: [0.62.0] Use SpanStatusCodeOk instead
	StatusCodeOk = SpanStatusCodeOk

	// Deprecated: [0.62.0] Use SpanStatusCodeError instead
	StatusCodeError = SpanStatusCodeError
)

// String returns the string representation of the StatusCode.
func (sc StatusCode) String() string { return otlptrace.Status_StatusCode(sc).String() }
