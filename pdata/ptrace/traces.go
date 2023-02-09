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
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
)

// Traces is the top-level struct that is propagated through the traces pipeline.
// Use NewTraces to create new instance, zero-initialized instance is not valid for use.
type Traces internal.Traces

func newTraces(orig *otlpcollectortrace.ExportTraceServiceRequest) Traces {
	return Traces(internal.NewTraces(orig))
}

func (ms Traces) getOrig() *otlpcollectortrace.ExportTraceServiceRequest {
	return internal.GetOrigTraces(internal.Traces(ms))
}

// NewTraces creates a new Traces struct.
func NewTraces() Traces {
	return newTraces(&otlpcollectortrace.ExportTraceServiceRequest{})
}

// CopyTo copies the Traces instance overriding the destination.
func (ms Traces) CopyTo(dest Traces) {
	ms.ResourceSpans().CopyTo(dest.ResourceSpans())
}

// SpanCount calculates the total number of spans.
func (ms Traces) SpanCount() int {
	spanCount := 0
	rss := ms.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			spanCount += ilss.At(j).Spans().Len()
		}
	}
	return spanCount
}

// ResourceSpans returns the ResourceSpansSlice associated with this Metrics.
func (ms Traces) ResourceSpans() ResourceSpansSlice {
	return newResourceSpansSlice(&ms.getOrig().ResourceSpans)
}
