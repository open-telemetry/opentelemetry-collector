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
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Traces is the top-level struct that is propagated through the traces pipeline.
// Use NewTraces to create new instance, zero-initialized instance is not valid for use.
type Traces internal.Traces

func (ms Traces) getOrig() *otlpcollectortrace.ExportTraceServiceRequest {
	return internal.GetOrigTraces(internal.Traces(ms))
}

// NewTraces creates a new Traces struct.
func NewTraces() Traces {
	orig := &otlpcollectortrace.ExportTraceServiceRequest{}
	return Traces(internal.NewTraces(orig, internal.StateExclusive))
}

// MoveTo moves the Traces instance overriding the destination and
// resetting the current instance to its zero value.
// Deprecated: [1.0.0-rc5] The method can be replaced with a plain assignment.
func (ms Traces) MoveTo(dest Traces) {
	*dest.getOrig() = *ms.getOrig()
	*ms.getOrig() = otlpcollectortrace.ExportTraceServiceRequest{}
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
	return newImmutableResourceSpansSlice(&ms.getOrig().ResourceSpans)
}

// MutableResourceSpans returns the MutableResourceSpansSlice associated with this Spans object.
// This method should be called at once per ConsumeSpans call if the slice has to be changed,
// otherwise use ResourceSpans method.
func (ms Traces) MutableResourceSpans() MutableResourceSpansSlice {
	if internal.GetTracesState(internal.Traces(ms)) == pcommon.StateShared {
		rms := NewResourceSpansSlice()
		ms.ResourceSpans().CopyTo(rms)
		ms.getOrig().ResourceSpans = *rms.getOrig()
		internal.SetTracesState(internal.Traces(ms), pcommon.StateExclusive)
		return rms
	}
	return newMutableResourceSpansSlice(&ms.getOrig().ResourceSpans)
}
