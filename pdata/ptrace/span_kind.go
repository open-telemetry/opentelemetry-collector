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
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

// SpanKind is the type of span. Can be used to specify additional relationships between spans
// in addition to a parent/child relationship.
type SpanKind int32

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

// String returns the string representation of the SpanKind.
func (sk SpanKind) String() string {
	switch sk {
	case SpanKindUnspecified:
		return "Unspecified"
	case SpanKindInternal:
		return "Internal"
	case SpanKindServer:
		return "Server"
	case SpanKindClient:
		return "Client"
	case SpanKindProducer:
		return "Producer"
	case SpanKindConsumer:
		return "Consumer"
	}
	return ""
}
