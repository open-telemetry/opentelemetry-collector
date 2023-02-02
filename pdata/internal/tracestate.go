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

package internal // import "go.opentelemetry.io/collector/pdata/internal"

// TraceState represents the trace state from the w3c-trace-context.
type TraceState interface {
	getOrig() *string
	AsRaw() string
	CopyTo(dest MutableTraceState)
}

type MutableTraceState interface {
	TraceState
	FromRaw(v string)
	MoveTo(dest MutableTraceState)
}

func NewTraceState() TraceState {
	return internalTraceState{new(string)}
}

func NewImmutableTraceState(orig *string) TraceState {
	return internalTraceState{orig}
}

func NewMutableTraceState(orig *string) MutableTraceState {
	return internalTraceState{orig}
}

type internalTraceState struct {
	orig *string
}

func (ms internalTraceState) getOrig() *string {
	return ms.orig
}

// AsRaw returns the string representation of the tracestate in w3c-trace-context format: https://www.w3.org/TR/trace-context/#tracestate-header
func (ms internalTraceState) AsRaw() string {
	return *ms.getOrig()
}

// FromRaw copies the string representation in w3c-trace-context format of the tracestate into this TraceState.
func (ms internalTraceState) FromRaw(v string) {
	*ms.getOrig() = v
}

// MoveTo moves the TraceState instance overriding the destination
// and resetting the current instance to its zero value.
func (ms internalTraceState) MoveTo(dest MutableTraceState) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = ""
}

// CopyTo copies the TraceState instance overriding the destination.
func (ms internalTraceState) CopyTo(dest MutableTraceState) {
	*dest.getOrig() = *ms.getOrig()
}
