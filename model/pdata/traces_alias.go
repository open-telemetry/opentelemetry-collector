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

import "go.opentelemetry.io/collector/model/ptrace"

// TracesMarshaler is an alias for ptrace.TracesMarshaler interface.
type TracesMarshaler = ptrace.TracesMarshaler

// TracesUnmarshaler is an alias for ptrace.TracesUnmarshaler interface.
type TracesUnmarshaler = ptrace.TracesUnmarshaler

// TracesSizer is an alias for ptrace.TracesSizer interface.
type TracesSizer = ptrace.TracesSizer

// Traces is an alias for ptrace.Traces struct.
type Traces = ptrace.Traces

// NewTraces is an alias for a function to create new Traces.
var NewTraces = ptrace.NewTraces

// TracesFromInternalRep is an alias for ptrace.TracesFromInternalRep.
// TODO: Can be removed, internal ptrace.TracesFromInternalRep should be used instead.
var TracesFromInternalRep = ptrace.TracesFromInternalRep

// TraceState is an alias for ptrace.TraceState type.
type TraceState = ptrace.TraceState

const (
	TraceStateEmpty = ptrace.TraceStateEmpty
)

// SpanKind is an alias for ptrace.SpanKind type.
type SpanKind = ptrace.SpanKind

const (
	SpanKindUnspecified = ptrace.SpanKindUnspecified
	SpanKindInternal    = ptrace.SpanKindInternal
	SpanKindServer      = ptrace.SpanKindServer
	SpanKindClient      = ptrace.SpanKindClient
	SpanKindProducer    = ptrace.SpanKindProducer
	SpanKindConsumer    = ptrace.SpanKindConsumer
)

// StatusCode is an alias for ptrace.StatusCode type.
type StatusCode = ptrace.StatusCode

const (
	StatusCodeUnset = ptrace.StatusCodeUnset
	StatusCodeOk    = ptrace.StatusCodeOk
	StatusCodeError = ptrace.StatusCodeError
)
