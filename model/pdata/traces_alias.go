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

package pdata // import "go.opentelemetry.io/collector/model/pdata"

// This file contains aliases for trace data structures.

import "go.opentelemetry.io/collector/model/internal"

// TracesMarshaler is an alias for internal.TracesMarshaler interface.
type TracesMarshaler = internal.TracesMarshaler

// TracesUnmarshaler is an alias for internal.TracesUnmarshaler interface.
type TracesUnmarshaler = internal.TracesUnmarshaler

// TracesSizer is an alias for internal.TracesSizer interface.
type TracesSizer = internal.TracesSizer

// Traces is an alias for internal.Traces struct.
type Traces = internal.Traces

// NewTraces is an alias for a function to create new Traces.
var NewTraces = internal.NewTraces

// TraceState is an alias for internal.TraceState type.
type TraceState = internal.TraceState

const (
	TraceStateEmpty = internal.TraceStateEmpty
)

// SpanKind is an alias for internal.SpanKind type.
type SpanKind = internal.SpanKind

const (
	SpanKindUnspecified = internal.SpanKindUnspecified
	SpanKindInternal    = internal.SpanKindInternal
	SpanKindServer      = internal.SpanKindServer
	SpanKindClient      = internal.SpanKindClient
	SpanKindProducer    = internal.SpanKindProducer
	SpanKindConsumer    = internal.SpanKindConsumer
)

// StatusCode is an alias for internal.StatusCode type.
type StatusCode = internal.StatusCode

const (
	StatusCodeUnset = internal.StatusCodeUnset
	StatusCodeOk    = internal.StatusCodeOk
	StatusCodeError = internal.StatusCodeError
)

// Deprecated: [v0.48.0] Use ScopeSpansSlice instead.
type InstrumentationLibrarySpansSlice = internal.ScopeSpansSlice

// Deprecated: [v0.48.0] Use NewScopeSpansSlice instead.
var NewInstrumentationLibrarySpansSlice = internal.NewScopeSpansSlice

// Deprecated: [v0.48.0] Use ScopeSpans instead.
type InstrumentationLibrarySpans = internal.ScopeSpans

// Deprecated: [v0.48.0] Use NewScopeSpans instead.
var NewInstrumentationLibrarySpans = internal.NewScopeSpans
