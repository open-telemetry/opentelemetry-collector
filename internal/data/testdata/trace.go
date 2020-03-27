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
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

var (
	TestStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestStartTimestamp = data.TimestampUnixNano(TestStartTime.UnixNano())

	TestEventTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestEventTimestamp = data.TimestampUnixNano(TestEventTime.UnixNano())

	TestEndTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestEndTimestamp = data.TimestampUnixNano(TestEndTime.UnixNano())
)

// TraceTestCase represents one test case with the coresponding TraceData and OTLP representation.
type TraceTestCase struct {
	Name      string
	TraceData data.TraceData
	OtlpData  []*otlptrace.ResourceSpans
}

var (
	NoResourceSpansTraceTestCase = TraceTestCase{
		Name:      "no-resource-spans",
		TraceData: data.NewTraceData(),
		OtlpData:  []*otlptrace.ResourceSpans(nil),
	}
	OneEmptyResourceSpansTraceTestCase = TraceTestCase{
		Name:      "one-empty-resource-spans",
		TraceData: generateTraceDataOneEmptyResourceSpans(),
		OtlpData:  generateTraceOtlpOneEmptyResourceSpans(),
	}
	NoLibrariesTraceTestCase = TraceTestCase{
		Name:      "no-libraries",
		TraceData: generateTraceDataNoLibraries(),
		OtlpData:  generateTraceOtlpNoLibraries(),
	}
	NoSpansTraceTestCase = TraceTestCase{
		Name:      "no-spans",
		TraceData: generateTraceDataNoSpans(),
		OtlpData:  generateTraceOtlpNoSpans(),
	}
	OneSpanNoResourceTraceTestCase = TraceTestCase{
		Name:      "one-span-no-resource",
		TraceData: generateTraceDataOneSpanNoResource(),
		OtlpData:  generateTraceOtlpOneSpanNoResource(),
	}
	OneSpanTraceTestCase = TraceTestCase{
		Name:      "one-span",
		TraceData: generateTraceDataOneSpan(),
		OtlpData:  generateTraceOtlpOneSpan(),
	}
	TwoSpansSameResourceTraceTestCase = TraceTestCase{
		Name:      "two-spans-same-resource",
		TraceData: generateTraceDataSameResourceTwoSpans(),
		OtlpData:  generateTraceOtlpSameResourceTwoSpans(),
	}
	TwoSpansSameResourceOneDifferentTraceTestCase = TraceTestCase{
		Name:      "two-spans-same-resource-one-different",
		TraceData: generateTraceDataTwoSpansSameResourceOneDifferent(),
		OtlpData:  generateTraceOtlpTwoSpansSameResourceOneDifferent(),
	}

	// AllTraceTestCases represents a set of test cases.
	AllTraceTestCases = []TraceTestCase{
		NoResourceSpansTraceTestCase,
		OneEmptyResourceSpansTraceTestCase,
		NoLibrariesTraceTestCase,
		NoSpansTraceTestCase,
		OneSpanNoResourceTraceTestCase,
		OneSpanTraceTestCase,
		TwoSpansSameResourceTraceTestCase,
		TwoSpansSameResourceOneDifferentTraceTestCase,
	}
)

func generateTraceDataOneEmptyResourceSpans() data.TraceData {
	td := data.NewTraceData()
	td.SetResourceSpans(data.NewResourceSpansSlice(1))
	return td
}

// generateTraceOtlpOneEmptyResourceSpans returns the OTLP representation of the generateTraceDataOneEmptyResourceSpans
func generateTraceOtlpOneEmptyResourceSpans() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{},
	}
}

func generateTraceDataNoLibraries() data.TraceData {
	td := generateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().Get(0)
	fillResource1(rs0.Resource())
	return td
}

// generateTraceOtlpDataNoLibraries returns the OTLP representation of the generateTraceDataNoLibraries.
func generateTraceOtlpNoLibraries() []*otlptrace.ResourceSpans {
	return []*otlptrace.ResourceSpans{
		{
			Resource: generateOtlpResource1(),
		},
	}
}

func generateTraceDataNoSpans() data.TraceData {
	td := generateTraceDataNoLibraries()
	rs0 := td.ResourceSpans().Get(0)
	rs0.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	return td
}

// generateTraceOtlpNoSpans returns the OTLP representation of the generateTraceDataNoSpans.
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

func generateTraceDataOneSpanNoResource() data.TraceData {
	td := generateTraceDataOneEmptyResourceSpans()
	rs0 := td.ResourceSpans().Get(0)
	rs0.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	rs0ils0 := rs0.InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(1))
	fillSpanOne(rs0ils0.Spans().Get(0))
	return td
}

