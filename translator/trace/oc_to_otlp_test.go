// Copyright 2019 OpenTelemetry Authors
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

package tracetranslator

import (
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

func TestOcTraceStateToOtlp(t *testing.T) {
	assert.EqualValues(t, "", ocTraceStateToOtlp(nil))

	tracestate := &octrace.Span_Tracestate{
		Entries: []*octrace.Span_Tracestate_Entry{
			{
				Key:   "abc",
				Value: "def",
			},
		},
	}
	assert.EqualValues(t, "abc=def", ocTraceStateToOtlp(tracestate))

	tracestate.Entries = append(tracestate.Entries,
		&octrace.Span_Tracestate_Entry{
			Key:   "123",
			Value: "4567",
		})
	assert.EqualValues(t, "abc=def,123=4567", ocTraceStateToOtlp(tracestate))
}

func TestOcAttrsToOtlp(t *testing.T) {
	otlpAttrs, droppedCount := ocAttrsToOtlp(nil)
	assert.True(t, otlpAttrs == nil)
	assert.EqualValues(t, 0, droppedCount)

	ocAttrs := &octrace.Span_Attributes{}
	otlpAttrs, droppedCount = ocAttrsToOtlp(ocAttrs)
	assert.EqualValues(t, []*otlpcommon.AttributeKeyValue{}, otlpAttrs)
	assert.EqualValues(t, 0, droppedCount)

	ocAttrs = &octrace.Span_Attributes{
		DroppedAttributesCount: 123,
	}
	otlpAttrs, droppedCount = ocAttrsToOtlp(ocAttrs)
	assert.EqualValues(t, []*otlpcommon.AttributeKeyValue{}, otlpAttrs)
	assert.EqualValues(t, 123, droppedCount)

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap:           map[string]*octrace.AttributeValue{},
		DroppedAttributesCount: 234,
	}
	otlpAttrs, droppedCount = ocAttrsToOtlp(ocAttrs)
	assert.EqualValues(t, []*otlpcommon.AttributeKeyValue{}, otlpAttrs)
	assert.EqualValues(t, 234, droppedCount)

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap: map[string]*octrace.AttributeValue{
			"abc": {
				Value: &octrace.AttributeValue_StringValue{StringValue: &octrace.TruncatableString{Value: "def"}},
			},
		},
		DroppedAttributesCount: 234,
	}
	otlpAttrs, droppedCount = ocAttrsToOtlp(ocAttrs)
	assert.EqualValues(t, []*otlpcommon.AttributeKeyValue{
		{
			Key:         "abc",
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "def",
		},
	}, otlpAttrs)
	assert.EqualValues(t, 234, droppedCount)

	ocAttrs.AttributeMap["intval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_IntValue{IntValue: 345},
	}
	ocAttrs.AttributeMap["boolval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_BoolValue{BoolValue: true},
	}
	ocAttrs.AttributeMap["doubleval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_DoubleValue{DoubleValue: 4.5},
	}
	otlpAttrs, droppedCount = ocAttrsToOtlp(ocAttrs)
	assert.EqualValues(t, 234, droppedCount)
	assert.EqualValues(t, 4, len(otlpAttrs))
	assertHasAttrVal(t, otlpAttrs, "abc", &otlpcommon.AttributeKeyValue{
		Key:         "abc",
		Type:        otlpcommon.AttributeKeyValue_STRING,
		StringValue: "def",
	})
	assertHasAttrVal(t, otlpAttrs, "intval", &otlpcommon.AttributeKeyValue{
		Key:      "intval",
		Type:     otlpcommon.AttributeKeyValue_INT,
		IntValue: 345,
	})
	assertHasAttrVal(t, otlpAttrs, "boolval", &otlpcommon.AttributeKeyValue{
		Key:       "boolval",
		Type:      otlpcommon.AttributeKeyValue_BOOL,
		BoolValue: true,
	})
	assertHasAttrVal(t, otlpAttrs, "doubleval", &otlpcommon.AttributeKeyValue{
		Key:         "doubleval",
		Type:        otlpcommon.AttributeKeyValue_DOUBLE,
		DoubleValue: 4.5,
	})
}

