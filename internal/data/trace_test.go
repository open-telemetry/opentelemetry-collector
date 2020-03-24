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
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanCount(t *testing.T) {
	td := TraceData{}
	assert.EqualValues(t, 0, td.SpanCount())

	td = TraceData{
		resourceSpans: []*ResourceSpans{
			NewResourceSpans(NewResource(), []*InstrumentationLibrarySpans{}),
		},
	}
	assert.EqualValues(t, 0, td.SpanCount())

	td = TraceData{
		resourceSpans: []*ResourceSpans{
			NewResourceSpans(NewResource(), []*InstrumentationLibrarySpans{
				NewInstrumentationLibrarySpans(NewInstrumentationLibrary(), []Span{})}),
		},
	}
	assert.EqualValues(t, 0, td.SpanCount())

	td = TraceData{
		resourceSpans: []*ResourceSpans{
			NewResourceSpans(NewResource(), []*InstrumentationLibrarySpans{
				NewInstrumentationLibrarySpans(NewInstrumentationLibrary(), NewSpanSlice(1))}),
		},
	}
	assert.EqualValues(t, 1, td.SpanCount())

	td = TraceData{
		resourceSpans: []*ResourceSpans{
			NewResourceSpans(NewResource(), []*InstrumentationLibrarySpans{
				NewInstrumentationLibrarySpans(NewInstrumentationLibrary(), NewSpanSlice(1))}),
			NewResourceSpans(NewResource(), []*InstrumentationLibrarySpans{
				NewInstrumentationLibrarySpans(NewInstrumentationLibrary(), []Span{})}),
			NewResourceSpans(NewResource(), []*InstrumentationLibrarySpans{
				NewInstrumentationLibrarySpans(NewInstrumentationLibrary(), NewSpanSlice(5))}),
		},
	}
	assert.EqualValues(t, 6, td.SpanCount())
}

func TestNewSpanSlice(t *testing.T) {
	spans := NewSpanSlice(0)
	assert.EqualValues(t, 0, len(spans))

	n := rand.Intn(10)
	spans = NewSpanSlice(n)
	assert.EqualValues(t, n, len(spans))
	for span := range spans {
		assert.NotNil(t, span)
	}
}

func TestAttrs(t *testing.T) {
	attrs := NewAttributeMap(AttributesMap{"attr1": NewAttributeValueString("abc")})

	span := NewSpan()
	assert.EqualValues(t, 0, span.DroppedAttributesCount())
	span.SetAttributes(attrs)
	assert.EqualValues(t, attrs, span.Attributes())
	span.SetDroppedAttributesCount(123)
	assert.EqualValues(t, 123, span.DroppedAttributesCount())

	event := NewSpanEvent()
	event.SetAttributes(attrs)
	assert.EqualValues(t, attrs, event.Attributes())
	event.SetDroppedAttributesCount(234)
	assert.EqualValues(t, 234, event.DroppedAttributesCount())

	link := NewSpanLink()
	assert.EqualValues(t, 0, link.DroppedAttributesCount())
	link.SetAttributes(attrs)
	assert.EqualValues(t, attrs, link.Attributes())
	link.SetDroppedAttributesCount(456)
	assert.EqualValues(t, 456, link.DroppedAttributesCount())
}
