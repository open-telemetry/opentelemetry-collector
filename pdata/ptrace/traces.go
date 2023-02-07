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
	return Traces(internal.NewTraces(orig, internal.StateExclusive))
}

func (ms Traces) getOrig() *otlpcollectortrace.ExportTraceServiceRequest {
	return internal.GetTracesOrig(internal.Traces(ms))
}

// NewTraces creates a new Traces struct.
func NewTraces() Traces {
	orig := &otlpcollectortrace.ExportTraceServiceRequest{}
	return Traces(internal.NewTraces(orig, internal.StateExclusive))
}

// AsShared returns the same Traces instance that is marked as shared.
// It guarantees that the underlying data structure will not be modified in the downstream processing.
func (ms Traces) AsShared() Traces {
	return Traces(internal.NewTraces(ms.getOrig(), internal.StateShared))
}

// CopyTo copies the Traces instance overriding the destination.
func (ms Traces) CopyTo(dest Traces) {
	ms.ResourceSpans().CopyTo(dest.MutableResourceSpans())
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
	return newResourceSpansSliceFromOrig(&ms.getOrig().ResourceSpans)
}

// MutableResourceSpans returns the MutableResourceSpansSlice associated with this Spans object.
// This method should be called at once per ConsumeSpans call if the slice has to be changed,
// otherwise use ResourceSpans method.
func (ms Traces) MutableResourceSpans() MutableResourceSpansSlice {
	if internal.GetTracesState(internal.Traces(ms)) == internal.StateShared {
		rms := NewMutableResourceSpansSlice()
		ms.ResourceSpans().CopyTo(rms)
		internal.ResetStateTraces(internal.Traces(ms), internal.StateExclusive)
		ms.getOrig().ResourceSpans = *rms.orig
		return rms
	}
	return newMutableResourceSpansSliceFromOrig(&ms.getOrig().ResourceSpans)
}