func TestOcSpanKindToOtlp(t *testing.T) {
	tests := []struct {
		ocAttrs  *octrace.Span_Attributes
		ocKind   octrace.Span_SpanKind
		otlpKind otlptrace.Span_SpanKind
	}{
		{
			ocKind:   octrace.Span_CLIENT,
			otlpKind: otlptrace.Span_CLIENT,
		},
		{
			ocKind:   octrace.Span_SERVER,
			otlpKind: otlptrace.Span_SERVER,
		},
		{
			ocKind:   octrace.Span_SPAN_KIND_UNSPECIFIED,
			otlpKind: otlptrace.Span_SPAN_KIND_UNSPECIFIED,
		},
		{
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
			ocAttrs: &octrace.Span_Attributes{
				AttributeMap: map[string]*octrace.AttributeValue{
					"span.kind": {Value: &octrace.AttributeValue_StringValue{
						StringValue: &octrace.TruncatableString{Value: "consumer"}}},
				},
			},
			otlpKind: otlptrace.Span_CONSUMER,
		},
		{
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
			ocAttrs: &octrace.Span_Attributes{
				AttributeMap: map[string]*octrace.AttributeValue{
					"span.kind": {Value: &octrace.AttributeValue_StringValue{
						StringValue: &octrace.TruncatableString{Value: "producer"}}},
				},
			},
			otlpKind: otlptrace.Span_PRODUCER,
		},
		{
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
			ocAttrs: &octrace.Span_Attributes{
				AttributeMap: map[string]*octrace.AttributeValue{
					"span.kind": {Value: &octrace.AttributeValue_IntValue{
						IntValue: 123}},
				},
			},
			otlpKind: otlptrace.Span_SPAN_KIND_UNSPECIFIED,
		},
		{
			ocKind: octrace.Span_CLIENT,
			ocAttrs: &octrace.Span_Attributes{
				AttributeMap: map[string]*octrace.AttributeValue{
					"span.kind": {Value: &octrace.AttributeValue_StringValue{
						StringValue: &octrace.TruncatableString{Value: "consumer"}}},
				},
			},
			otlpKind: otlptrace.Span_CLIENT,
		},
	}

	for _, test := range tests {
		t.Run(test.otlpKind.String(), func(t *testing.T) {
			got := ocSpanKindToOtlp(test.ocKind, test.ocAttrs)
			assert.EqualValues(t, test.otlpKind, got, "Expected "+test.otlpKind.String()+", got "+got.String())
		})
	}
}

