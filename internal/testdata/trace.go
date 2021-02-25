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

	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"

	"go.opentelemetry.io/collector/consumer/pdata"
)

var (
	TestSpanStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestSpanStartTimestamp = pdata.TimestampFromTime(TestSpanStartTime)

	TestSpanEventTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestSpanEventTimestamp = pdata.TimestampFromTime(TestSpanEventTime)

	TestSpanEndTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestSpanEndTimestamp = pdata.TimestampFromTime(TestSpanEndTime)
)

func GenerateTraceDataEmpty() pdata.Traces {
	td := pdata.NewTraces()
	return td
}

func generateTraceOtlpEmpty() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans(nil)
}

func GenerateTraceDataOneEmptyResourceSpans() pdata.Traces {
	td := GenerateTraceDataEmpty()
	td.ResourceSpans().Resize(1)
	return td
}

func generateTraceOtlpOneEmptyResourceSpans() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{},
	}
}

func GenerateTraceDataNoLibraries() pdata.Traces {
	td := GenerateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().At(0)
	initResource1(rs0.Resource())
	return td
}

func generateTraceOtlpNoLibraries() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
		},
	}
}

func GenerateTraceDataOneEmptyInstrumentationLibrary() pdata.Traces {
	td := GenerateTraceDataNoLibraries()
	rs0 := td.ResourceSpans().At(0)
	rs0.InstrumentationLibrarySpans().Resize(1)
	return td
}

func generateTraceOtlpOneEmptyInstrumentationLibrary() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{},
			},
		},
	}
}

func GenerateTraceDataOneSpanNoResource() pdata.Traces {
	td := GenerateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().At(0)
	rs0.InstrumentationLibrarySpans().Resize(1)
	rs0ils0 := rs0.InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(1)
	fillSpanOne(rs0ils0.Spans().At(0))
	return td
}

func generateTraceOtlpOneSpanNoResource() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{
						generateOtlpSpanOne(),
					},
				},
			},
		},
	}
}

func GenerateTraceDataOneSpan() pdata.Traces {
	td := GenerateTraceDataOneEmptyInstrumentationLibrary()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(1)
	fillSpanOne(rs0ils0.Spans().At(0))
	return td
}

func generateTraceOtlpOneSpan() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{
						generateOtlpSpanOne(),
					},
				},
			},
		},
	}
}

func GenerateTraceDataTwoSpansSameResource() pdata.Traces {
	td := GenerateTraceDataOneEmptyInstrumentationLibrary()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(2)
	fillSpanOne(rs0ils0.Spans().At(0))
	fillSpanTwo(rs0ils0.Spans().At(1))
	return td
}

// GenerateTraceOtlpSameResourceTwoSpans returns the OTLP representation of the GenerateTraceOtlpSameResourceTwoSpans.
func GenerateTraceOtlpSameResourceTwoSpans() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{
						generateOtlpSpanOne(),
						generateOtlpSpanTwo(),
					},
				},
			},
		},
	}
}

func GenerateTraceDataTwoSpansSameResourceOneDifferent() pdata.Traces {
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(2)
	rs0 := td.ResourceSpans().At(0)
	initResource1(rs0.Resource())
	rs0.InstrumentationLibrarySpans().Resize(1)
	rs0ils0 := rs0.InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(2)
	fillSpanOne(rs0ils0.Spans().At(0))
	fillSpanTwo(rs0ils0.Spans().At(1))
	rs1 := td.ResourceSpans().At(1)
	initResource2(rs1.Resource())
	rs1.InstrumentationLibrarySpans().Resize(1)
	rs1ils0 := rs1.InstrumentationLibrarySpans().At(0)
	rs1ils0.Spans().Resize(1)
	fillSpanThree(rs1ils0.Spans().At(0))
	return td
}

func GenerateTraceDataManySpansSameResource(spansCount int) pdata.Traces {
	td := GenerateTraceDataOneEmptyInstrumentationLibrary()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(spansCount)
	for i := 0; i < spansCount; i++ {
		fillSpanOne(rs0ils0.Spans().At(i))
	}
	return td
}

