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

// Code generated by "internal/data_generator/main.go". DO NOT EDIT.
// To regenerate this file run "go run internal/data_generator/main.go".

package data

import (
	"testing"
	
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
)

func TestResourceSpansSlice(t *testing.T) {
	es := NewResourceSpansSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newResourceSpansSlice(&[]*otlptrace.ResourceSpans{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(13)
	emptyVal :=  NewResourceSpans()
	emptyVal.InitEmpty()
	testVal := generateTestResourceSpans()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestResourceSpans(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}

	// Test Resize less elements.
	const resizeSmallLen = 10
	expectedEs := make(map[*otlptrace.ResourceSpans]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlptrace.ResourceSpans]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 13
	oldLen := es.Len()
	expectedEs = make(map[*otlptrace.ResourceSpans]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlptrace.ResourceSpans]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewResourceSpansSlice(), es)
}

func TestResourceSpans(t *testing.T) {
	ms := NewResourceSpans()
	assert.EqualValues(t, true,ms.IsNil())
	ms.InitEmpty()
	assert.EqualValues(t, false, ms.IsNil())

	assert.EqualValues(t, true, ms.Resource().IsNil())
	ms.Resource().InitEmpty()
	assert.EqualValues(t, false, ms.Resource().IsNil())
	fillTestResource(ms.Resource())
	assert.EqualValues(t, generateTestResource(), ms.Resource())

	assert.EqualValues(t, NewInstrumentationLibrarySpansSlice(), ms.InstrumentationLibrarySpans())
	fillTestInstrumentationLibrarySpansSlice(ms.InstrumentationLibrarySpans())
	testValInstrumentationLibrarySpans := generateTestInstrumentationLibrarySpansSlice()
	assert.EqualValues(t, testValInstrumentationLibrarySpans, ms.InstrumentationLibrarySpans())

	assert.EqualValues(t, generateTestResourceSpans(), ms)
}

func TestInstrumentationLibrarySpansSlice(t *testing.T) {
	es := NewInstrumentationLibrarySpansSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newInstrumentationLibrarySpansSlice(&[]*otlptrace.InstrumentationLibrarySpans{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(13)
	emptyVal :=  NewInstrumentationLibrarySpans()
	emptyVal.InitEmpty()
	testVal := generateTestInstrumentationLibrarySpans()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestInstrumentationLibrarySpans(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}

	// Test Resize less elements.
	const resizeSmallLen = 10
	expectedEs := make(map[*otlptrace.InstrumentationLibrarySpans]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlptrace.InstrumentationLibrarySpans]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 13
	oldLen := es.Len()
	expectedEs = make(map[*otlptrace.InstrumentationLibrarySpans]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlptrace.InstrumentationLibrarySpans]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewInstrumentationLibrarySpansSlice(), es)
}

func TestInstrumentationLibrarySpans(t *testing.T) {
	ms := NewInstrumentationLibrarySpans()
	assert.EqualValues(t, true,ms.IsNil())
	ms.InitEmpty()
	assert.EqualValues(t, false, ms.IsNil())

	assert.EqualValues(t, true, ms.InstrumentationLibrary().IsNil())
	ms.InstrumentationLibrary().InitEmpty()
	assert.EqualValues(t, false, ms.InstrumentationLibrary().IsNil())
	fillTestInstrumentationLibrary(ms.InstrumentationLibrary())
	assert.EqualValues(t, generateTestInstrumentationLibrary(), ms.InstrumentationLibrary())

	assert.EqualValues(t, NewSpanSlice(), ms.Spans())
	fillTestSpanSlice(ms.Spans())
	testValSpans := generateTestSpanSlice()
	assert.EqualValues(t, testValSpans, ms.Spans())

	assert.EqualValues(t, generateTestInstrumentationLibrarySpans(), ms)
}

func TestSpanSlice(t *testing.T) {
	es := NewSpanSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newSpanSlice(&[]*otlptrace.Span{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(13)
	emptyVal :=  NewSpan()
	emptyVal.InitEmpty()
	testVal := generateTestSpan()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestSpan(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}

	// Test Resize less elements.
	const resizeSmallLen = 10
	expectedEs := make(map[*otlptrace.Span]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlptrace.Span]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 13
	oldLen := es.Len()
	expectedEs = make(map[*otlptrace.Span]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlptrace.Span]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewSpanSlice(), es)
}

func TestSpan(t *testing.T) {
	ms := NewSpan()
	assert.EqualValues(t, true,ms.IsNil())
	ms.InitEmpty()
	assert.EqualValues(t, false, ms.IsNil())

	assert.EqualValues(t, NewTraceID(nil), ms.TraceID())
	testValTraceID := NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	ms.SetTraceID(testValTraceID)
	assert.EqualValues(t, testValTraceID, ms.TraceID())

	assert.EqualValues(t, NewSpanID(nil), ms.SpanID())
	testValSpanID := NewSpanID([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	ms.SetSpanID(testValSpanID)
	assert.EqualValues(t, testValSpanID, ms.SpanID())

	assert.EqualValues(t, TraceState(""), ms.TraceState())
	testValTraceState := TraceState("congo=congos")
	ms.SetTraceState(testValTraceState)
	assert.EqualValues(t, testValTraceState, ms.TraceState())

	assert.EqualValues(t, NewSpanID(nil), ms.ParentSpanID())
	testValParentSpanID := NewSpanID([]byte{8, 7, 6, 5, 4, 3, 2, 1})
	ms.SetParentSpanID(testValParentSpanID)
	assert.EqualValues(t, testValParentSpanID, ms.ParentSpanID())

	assert.EqualValues(t, "", ms.Name())
	testValName := "test_name"
	ms.SetName(testValName)
	assert.EqualValues(t, testValName, ms.Name())

	assert.EqualValues(t, SpanKindUNSPECIFIED, ms.Kind())
	testValKind := SpanKindSERVER
	ms.SetKind(testValKind)
	assert.EqualValues(t, testValKind, ms.Kind())

	assert.EqualValues(t, TimestampUnixNano(0), ms.StartTime())
	testValStartTime := TimestampUnixNano(1234567890)
	ms.SetStartTime(testValStartTime)
	assert.EqualValues(t, testValStartTime, ms.StartTime())

	assert.EqualValues(t, TimestampUnixNano(0), ms.EndTime())
	testValEndTime := TimestampUnixNano(1234567890)
	ms.SetEndTime(testValEndTime)
	assert.EqualValues(t, testValEndTime, ms.EndTime())

	assert.EqualValues(t, NewAttributeMap(nil), ms.Attributes())
	testValAttributes := generateTestAttributeMap()
	ms.SetAttributes(testValAttributes)
	assert.EqualValues(t, testValAttributes, ms.Attributes())

	assert.EqualValues(t, uint32(0), ms.DroppedAttributesCount())
	testValDroppedAttributesCount := uint32(17)
	ms.SetDroppedAttributesCount(testValDroppedAttributesCount)
	assert.EqualValues(t, testValDroppedAttributesCount, ms.DroppedAttributesCount())

	assert.EqualValues(t, NewSpanEventSlice(), ms.Events())
	fillTestSpanEventSlice(ms.Events())
	testValEvents := generateTestSpanEventSlice()
	assert.EqualValues(t, testValEvents, ms.Events())

	assert.EqualValues(t, uint32(0), ms.DroppedEventsCount())
	testValDroppedEventsCount := uint32(17)
	ms.SetDroppedEventsCount(testValDroppedEventsCount)
	assert.EqualValues(t, testValDroppedEventsCount, ms.DroppedEventsCount())

	assert.EqualValues(t, NewSpanLinkSlice(), ms.Links())
	fillTestSpanLinkSlice(ms.Links())
	testValLinks := generateTestSpanLinkSlice()
	assert.EqualValues(t, testValLinks, ms.Links())

	assert.EqualValues(t, uint32(0), ms.DroppedLinksCount())
	testValDroppedLinksCount := uint32(17)
	ms.SetDroppedLinksCount(testValDroppedLinksCount)
	assert.EqualValues(t, testValDroppedLinksCount, ms.DroppedLinksCount())

	assert.EqualValues(t, true, ms.Status().IsNil())
	ms.Status().InitEmpty()
	assert.EqualValues(t, false, ms.Status().IsNil())
	fillTestSpanStatus(ms.Status())
	assert.EqualValues(t, generateTestSpanStatus(), ms.Status())

	assert.EqualValues(t, generateTestSpan(), ms)
}

func TestSpanEventSlice(t *testing.T) {
	es := NewSpanEventSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newSpanEventSlice(&[]*otlptrace.Span_Event{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(13)
	emptyVal :=  NewSpanEvent()
	emptyVal.InitEmpty()
	testVal := generateTestSpanEvent()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestSpanEvent(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}

	// Test Resize less elements.
	const resizeSmallLen = 10
	expectedEs := make(map[*otlptrace.Span_Event]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlptrace.Span_Event]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 13
	oldLen := es.Len()
	expectedEs = make(map[*otlptrace.Span_Event]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlptrace.Span_Event]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewSpanEventSlice(), es)
}

func TestSpanEvent(t *testing.T) {
	ms := NewSpanEvent()
	assert.EqualValues(t, true,ms.IsNil())
	ms.InitEmpty()
	assert.EqualValues(t, false, ms.IsNil())

	assert.EqualValues(t, TimestampUnixNano(0), ms.Timestamp())
	testValTimestamp := TimestampUnixNano(1234567890)
	ms.SetTimestamp(testValTimestamp)
	assert.EqualValues(t, testValTimestamp, ms.Timestamp())

	assert.EqualValues(t, "", ms.Name())
	testValName := "test_name"
	ms.SetName(testValName)
	assert.EqualValues(t, testValName, ms.Name())

	assert.EqualValues(t, NewAttributeMap(nil), ms.Attributes())
	testValAttributes := generateTestAttributeMap()
	ms.SetAttributes(testValAttributes)
	assert.EqualValues(t, testValAttributes, ms.Attributes())

	assert.EqualValues(t, uint32(0), ms.DroppedAttributesCount())
	testValDroppedAttributesCount := uint32(17)
	ms.SetDroppedAttributesCount(testValDroppedAttributesCount)
	assert.EqualValues(t, testValDroppedAttributesCount, ms.DroppedAttributesCount())

	assert.EqualValues(t, generateTestSpanEvent(), ms)
}

func TestSpanLinkSlice(t *testing.T) {
	es := NewSpanLinkSlice()
	assert.EqualValues(t, 0, es.Len())
	es = newSpanLinkSlice(&[]*otlptrace.Span_Link{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(13)
	emptyVal :=  NewSpanLink()
	emptyVal.InitEmpty()
	testVal := generateTestSpanLink()
	assert.EqualValues(t, 13, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTestSpanLink(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}

	// Test Resize less elements.
	const resizeSmallLen = 10
	expectedEs := make(map[*otlptrace.Span_Link]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*otlptrace.Span_Link]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 13
	oldLen := es.Len()
	expectedEs = make(map[*otlptrace.Span_Link]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*otlptrace.Span_Link]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, NewSpanLinkSlice(), es)
}

func TestSpanLink(t *testing.T) {
	ms := NewSpanLink()
	assert.EqualValues(t, true,ms.IsNil())
	ms.InitEmpty()
	assert.EqualValues(t, false, ms.IsNil())

	assert.EqualValues(t, NewTraceID(nil), ms.TraceID())
	testValTraceID := NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	ms.SetTraceID(testValTraceID)
	assert.EqualValues(t, testValTraceID, ms.TraceID())

	assert.EqualValues(t, NewSpanID(nil), ms.SpanID())
	testValSpanID := NewSpanID([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	ms.SetSpanID(testValSpanID)
	assert.EqualValues(t, testValSpanID, ms.SpanID())

	assert.EqualValues(t, TraceState(""), ms.TraceState())
	testValTraceState := TraceState("congo=congos")
	ms.SetTraceState(testValTraceState)
	assert.EqualValues(t, testValTraceState, ms.TraceState())

	assert.EqualValues(t, NewAttributeMap(nil), ms.Attributes())
	testValAttributes := generateTestAttributeMap()
	ms.SetAttributes(testValAttributes)
	assert.EqualValues(t, testValAttributes, ms.Attributes())

	assert.EqualValues(t, uint32(0), ms.DroppedAttributesCount())
	testValDroppedAttributesCount := uint32(17)
	ms.SetDroppedAttributesCount(testValDroppedAttributesCount)
	assert.EqualValues(t, testValDroppedAttributesCount, ms.DroppedAttributesCount())

	assert.EqualValues(t, generateTestSpanLink(), ms)
}

func TestSpanStatus(t *testing.T) {
	ms := NewSpanStatus()
	assert.EqualValues(t, true,ms.IsNil())
	ms.InitEmpty()
	assert.EqualValues(t, false, ms.IsNil())

	assert.EqualValues(t, StatusCode(0), ms.Code())
	testValCode := StatusCode(1)
	ms.SetCode(testValCode)
	assert.EqualValues(t, testValCode, ms.Code())

	assert.EqualValues(t, "", ms.Message())
	testValMessage := "cancelled"
	ms.SetMessage(testValMessage)
	assert.EqualValues(t, testValMessage, ms.Message())

	assert.EqualValues(t, generateTestSpanStatus(), ms)
}

func generateTestResourceSpansSlice() ResourceSpansSlice {
	tv := NewResourceSpansSlice()
	fillTestResourceSpansSlice(tv)
	return tv
}

func fillTestResourceSpansSlice(tv ResourceSpansSlice) {
	tv.Resize(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestResourceSpans(tv.At(i))
	}
}

func generateTestResourceSpans() ResourceSpans {
	tv := NewResourceSpans()
	tv.InitEmpty()
	fillTestResourceSpans(tv)
	return tv
}

func fillTestResourceSpans(tv ResourceSpans) {
	tv.Resource().InitEmpty()
	fillTestResource(tv.Resource())
	fillTestInstrumentationLibrarySpansSlice(tv.InstrumentationLibrarySpans())
}

func generateTestInstrumentationLibrarySpansSlice() InstrumentationLibrarySpansSlice {
	tv := NewInstrumentationLibrarySpansSlice()
	fillTestInstrumentationLibrarySpansSlice(tv)
	return tv
}

func fillTestInstrumentationLibrarySpansSlice(tv InstrumentationLibrarySpansSlice) {
	tv.Resize(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestInstrumentationLibrarySpans(tv.At(i))
	}
}

func generateTestInstrumentationLibrarySpans() InstrumentationLibrarySpans {
	tv := NewInstrumentationLibrarySpans()
	tv.InitEmpty()
	fillTestInstrumentationLibrarySpans(tv)
	return tv
}

func fillTestInstrumentationLibrarySpans(tv InstrumentationLibrarySpans) {
	tv.InstrumentationLibrary().InitEmpty()
	fillTestInstrumentationLibrary(tv.InstrumentationLibrary())
	fillTestSpanSlice(tv.Spans())
}

func generateTestSpanSlice() SpanSlice {
	tv := NewSpanSlice()
	fillTestSpanSlice(tv)
	return tv
}

func fillTestSpanSlice(tv SpanSlice) {
	tv.Resize(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestSpan(tv.At(i))
	}
}

func generateTestSpan() Span {
	tv := NewSpan()
	tv.InitEmpty()
	fillTestSpan(tv)
	return tv
}

func fillTestSpan(tv Span) {
	tv.SetTraceID(NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
	tv.SetSpanID(NewSpanID([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	tv.SetTraceState(TraceState("congo=congos"))
	tv.SetParentSpanID(NewSpanID([]byte{8, 7, 6, 5, 4, 3, 2, 1}))
	tv.SetName("test_name")
	tv.SetKind(SpanKindSERVER)
	tv.SetStartTime(TimestampUnixNano(1234567890))
	tv.SetEndTime(TimestampUnixNano(1234567890))
	tv.SetAttributes(generateTestAttributeMap())
	tv.SetDroppedAttributesCount(uint32(17))
	fillTestSpanEventSlice(tv.Events())
	tv.SetDroppedEventsCount(uint32(17))
	fillTestSpanLinkSlice(tv.Links())
	tv.SetDroppedLinksCount(uint32(17))
	tv.Status().InitEmpty()
	fillTestSpanStatus(tv.Status())
}

func generateTestSpanEventSlice() SpanEventSlice {
	tv := NewSpanEventSlice()
	fillTestSpanEventSlice(tv)
	return tv
}

func fillTestSpanEventSlice(tv SpanEventSlice) {
	tv.Resize(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestSpanEvent(tv.At(i))
	}
}

func generateTestSpanEvent() SpanEvent {
	tv := NewSpanEvent()
	tv.InitEmpty()
	fillTestSpanEvent(tv)
	return tv
}

func fillTestSpanEvent(tv SpanEvent) {
	tv.SetTimestamp(TimestampUnixNano(1234567890))
	tv.SetName("test_name")
	tv.SetAttributes(generateTestAttributeMap())
	tv.SetDroppedAttributesCount(uint32(17))
}

func generateTestSpanLinkSlice() SpanLinkSlice {
	tv := NewSpanLinkSlice()
	fillTestSpanLinkSlice(tv)
	return tv
}

func fillTestSpanLinkSlice(tv SpanLinkSlice) {
	tv.Resize(13)
	for i := 0; i < tv.Len(); i++ {
		fillTestSpanLink(tv.At(i))
	}
}

func generateTestSpanLink() SpanLink {
	tv := NewSpanLink()
	tv.InitEmpty()
	fillTestSpanLink(tv)
	return tv
}

func fillTestSpanLink(tv SpanLink) {
	tv.SetTraceID(NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
	tv.SetSpanID(NewSpanID([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	tv.SetTraceState(TraceState("congo=congos"))
	tv.SetAttributes(generateTestAttributeMap())
	tv.SetDroppedAttributesCount(uint32(17))
}

func generateTestSpanStatus() SpanStatus {
	tv := NewSpanStatus()
	tv.InitEmpty()
	fillTestSpanStatus(tv)
	return tv
}

func fillTestSpanStatus(tv SpanStatus) {
	tv.SetCode(StatusCode(1))
	tv.SetMessage("cancelled")
}