func TestOcToOtlp(t *testing.T) {
	ocNode := &occommon.Node{}
	ocResource := &ocresource.Resource{}

	timestampP, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	ocSpan1 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationB"},
		StartTime: timestampP,
		EndTime:   timestampP,
		TimeEvents: &octrace.Span_TimeEvents{
			TimeEvent: []*octrace.Span_TimeEvent{
				{
					Time: timestampP,
					Value: &octrace.Span_TimeEvent_Annotation_{
						Annotation: &octrace.Span_TimeEvent_Annotation{
							Description: &octrace.TruncatableString{Value: "event1"},
							Attributes: &octrace.Span_Attributes{
								AttributeMap: map[string]*octrace.AttributeValue{
									"eventattr1": {
										Value: &octrace.AttributeValue_StringValue{
											StringValue: &octrace.TruncatableString{Value: "eventattrval1"},
										},
									},
								},
								DroppedAttributesCount: 4,
							},
						},
					},
				},
				nil,
			},
			DroppedAnnotationsCount:   1,
			DroppedMessageEventsCount: 2,
		},
		Status: &octrace.Status{Message: "status-cancelled", Code: 1},
	}

	ocSpan2 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationC"},
		StartTime: timestampP,
		EndTime:   timestampP,
		Links: &octrace.Span_Links{
			Link:              []*octrace.Span_Link{{}, nil},
			DroppedLinksCount: 1,
		},
	}

	ocSpan3 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationD"},
		StartTime: timestampP,
		EndTime:   timestampP,
		Resource:  ocResource,
	}

	otlpSpan1 := &otlptrace.Span{
		Name:              "operationB",
		StartTimeUnixnano: timestampToUnixnano(timestampP),
		EndTimeUnixnano:   timestampToUnixnano(timestampP),
		Events: []*otlptrace.Span_Event{
			{
				TimeUnixnano: timestampToUnixnano(timestampP),
				Name:         "event1",
				Attributes: []*otlpcommon.AttributeKeyValue{
					{
						Key:         "eventattr1",
						Type:        otlpcommon.AttributeKeyValue_STRING,
						StringValue: "eventattrval1",
					},
				},
				DroppedAttributesCount: 4,
			},
		},
		DroppedEventsCount: 3,
		Status:             &otlptrace.Status{Message: "status-cancelled", Code: otlptrace.Status_Cancelled},
	}

	otlpSpan2 := &otlptrace.Span{
		Name:              "operationC",
		StartTimeUnixnano: timestampToUnixnano(timestampP),
		EndTimeUnixnano:   timestampToUnixnano(timestampP),
		Links:             []*otlptrace.Span_Link{{}},
		DroppedLinksCount: 1,
	}

	otlpSpan3 := &otlptrace.Span{
		Name:              "operationD",
		StartTimeUnixnano: timestampToUnixnano(timestampP),
		EndTimeUnixnano:   timestampToUnixnano(timestampP),
	}

	tests := []struct {
		name string
		oc   consumerdata.TraceData
		otlp []*otlptrace.ResourceSpans
	}{
		{
			name: "empty",
			oc:   consumerdata.TraceData{},
			otlp: nil,
		},

		{
			name: "nil-resource",
			oc:   consumerdata.TraceData{Node: ocNode},
			otlp: []*otlptrace.ResourceSpans{
				{Resource: &otlpresource.Resource{}},
			},
		},

		{
			name: "nil-node",
			oc:   consumerdata.TraceData{Resource: ocResource},
			otlp: []*otlptrace.ResourceSpans{
				{Resource: &otlpresource.Resource{}},
			},
		},

		{
			name: "no-spans",
			oc:   consumerdata.TraceData{Node: ocNode, Resource: ocResource},
			otlp: []*otlptrace.ResourceSpans{
				{Resource: &otlpresource.Resource{}},
			},
		},

		{
			name: "one-spans",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1},
			},
			otlp: []*otlptrace.ResourceSpans{
				{
					Resource: &otlpresource.Resource{},
					InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
						{
							Spans: []*otlptrace.Span{otlpSpan1},
						},
					},
				},
			},
		},

		{
			name: "two-spans",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1, nil, ocSpan2},
			},
			otlp: []*otlptrace.ResourceSpans{
				{
					Resource: &otlpresource.Resource{},
					InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
						{
							Spans: []*otlptrace.Span{otlpSpan1, otlpSpan2},
						},
					},
				},
			},
		},

		{
			name: "two-spans-plus-one-separate",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1, ocSpan2, ocSpan3},
			},
			otlp: []*otlptrace.ResourceSpans{
				{
					Resource: &otlpresource.Resource{},
					InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
						{
							Spans: []*otlptrace.Span{otlpSpan1, otlpSpan2},
						},
					},
				},
				{
					Resource: &otlpresource.Resource{},
					InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
						{
							Spans: []*otlptrace.Span{otlpSpan3},
						},
					},
				},
			},
		},

		{
			name: "two-spans-and-separate-in-the-middle",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1, ocSpan3, ocSpan2},
			},
			otlp: []*otlptrace.ResourceSpans{
				{
					Resource: &otlpresource.Resource{},
					InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
						{
							Spans: []*otlptrace.Span{otlpSpan1, otlpSpan2},
						},
					},
				},
				{
					Resource: &otlpresource.Resource{},
					InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
						{
							Spans: []*otlptrace.Span{otlpSpan3},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := OCToOTLP(test.oc)
			assert.EqualValues(t, test.otlp, got)
		})
	}
}

func assertHasAttrVal(t *testing.T, attrs []*otlpcommon.AttributeKeyValue, key string, val *otlpcommon.AttributeKeyValue) {
	found := false
	for _, attr := range attrs {
		if attr != nil && attr.Key == key {
			assert.EqualValues(t, false, found, "Duplicate key "+key)
			assert.EqualValues(t, val, attr)
			found = true
		}
	}
	assert.EqualValues(t, true, found, "Cannot find key "+key)
}