// generateTraceOtlpOneSpanNoResource returns the OTLP representation of the generateTraceDataOneSpanNoResource.
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

func generateTraceDataOneSpan() data.TraceData {
	td := generateTraceDataNoSpans()
	rs0ils0 := td.ResourceSpans().Get(0).InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(1))
	fillSpanOne(rs0ils0.Spans().Get(0))
	return td
}

// generateTraceOtlpOneSpan returns the OTLP representation of the generateTraceDataOneSpan.
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

func generateTraceDataSameResourceTwoSpans() data.TraceData {
	td := generateTraceDataNoSpans()
	rs0ils0 := td.ResourceSpans().Get(0).InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(2))
	fillSpanOne(rs0ils0.Spans().Get(0))
	fillSpanTwo(rs0ils0.Spans().Get(1))
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

func generateTraceDataTwoSpansSameResourceOneDifferent() data.TraceData {
	td := data.NewTraceData()
	td.SetResourceSpans(data.NewResourceSpansSlice(2))
	rs0 := td.ResourceSpans().Get(0)
	rs0.Resource().SetAttributes(generateResourceAttributes1())
	rs0.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	rs0ils0 := rs0.InstrumentationLibrarySpans().Get(0)
	rs0ils0.SetSpans(data.NewSpanSlice(2))
	fillSpanOne(rs0ils0.Spans().Get(0))
	fillSpanTwo(rs0ils0.Spans().Get(1))
	rs1 := td.ResourceSpans().Get(1)
	fillResource2(rs1.Resource())
	rs1.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	rs1ils0 := rs1.InstrumentationLibrarySpans().Get(0)
	rs1ils0.SetSpans(data.NewSpanSlice(1))
	fillSpanThree(rs1ils0.Spans().Get(0))
	return td
}

// generateTraceOtlpTwoSpansSameResourceOneDifferent returns the OTLP representation of the generateTraceDataTwoSpansSameResourceOneDifferent.
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
	span.SetStartTime(TestStartTimestamp)
	span.SetEndTime(TestEndTimestamp)
	span.SetDroppedAttributesCount(1)
	span.SetEvents(data.NewSpanEventSlice(2))
	ev0 := span.Events().Get(0)
	ev0.SetTimestamp(TestEventTimestamp)
	ev0.SetName("event-with-attr")
	ev0.SetAttributes(generateSpanEventAttributes())
	ev0.SetDroppedAttributesCount(2)
	ev1 := span.Events().Get(1)
	ev1.SetTimestamp(TestEventTimestamp)
	ev1.SetName("event")
	ev1.SetDroppedAttributesCount(2)
	span.SetDroppedEventsCount(1)
	span.Status().SetCode(data.StatusCode(1))
	span.Status().SetMessage("status-cancelled")
}

func generateOtlpSpanOne() *otlptrace.Span {
	return &otlptrace.Span{
		Name:                   "operationA",
		StartTimeUnixNano:      uint64(TestStartTimestamp),
		EndTimeUnixNano:        uint64(TestEndTimestamp),
		DroppedAttributesCount: 1,
		Events: []*otlptrace.Span_Event{
			{
				Name:                   "event-with-attr",
				TimeUnixNano:           uint64(TestEventTimestamp),
				Attributes:             generateOtlpSpanEventAttributes(),
				DroppedAttributesCount: 2,
			},
			{
				Name:                   "event",
				TimeUnixNano:           uint64(TestEventTimestamp),
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
	span.SetStartTime(TestStartTimestamp)
	span.SetEndTime(TestEndTimestamp)
	span.SetLinks(data.NewSpanLinkSlice(2))
	span.Links().Get(0).SetAttributes(generateSpanLinkAttributes())
	span.Links().Get(0).SetDroppedAttributesCount(4)
	span.Links().Get(1).SetDroppedAttributesCount(4)
	span.SetDroppedLinksCount(3)
}

func generateOtlpSpanTwo() *otlptrace.Span {
	return &otlptrace.Span{
		Name:              "operationB",
		StartTimeUnixNano: uint64(TestStartTimestamp),
		EndTimeUnixNano:   uint64(TestEndTimestamp),
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
	span.SetStartTime(TestStartTimestamp)
	span.SetEndTime(TestEndTimestamp)
	span.SetAttributes(generateSpanAttributes())
	span.SetDroppedAttributesCount(5)
}

func generateOtlpSpanThree() *otlptrace.Span {
	return &otlptrace.Span{
		Name:                   "operationC",
		StartTimeUnixNano:      uint64(TestStartTimestamp),
		EndTimeUnixNano:        uint64(TestEndTimestamp),
		Attributes:             generateOtlpSpanAttributes(),
		DroppedAttributesCount: 5,
	}
}
