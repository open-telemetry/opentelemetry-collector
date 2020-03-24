// Copyright 2020 OpenTelemetry Authors
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

package data

import (
	"testing"

	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
)

func TestSpanCount(t *testing.T) {
	md := NewTraceData()
	assert.EqualValues(t, 0, md.SpanCount())

	md.SetResourceSpans(NewResourceSpansSlice(1))
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().Get(0).SetInstrumentationLibrarySpans(NewInstrumentationLibrarySpansSlice(1))
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().Get(0).InstrumentationLibrarySpans().Get(0).SetSpans(NewSpanSlice(1))
	assert.EqualValues(t, 1, md.SpanCount())

	rms := NewResourceSpansSlice(3)
	rms.Get(0).SetInstrumentationLibrarySpans(NewInstrumentationLibrarySpansSlice(1))
	rms.Get(0).InstrumentationLibrarySpans().Get(0).SetSpans(NewSpanSlice(1))
	rms.Get(1).SetInstrumentationLibrarySpans(NewInstrumentationLibrarySpansSlice(1))
	rms.Get(2).SetInstrumentationLibrarySpans(NewInstrumentationLibrarySpansSlice(1))
	rms.Get(2).InstrumentationLibrarySpans().Get(0).SetSpans(NewSpanSlice(5))
	md.SetResourceSpans(rms)
	assert.EqualValues(t, 6, md.SpanCount())
}

func TestTraceID(t *testing.T) {
	tid := NewTraceID(nil)
	assert.EqualValues(t, []byte(nil), tid.Bytes())

	tid = NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	assert.EqualValues(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, tid.Bytes())
}

func TestAttrs(t *testing.T) {
	attrs := NewAttributeMap(AttributesMap{"attr1": NewAttributeValueString("abc")})

	span := NewEmptySpan()
	assert.EqualValues(t, 0, span.DroppedAttributesCount())
	span.SetAttributes(attrs)
	assert.EqualValues(t, attrs, span.Attributes())
	span.SetDroppedAttributesCount(123)
	assert.EqualValues(t, 123, span.DroppedAttributesCount())

	event := NewEmptySpanEvent()
	event.SetAttributes(attrs)
	assert.EqualValues(t, attrs, event.Attributes())
	event.SetDroppedAttributesCount(234)
	assert.EqualValues(t, 234, event.DroppedAttributesCount())

	link := NewEmptySpanLink()
	assert.EqualValues(t, 0, link.DroppedAttributesCount())
	link.SetAttributes(attrs)
	assert.EqualValues(t, attrs, link.Attributes())
	link.SetDroppedAttributesCount(456)
	assert.EqualValues(t, 456, link.DroppedAttributesCount())
}

func TestToFromOtlp(t *testing.T) {
	otlp := []*otlptrace.ResourceSpans(nil)
	td := TraceDataFromOtlp(otlp)
	assert.EqualValues(t, NewTraceData(), td)
	assert.EqualValues(t, otlp, TraceDataToOtlp(td))
	// More tests in ./tracedata/trace_test.go. Cannot have them here because of
	// circular dependency.
}
