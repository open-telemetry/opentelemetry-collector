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
	*pTraces
}

type pTraces struct {
	orig  *otlpcollectortrace.ExportTraceServiceRequest
	state State
}

func NewTracesFromResourceSpansOrig(orig *[]*otlptrace.ResourceSpans) Traces {
	return Traces{&pTraces{orig: &otlpcollectortrace.ExportTraceServiceRequest{ResourceSpans: *orig}, state: StateExclusive}}
}

func (ms Traces) AsShared() Traces {
	ms.state = StateShared
	return Traces{&pTraces{orig: ms.orig, state: StateShared}}
}

func (ms Traces) IsShared() bool {
	return ms.pTraces != nil && ms.state == StateShared
}

func (ms Traces) MarkExclusive() {
	ms.state = StateExclusive
}

func (ms Traces) GetOrig() *otlpcollectortrace.ExportTraceServiceRequest {
	return ms.orig
}
func (ms Traces) SetOrig(orig *otlpcollectortrace.ExportTraceServiceRequest) {
	ms.orig = orig
}

func NewTraces(orig *otlpcollectortrace.ExportTraceServiceRequest) Traces {
	return Traces{&pTraces{orig: orig, state: StateExclusive}}
}

// TracesToProto internal helper to convert Traces to protobuf representation.
func TracesToProto(l Traces) otlptrace.TracesData {
	return otlptrace.TracesData{
		ResourceSpans: l.orig.ResourceSpans,
	}
}

// TracesFromProto internal helper to convert protobuf representation to Traces.
func TracesFromProto(orig otlptrace.TracesData) Traces {
	return Traces{&pTraces{
		orig: &otlpcollectortrace.ExportTraceServiceRequest{
			ResourceSpans: orig.ResourceSpans,
		},
		state: StateExclusive,
	}}
}
