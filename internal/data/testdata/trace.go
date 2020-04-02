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

	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

var (
	TestSpanStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestSpanStartTimestamp = data.TimestampUnixNano(TestSpanStartTime.UnixNano())

	TestSpanEventTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestSpanEventTimestamp = data.TimestampUnixNano(TestSpanEventTime.UnixNano())

	TestSpanEndTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestSpanEndTimestamp = data.TimestampUnixNano(TestSpanEndTime.UnixNano())
)

const (
	NumTraceTests = 8
)

func GenerateTraceDataEmpty() data.TraceData {
	td := data.NewTraceData()
	return td
}

func generateTraceOtlpEmpty() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans(nil)
}

func GenerateTraceDataOneEmptyResourceSpans() data.TraceData {
	td := GenerateTraceDataEmpty()
	td.ResourceSpans().Resize(1)
	return td
}

// generateTraceOtlpOneEmptyResourceSpans returns the OTLP representation of the GenerateTraceDataOneEmptyResourceSpans
func generateTraceOtlpOneEmptyResourceSpans() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{},
	}
}

func GenerateTraceDataNoLibraries() data.TraceData {
	td := GenerateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().At(0)
	initResource1(rs0.Resource())
	return td
}

// generateTraceOtlpDataNoLibraries returns the OTLP representation of the GenerateTraceDataNoLibraries.
func generateTraceOtlpNoLibraries() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
		},
	}
}

func GenerateTraceDataNoSpans() data.TraceData {
	td := GenerateTraceDataNoLibraries()
	rs0 := td.ResourceSpans().At(0)
	rs0.InstrumentationLibrarySpans().Resize(1)
	return td
}

// generateTraceOtlpNoSpans returns the OTLP representation of the GenerateTraceDataNoSpans.
func generateTraceOtlpNoSpans() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{},
			},
		},
	}
}

func GenerateTraceDataOneSpanNoResource() data.TraceData {
	td := GenerateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().At(0)
	rs0.InstrumentationLibrarySpans().Resize(1)
	rs0ils0 := rs0.InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(1)
	fillSpanOne(rs0ils0.Spans().At(0))
	return td
}

// generateTraceOtlpOneSpanNoResource returns the OTLP representation of the GenerateTraceDataOneSpanNoResource.
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

func GenerateTraceDataOneSpan() data.TraceData {
	td := GenerateTraceDataNoSpans()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(1)
	fillSpanOne(rs0ils0.Spans().At(0))
	return td
}

// generateTraceOtlpOneSpan returns the OTLP representation of the GenerateTraceDataOneSpan.
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

func GenerateTraceDataTwoSpansSameResource() data.TraceData {
	td := GenerateTraceDataNoSpans()
	rs0ils0 := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	rs0ils0.Spans().Resize(2)
	fillSpanOne(rs0ils0.Spans().At(0))
	fillSpanTwo(rs0ils0.Spans().At(1))
	return td
}

// generateTraceOtlpSameResourcewoSpans returns the OTLP representation of the generateTraceDataSameResourcewoSpans.
func generateTraceOtlpSameResourceTwoSpans() []*otlptrace.ResourceSpans {
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

func GenerateTraceDataTwoSpansSameResourceOneDifferent() data.TraceData {
	td := data.NewTraceData()
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

// generateTraceOtlpTwoSpansSameResourceOneDifferent returns the OTLP representation of the GenerateTraceDataTwoSpansSameResourceOneDifferent.
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

func fillSpanOne(span data.Span) {
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
	status.InitEmpty()
	status.SetCode(data.StatusCode(1))
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
		Status: &otlptrace.Status{
			Code:    otlptrace.Status_Cancelled,
			Message: "status-cancelled",
		},
	}
}

func fillSpanTwo(span data.Span) {
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

func fillSpanThree(span data.Span) {
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
