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

import (
	"go.opentelemetry.io/collector/model/internal/pdata"
)

// TracesMarshaler is an alias for pdata.TracesMarshaler interface.
// Deprecated: [v0.49.0] Use traces.Marshaler instead.
type TracesMarshaler = pdata.TracesMarshaler

// TracesUnmarshaler is an alias for pdata.TracesUnmarshaler interface.
// Deprecated: [v0.49.0] Use traces.Unmarshaler instead.
type TracesUnmarshaler = pdata.TracesUnmarshaler

// TracesSizer is an alias for pdata.TracesSizer interface.
// Deprecated: [v0.49.0] Use traces.Sizer instead.
type TracesSizer = pdata.TracesSizer

// Traces is an alias for pdata.Traces struct.
// Deprecated: [v0.49.0] Use traces.Traces instead.
type Traces = pdata.Traces

// NewTraces is an alias for a function to create new Traces.
// Deprecated: [v0.49.0] Use traces.New instead.
var NewTraces = pdata.NewTraces

// TraceState is an alias for pdata.TraceState type.
// Deprecated: [v0.49.0] Use traces.TraceState instead.
type TraceState = pdata.TraceState

const (
	// Deprecated: [v0.49.0] Use traces.TraceStateEmpty instead.
	TraceStateEmpty = pdata.TraceStateEmpty
)

// SpanKind is an alias for pdata.SpanKind type.
// Deprecated: [v0.49.0] Use traces.SpanKind instead.
type SpanKind = pdata.SpanKind

const (

	// Deprecated: [v0.49.0] Use traces.SpanKindUnspecified instead.
	SpanKindUnspecified = pdata.SpanKindUnspecified

	// Deprecated: [v0.49.0] Use traces.SpanKindInternal instead.
	SpanKindInternal = pdata.SpanKindInternal

	// Deprecated: [v0.49.0] Use traces.SpanKindServer instead.
	SpanKindServer = pdata.SpanKindServer

	// Deprecated: [v0.49.0] Use traces.SpanKindClient instead.
	SpanKindClient = pdata.SpanKindClient

	// Deprecated: [v0.49.0] Use traces.SpanKindProducer instead.
	SpanKindProducer = pdata.SpanKindProducer

	// Deprecated: [v0.49.0] Use traces.SpanKindConsumer instead.
	SpanKindConsumer = pdata.SpanKindConsumer
)

// StatusCode is an alias for pdata.StatusCode type.
// Deprecated: [v0.49.0] Use traces.StatusCode instead.
type StatusCode = pdata.StatusCode

const (

	// Deprecated: [v0.49.0] Use traces.StatusCodeUnset instead.
	StatusCodeUnset = pdata.StatusCodeUnset

	// Deprecated: [v0.49.0] Use traces.StatusCodeOk instead.
	StatusCodeOk = pdata.StatusCodeOk

	// Deprecated: [v0.49.0] Use traces.StatusCodeError instead.
	StatusCodeError = pdata.StatusCodeError
)

// Deprecated: [v0.48.0] Use ScopeSpansSlice instead.
type InstrumentationLibrarySpansSlice = pdata.ScopeSpansSlice

// Deprecated: [v0.48.0] Use NewScopeSpansSlice instead.
var NewInstrumentationLibrarySpansSlice = pdata.NewScopeSpansSlice

// Deprecated: [v0.48.0] Use ScopeSpans instead.
type InstrumentationLibrarySpans = pdata.ScopeSpans

// Deprecated: [v0.48.0] Use NewScopeSpans instead.
var NewInstrumentationLibrarySpans = pdata.NewScopeSpans
