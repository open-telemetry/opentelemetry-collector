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

package ptrace

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

func TestSpanCount(t *testing.T) {
	traces := NewTraces()
	assert.EqualValues(t, 0, traces.SpanCount())

	rs := traces.ResourceSpans().AppendEmpty()
	assert.EqualValues(t, 0, traces.SpanCount())

	ils := rs.ScopeSpans().AppendEmpty()
	assert.EqualValues(t, 0, traces.SpanCount())

	ils.Spans().AppendEmpty()
	assert.EqualValues(t, 1, traces.SpanCount())

	rms := traces.ResourceSpans()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeSpans().AppendEmpty()
	ilss := rms.AppendEmpty().ScopeSpans().AppendEmpty().Spans()
	for i := 0; i < 5; i++ {
		ilss.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.EqualValues(t, 6, traces.SpanCount())
}

func TestSpanCountWithEmpty(t *testing.T) {
	assert.EqualValues(t, 0, newTraces(&otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{{}},
	}).SpanCount())
	assert.EqualValues(t, 0, newTraces(&otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				ScopeSpans: []*otlptrace.ScopeSpans{{}},
			},
		},
	}).SpanCount())
	assert.EqualValues(t, 1, newTraces(&otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				ScopeSpans: []*otlptrace.ScopeSpans{
					{
						Spans: []*otlptrace.Span{{}},
					},
				},
			},
		},
	}).SpanCount())
}

func TestToFromOtlp(t *testing.T) {
	otlp := &otlpcollectortrace.ExportTraceServiceRequest{}
	traces := newTraces(otlp)
	assert.EqualValues(t, NewTraces(), traces)
	assert.EqualValues(t, otlp, traces.getOrig())
	// More tests in ./tracedata/traces_test.go. Cannot have them here because of
	// circular dependency.
}

func TestResourceSpansWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceSpans as pdata struct.
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(traces.getOrig())
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage emptypb.Empty
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRS2 otlpcollectortrace.ExportTraceServiceRequest
	err = gogoproto.Unmarshal(wire2, &gogoprotoRS2)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.EqualValues(t, traces.getOrig(), &gogoprotoRS2)
}

func TestTracesCopyTo(t *testing.T) {
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())
	tracesCopy := NewTraces()
	traces.CopyTo(tracesCopy)
	assert.EqualValues(t, traces, tracesCopy)
}
