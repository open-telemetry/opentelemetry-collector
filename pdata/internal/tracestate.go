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
type TraceState struct {
	commonTraceState
}

type MutableTraceState struct {
	commonTraceState
	preventConversion struct{} // nolint:unused
}

type commonTraceState struct {
	orig *string
}

func (ms MutableTraceState) AsImmutable() TraceState {
	return TraceState{commonTraceState{orig: ms.orig}}
}

func NewTraceStateFromOrig(orig *string) TraceState {
	return TraceState{commonTraceState{orig}}
}

func NewMutableTraceStateFromOrig(orig *string) MutableTraceState {
	return MutableTraceState{commonTraceState: commonTraceState{orig}}
}

func NewMutableTraceState() MutableTraceState {
	return NewMutableTraceStateFromOrig(new(string))
}

// AsRaw returns the string representation of the tracestate in w3c-trace-context format: https://www.w3.org/TR/trace-context/#tracestate-header
func (ms commonTraceState) AsRaw() string {
	return *ms.orig
}

// FromRaw copies the string representation in w3c-trace-context format of the tracestate into this TraceState.
func (ms MutableTraceState) FromRaw(v string) {
	*ms.orig = v
}

// MoveTo moves the TraceState instance overriding the destination
// and resetting the current instance to its zero value.
func (ms MutableTraceState) MoveTo(dest MutableTraceState) {
	*dest.orig = *ms.orig
	*ms.orig = ""
}

// CopyTo copies the TraceState instance overriding the destination.
func (ms commonTraceState) CopyTo(dest MutableTraceState) {
	*dest.orig = *ms.orig
}

func GenerateTestTraceState() MutableTraceState {
	var orig string
	ms := NewMutableTraceStateFromOrig(&orig)
	FillTestTraceState(ms)
	return ms
}

func FillTestTraceState(dest MutableTraceState) {
	*dest.orig = "rojo=00f067aa0ba902b7"
}
