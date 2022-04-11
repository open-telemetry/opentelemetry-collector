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

import "go.opentelemetry.io/collector/pdata/internal"

// This file contains aliases for trace data structures.

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
