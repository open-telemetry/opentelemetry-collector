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

package testdata

import (
	"time"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

var (
	resourceAttributes1 = map[string]data.AttributeValue{"resource-attr": data.NewAttributeValueString("resource-attr-val-1")}
	resourceAttributes2 = map[string]data.AttributeValue{"resource-attr": data.NewAttributeValueString("resource-attr-val-2")}
	eventAttributes     = map[string]data.AttributeValue{"event-attr": data.NewAttributeValueString("event-attr-val")}
	linkAttributes      = map[string]data.AttributeValue{"link-attr": data.NewAttributeValueString("link-attr-val")}
	spanAttributes      = map[string]data.AttributeValue{"span-attr": data.NewAttributeValueString("span-attr-val")}

	TestStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestStartTimestamp = data.TimestampUnixNano(TestStartTime.UnixNano())

	TestEventTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestEventTimestamp = data.TimestampUnixNano(TestEventTime.UnixNano())

	TestEndTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestEndTimestamp = data.TimestampUnixNano(TestEndTime.UnixNano())
)

func GenerateTraceDataOneEmptyResourceSpans() data.TraceData {
	td := data.NewTraceData()
	td.SetResourceSpans(data.NewResourceSpansSlice(1))
	return td
}

func GenerateTraceDataNoLibraries() data.TraceData {
	td := GenerateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().Get(0)
	rs0.Resource().SetAttributes(data.NewAttributeMap(resourceAttributes1))
	return td
}

func GenerateTraceDataNoSpans() data.TraceData {
	td := GenerateTraceDataNoLibraries()
	rs0 := td.ResourceSpans().Get(0)
	rs0.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	return td
}

func GenerateTraceDataOneSpanNoResource() data.TraceData {
	td := GenerateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().Get(0)
	rs0.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	rs0ils0 := rs0.InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(1))
	fillSpanOne(rs0ils0.Spans().Get(0))
	return td
}

func GenerateTraceDataOneSpan() data.TraceData {
	td := GenerateTraceDataNoSpans()
	rs0ils0 := td.ResourceSpans().Get(0).InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(1))
	fillSpanOne(rs0ils0.Spans().Get(0))
	return td
}

func GenerateTraceDataSameResourcewoSpans() data.TraceData {
	td := GenerateTraceDataNoSpans()
	rs0ils0 := td.ResourceSpans().Get(0).InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(2))
	fillSpanOne(rs0ils0.Spans().Get(0))
	fillSpanTwo(rs0ils0.Spans().Get(1))
	return td
}

func GenerateTraceDataTwoSpansSameResourceOneDifferent() data.TraceData {
	td := data.NewTraceData()
	td.SetResourceSpans(data.NewResourceSpansSlice(2))
	rs0 := td.ResourceSpans().Get(0)
	rs0.Resource().SetAttributes(data.NewAttributeMap(resourceAttributes1))
	rs0.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	rs0ils0 := rs0.InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(2))
	fillSpanOne(rs0ils0.Spans().Get(0))
	fillSpanTwo(rs0ils0.Spans().Get(1))
	rs1 := td.ResourceSpans().Get(1)
	rs1.Resource().SetAttributes(data.NewAttributeMap(resourceAttributes2))
	rs1.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	rs1ils0 := rs1.InstrumentationLibrarySpans().Get(0)
	rs1ils0.SetSpans(data.NewSpanSlice(1))
	fillSpanThree(rs1ils0.Spans().Get(0))
	return td
}

func fillSpanOne(span data.Span) {
	span.SetName("operationA")
	span.SetStartTime(TestStartTimestamp)
	span.SetEndTime(TestEndTimestamp)
	span.SetDroppedAttributesCount(1)
	span.SetEvents(data.NewSpanEventSlice(2))
	ev0 := span.Events().Get(0)
	ev0.SetTimestamp(TestEventTimestamp)
	ev0.SetName("event-with-attr")
	ev0.SetAttributes(data.NewAttributeMap(eventAttributes))
	ev0.SetDroppedAttributesCount(2)
	ev1 := span.Events().Get(1)
	ev1.SetTimestamp(TestEventTimestamp)
	ev1.SetName("event")
	ev1.SetDroppedAttributesCount(2)
	span.SetDroppedEventsCount(1)
	span.Status().SetCode(data.StatusCode(1))
	span.Status().SetMessage("status-cancelled")
}

func fillSpanTwo(span data.Span) {
	span.SetName("operationB")
	span.SetStartTime(TestStartTimestamp)
	span.SetEndTime(TestEndTimestamp)
	span.SetLinks(data.NewSpanLinkSlice(2))
	span.Links().Get(0).SetAttributes(data.NewAttributeMap(linkAttributes))
	span.Links().Get(0).SetDroppedAttributesCount(4)
	span.Links().Get(1).SetDroppedAttributesCount(4)
	span.SetDroppedLinksCount(3)
}

func fillSpanThree(span data.Span) {
	span.SetName("operationC")
	span.SetStartTime(TestStartTimestamp)
	span.SetEndTime(TestEndTimestamp)
	span.SetAttributes(data.NewAttributeMap(spanAttributes))
	span.SetDroppedAttributesCount(5)
}
