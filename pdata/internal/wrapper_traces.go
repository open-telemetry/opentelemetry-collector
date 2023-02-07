// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

type Traces struct {
	st *stateTraces
}

type stateTraces struct {
	orig  *otlpcollectortrace.ExportTraceServiceRequest
	state State
}

func GetTracesOrig(ms Traces) *otlpcollectortrace.ExportTraceServiceRequest {
	return ms.st.orig
}

func GetTracesState(ms Traces) State {
	return ms.st.state
}

// ResetStateTraces replaces the internal StateTraces with a new empty and the provided state.
func ResetStateTraces(ms Traces, s State) {
	ms.st.orig = &otlpcollectortrace.ExportTraceServiceRequest{}
	ms.st.state = s
}

func NewTraces(orig *otlpcollectortrace.ExportTraceServiceRequest, s State) Traces {
	return Traces{&stateTraces{orig: orig, state: s}}
}

// TracesToProto internal helper to convert Traces to protobuf representation.
func TracesToProto(l Traces) otlptrace.TracesData {
	return otlptrace.TracesData{
		ResourceSpans: l.st.orig.ResourceSpans,
	}
}

// TracesFromProto internal helper to convert protobuf representation to Traces.
func TracesFromProto(orig otlptrace.TracesData) Traces {
	return Traces{&stateTraces{
		orig: &otlpcollectortrace.ExportTraceServiceRequest{
			ResourceSpans: orig.ResourceSpans,
		},
		state: StateExclusive,
	}}
}