func generateTraceOtlpTwoSpansSameResourceOneDifferent() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{
						generateOtlpSpanOne(),
						generateOtlpSpanTwo(),
					},
				},
			},
		},
		{
			Resource: generateOtlpResource2(),
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{
						generateOtlpSpanThree(),
					},
				},
			},
		},
	}
}

func fillSpanOne(span pdata.Span) {
	span.SetName("operationA")
	span.SetStartTime(TestSpanStartTimestamp)
	span.SetEndTime(TestSpanEndTimestamp)
	span.SetDroppedAttributesCount(1)
	evs := span.Events()
	evs.Resize(2)
	ev0 := evs.At(0)
	ev0.SetTimestamp(TestSpanEventTimestamp)
	ev0.SetName("event-with-attr")
	initSpanEventAttributes(ev0.Attributes())
	ev0.SetDroppedAttributesCount(2)
	ev1 := evs.At(1)
	ev1.SetTimestamp(TestSpanEventTimestamp)
	ev1.SetName("event")
	ev1.SetDroppedAttributesCount(2)
	span.SetDroppedEventsCount(1)
	status := span.Status()
	status.SetCode(pdata.StatusCodeError)
	status.SetMessage("status-cancelled")
}

func generateOtlpSpanOne() *otlptrace.Span {
	return &otlptrace.Span{
		Name:                   "operationA",
		StartTimeUnixNano:      uint64(TestSpanStartTimestamp),
		EndTimeUnixNano:        uint64(TestSpanEndTimestamp),
		DroppedAttributesCount: 1,
		Events: []*otlptrace.Span_Event{
			{
				Name:                   "event-with-attr",
				TimeUnixNano:           uint64(TestSpanEventTimestamp),
				Attributes:             generateOtlpSpanEventAttributes(),
				DroppedAttributesCount: 2,
			},
			{
				Name:                   "event",
				TimeUnixNano:           uint64(TestSpanEventTimestamp),
				DroppedAttributesCount: 2,
			},
		},
		DroppedEventsCount: 1,
		Status: otlptrace.Status{
			Code:           otlptrace.Status_STATUS_CODE_ERROR,
			DeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
			Message:        "status-cancelled",
		},
	}
}

func fillSpanTwo(span pdata.Span) {
	span.SetName("operationB")
	span.SetStartTime(TestSpanStartTimestamp)
	span.SetEndTime(TestSpanEndTimestamp)
	span.Links().Resize(2)
	initSpanLinkAttributes(span.Links().At(0).Attributes())
	span.Links().At(0).SetDroppedAttributesCount(4)
	span.Links().At(1).SetDroppedAttributesCount(4)
	span.SetDroppedLinksCount(3)
}

func generateOtlpSpanTwo() *otlptrace.Span {
	return &otlptrace.Span{
		Name:              "operationB",
		StartTimeUnixNano: uint64(TestSpanStartTimestamp),
		EndTimeUnixNano:   uint64(TestSpanEndTimestamp),
		Links: []*otlptrace.Span_Link{
			{
				Attributes:             generateOtlpSpanLinkAttributes(),
				DroppedAttributesCount: 4,
			},
			{
				DroppedAttributesCount: 4,
			},
		},
		DroppedLinksCount: 3,
	}
}

func fillSpanThree(span pdata.Span) {
	span.SetName("operationC")
	span.SetStartTime(TestSpanStartTimestamp)
	span.SetEndTime(TestSpanEndTimestamp)
	initSpanAttributes(span.Attributes())
	span.SetDroppedAttributesCount(5)
}

func generateOtlpSpanThree() *otlptrace.Span {
	return &otlptrace.Span{
		Name:                   "operationC",
		StartTimeUnixNano:      uint64(TestSpanStartTimestamp),
		EndTimeUnixNano:        uint64(TestSpanEndTimestamp),
		Attributes:             generateOtlpSpanAttributes(),
		DroppedAttributesCount: 5,
	}
}
