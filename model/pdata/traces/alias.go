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

package traces // import "go.opentelemetry.io/collector/model/pdata/traces"

// This file contains aliases for traces data structures.

import "go.opentelemetry.io/collector/model/internal/pdata"

// Marshaler is an alias for pdata.TracesMarshaler interface.
type Marshaler = pdata.TracesMarshaler

// Unmarshaler is an alias for pdata.TracesUnmarshaler interface.
type Unmarshaler = pdata.TracesUnmarshaler

// Sizer is an alias for pdata.TracesSizer interface.
type Sizer = pdata.TracesSizer

// Traces is an alias for pdata.Traces struct.
type Traces = pdata.Traces

// New is an alias for a function to create new Traces.
var New = pdata.NewTraces

// TraceState is an alias for pdata.TraceState type.
type TraceState = pdata.TraceState

const (
	TraceStateEmpty = pdata.TraceStateEmpty
)

// SpanKind is an alias for pdata.SpanKind type.
type SpanKind = pdata.SpanKind

const (
	SpanKindUnspecified = pdata.SpanKindUnspecified
	SpanKindInternal    = pdata.SpanKindInternal
	SpanKindServer      = pdata.SpanKindServer
	SpanKindClient      = pdata.SpanKindClient
	SpanKindProducer    = pdata.SpanKindProducer
	SpanKindConsumer    = pdata.SpanKindConsumer
)

// StatusCode is an alias for pdata.StatusCode type.
type StatusCode = pdata.StatusCode

const (
	StatusCodeUnset = pdata.StatusCodeUnset
	StatusCodeOk    = pdata.StatusCodeOk
	StatusCodeError = pdata.StatusCodeError
)
