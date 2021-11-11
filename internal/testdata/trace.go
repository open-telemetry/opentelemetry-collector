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

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/model/pdata"
)

var (
	TestSpanStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestSpanStartTimestamp = pdata.NewTimestampFromTime(TestSpanStartTime)

	TestSpanEventTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestSpanEventTimestamp = pdata.NewTimestampFromTime(TestSpanEventTime)

	TestSpanEndTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestSpanEndTimestamp = pdata.NewTimestampFromTime(TestSpanEndTime)
)

func GenerateTracesOneEmptyResourceSpans() pdata.Traces {
	td := pdata.NewTraces()
	td.ResourceSpans().AppendEmpty()
	return td
}

func GenerateTracesNoLibraries() pdata.Traces {
	td := GenerateTracesOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().At(0)
	initResource1(rs0.Resource())
	return td
}

func GenerateTracesOneEmptyInstrumentationLibrary() pdata.Traces {
	td := GenerateTracesNoLibraries()
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().AppendEmpty()
	return td
}

func GenerateTracesOneSpanNoResource() pdata.Traces {
	td := GenerateTracesOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().At(0)
	fillSpanOne(rs0.InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty())
	return td
}

func GenerateTracesOneSpan() pdata.Traces {
	td := GenerateTracesOneEmptyInstrumentationLibrary()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	fillSpanOne(rs0ils0.Spans().AppendEmpty())
	return td
}

func GenerateTracesTwoSpansSameResource() pdata.Traces {
	td := GenerateTracesOneEmptyInstrumentationLibrary()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	fillSpanOne(rs0ils0.Spans().AppendEmpty())
	fillSpanTwo(rs0ils0.Spans().AppendEmpty())
	return td
}

func GenerateTracesTwoSpansSameResourceOneDifferent() pdata.Traces {
	td := pdata.NewTraces()
	rs0 := td.ResourceSpans().AppendEmpty()
	initResource1(rs0.Resource())
	rs0ils0 := rs0.InstrumentationLibrarySpans().AppendEmpty()
	fillSpanOne(rs0ils0.Spans().AppendEmpty())
	fillSpanTwo(rs0ils0.Spans().AppendEmpty())
	rs1 := td.ResourceSpans().AppendEmpty()
	initResource2(rs1.Resource())
	rs1ils0 := rs1.InstrumentationLibrarySpans().AppendEmpty()
	fillSpanThree(rs1ils0.Spans().AppendEmpty())
	return td
}

func GenerateTracesManySpansSameResource(spanCount int) pdata.Traces {
	td := GenerateTracesOneEmptyInstrumentationLibrary()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().EnsureCapacity(spanCount)
	for i := 0; i < spanCount; i++ {
		fillSpanOne(rs0ils0.Spans().AppendEmpty())
	}
	return td
}

func fillSpanOne(span pdata.Span) {
	span.SetName("operationA")
	span.SetStartTimestamp(TestSpanStartTimestamp)
	span.SetEndTimestamp(TestSpanEndTimestamp)
	span.SetDroppedAttributesCount(1)
	span.SetTraceID(pdata.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}))
	span.SetSpanID(pdata.NewSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}))
	evs := span.Events()
	ev0 := evs.AppendEmpty()
	ev0.SetTimestamp(TestSpanEventTimestamp)
	ev0.SetName("event-with-attr")
	initSpanEventAttributes(ev0.Attributes())
	ev0.SetDroppedAttributesCount(2)
	ev1 := evs.AppendEmpty()
	ev1.SetTimestamp(TestSpanEventTimestamp)
	ev1.SetName("event")
	ev1.SetDroppedAttributesCount(2)
	span.SetDroppedEventsCount(1)
	status := span.Status()
	status.SetCode(pdata.StatusCodeError)
	status.SetMessage("status-cancelled")
}

func fillSpanTwo(span pdata.Span) {
	span.SetName("operationB")
	span.SetStartTimestamp(TestSpanStartTimestamp)
	span.SetEndTimestamp(TestSpanEndTimestamp)
	link0 := span.Links().AppendEmpty()
	initSpanLinkAttributes(link0.Attributes())
	link0.SetDroppedAttributesCount(4)
	link1 := span.Links().AppendEmpty()
	link1.SetDroppedAttributesCount(4)
	span.SetDroppedLinksCount(3)
}

func fillSpanThree(span pdata.Span) {
	span.SetName("operationC")
	span.SetStartTimestamp(TestSpanStartTimestamp)
	span.SetEndTimestamp(TestSpanEndTimestamp)
	initSpanAttributes(span.Attributes())
	span.SetDroppedAttributesCount(5)
}
