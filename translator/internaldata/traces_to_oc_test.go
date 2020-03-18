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

package internaldata

import (
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	v1 "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestInternalTraceStateToOC(t *testing.T) {
	assert.Equal(t, (*v1.Span_Tracestate)(nil), traceStateToOC(data.TraceState("")))

	ocTracestate := &octrace.Span_Tracestate{
		Entries: []*octrace.Span_Tracestate_Entry{
			{
				Key:   "abc",
				Value: "def",
			},
		},
	}
	assert.EqualValues(t, ocTracestate, traceStateToOC(data.TraceState("abc=def")))

	ocTracestate.Entries = append(ocTracestate.Entries,
		&octrace.Span_Tracestate_Entry{
			Key:   "123",
			Value: "4567",
		})
	assert.EqualValues(t, ocTracestate, traceStateToOC(data.TraceState("abc=def,123=4567")))
}

func TestAttributesMapToOC(t *testing.T) {
	assert.EqualValues(t, (*v1.Span_Attributes)(nil), attributesMapToOCSpanAttributes(data.AttributesMap{}, 0))

	ocAttrs := &octrace.Span_Attributes{
		DroppedAttributesCount: 123,
	}
	assert.EqualValues(t, ocAttrs, attributesMapToOCSpanAttributes(data.AttributesMap{}, 123))

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap: map[string]*octrace.AttributeValue{
			"abc": {
				Value: &octrace.AttributeValue_StringValue{StringValue: &octrace.TruncatableString{Value: "def"}},
			},
		},
		DroppedAttributesCount: 234,
	}
	assert.EqualValues(t, ocAttrs,
		attributesMapToOCSpanAttributes(
			data.AttributesMap{
				"abc": data.NewAttributeValueString("def"),
			},
			234))

	ocAttrs.AttributeMap["intval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_IntValue{IntValue: 345},
	}
	ocAttrs.AttributeMap["boolval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_BoolValue{BoolValue: true},
	}
	ocAttrs.AttributeMap["doubleval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_DoubleValue{DoubleValue: 4.5},
	}
	assert.EqualValues(t, ocAttrs,
		attributesMapToOCSpanAttributes(
			data.AttributesMap{
				"abc":       data.NewAttributeValueString("def"),
				"intval":    data.NewAttributeValueInt(345),
				"boolval":   data.NewAttributeValueBool(true),
				"doubleval": data.NewAttributeValueDouble(4.5),
			},
			234))
}

func TestSpanKindToOC(t *testing.T) {
	tests := []struct {
		kind   data.SpanKind
		ocKind octrace.Span_SpanKind
	}{
		{
			kind:   data.SpanKindCLIENT,
			ocKind: octrace.Span_CLIENT,
		},
		{
			kind:   data.SpanKindSERVER,
			ocKind: octrace.Span_SERVER,
		},
		{
			kind:   data.SpanKindCONSUMER,
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
		},
		{
			kind:   data.SpanKindPRODUCER,
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
		},
		{
			kind:   data.SpanKindUNSPECIFIED,
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
		},
	}

	for _, test := range tests {
		t.Run(test.kind.String(), func(t *testing.T) {
			got := spanKindToOC(test.kind)
			assert.EqualValues(t, test.ocKind, got, "Expected "+test.ocKind.String()+", got "+got.String())
		})
	}
}

func TestSpanKindToOCAttribute(t *testing.T) {
	tests := []struct {
		kind        data.SpanKind
		ocAttribute *octrace.AttributeValue
	}{
		{
			kind: data.SpanKindCONSUMER,
			ocAttribute: &octrace.AttributeValue{
				Value: &octrace.AttributeValue_StringValue{
					StringValue: &octrace.TruncatableString{
						Value: string(tracetranslator.OpenTracingSpanKindConsumer),
					},
				},
			},
		},
		{
			kind: data.SpanKindPRODUCER,
			ocAttribute: &octrace.AttributeValue{
				Value: &octrace.AttributeValue_StringValue{
					StringValue: &octrace.TruncatableString{
						Value: string(tracetranslator.OpenTracingSpanKindProducer),
					},
				},
			},
		},
		{
			kind:        data.SpanKindUNSPECIFIED,
			ocAttribute: nil,
		},
		{
			kind:        data.SpanKindSERVER,
			ocAttribute: nil,
		},
		{
			kind:        data.SpanKindCLIENT,
			ocAttribute: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.kind.String(), func(t *testing.T) {
			got := spanKindToOCAttribute(test.kind)
			assert.EqualValues(t, test.ocAttribute, got, "Expected "+test.ocAttribute.String()+", got "+got.String())
		})
	}
}

func TestInternalToOC(t *testing.T) {
	timestampP, err := ptypes.TimestampProto(time.Date(2020, 3, 9, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	resource := data.NewResource()
	resource.SetAttributes(data.NewAttributeMap(map[string]data.AttributeValue{"label1": data.NewAttributeValueString("value1")}))

	span1 := data.NewSpan()
	span1.SetName("operationB")
	span1.SetStartTime(internal.TimestampToUnixnano(timestampP))
	span1.SetEndTime(internal.TimestampToUnixnano(timestampP))
	span1.SetEvents([]*data.SpanEvent{
		data.NewSpanEvent(
			internal.TimestampToUnixnano(timestampP),
			"event1",
			data.NewAttributes(
				data.AttributesMap{
					"eventattr1": data.NewAttributeValueString("eventattrval1"),
				},
				4),
		),
	})
	span1.SetDroppedEventsCount(3)
	span1.SetStatus(data.NewSpanStatus(data.StatusCode(1), "status-cancelled"))

	span2 := data.NewSpan()
	span2.SetName("operationC")
	span2.SetStartTime(internal.TimestampToUnixnano(timestampP))
	span2.SetEndTime(internal.TimestampToUnixnano(timestampP))
	span2.SetLinks([]*data.SpanLink{data.NewSpanLink()})
	span2.SetDroppedLinksCount(1)
	span2event := data.NewSpanEvent(
		internal.TimestampToUnixnano(timestampP),
		"",
		data.NewAttributes(
			data.AttributesMap{
				conventions.OCTimeEventMessageEventType:  data.NewAttributeValueString(octrace.Span_TimeEvent_MessageEvent_SENT.String()),
				conventions.OCTimeEventMessageEventID:    data.NewAttributeValueInt(123),
				conventions.OCTimeEventMessageEventUSize: data.NewAttributeValueInt(345),
				conventions.OCTimeEventMessageEventCSize: data.NewAttributeValueInt(234),
			},
			0),
	)
	span2.SetEvents([]*data.SpanEvent{span2event})

	span3 := data.NewSpan()
	span3.SetName("operationD")
	span3.SetStartTime(internal.TimestampToUnixnano(timestampP))
	span3.SetEndTime(internal.TimestampToUnixnano(timestampP))
	span3ResourceType := "resource2"
	span3Resource := data.NewResource()
	span3Resource.SetAttributes(data.NewAttributeMap(map[string]data.AttributeValue{
		conventions.OCAttributeResourceType: data.NewAttributeValueString(span3ResourceType)}))
	resourceSpans3 := data.NewResourceSpans(span3Resource, []*data.InstrumentationLibrarySpans{
		data.NewInstrumentationLibrarySpans(data.NewInstrumentationLibrary(), []*data.Span{span3})})

	ocNode := &occommon.Node{}
	ocResource := &ocresource.Resource{
		Labels: map[string]string{
			"label1": "value1",
		},
	}

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
			},
			DroppedMessageEventsCount: 3,
		},
		Status: &octrace.Status{Message: "status-cancelled", Code: 1},
	}

	ocSpan2 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationC"},
		StartTime: timestampP,
		EndTime:   timestampP,
		Links: &octrace.Span_Links{
			Link:              []*octrace.Span_Link{{}},
			DroppedLinksCount: 1,
		},
		TimeEvents: &octrace.Span_TimeEvents{
			TimeEvent: []*octrace.Span_TimeEvent{
				{
					Time: timestampP,
					Value: &octrace.Span_TimeEvent_MessageEvent_{
						MessageEvent: &octrace.Span_TimeEvent_MessageEvent{
							Type:             octrace.Span_TimeEvent_MessageEvent_SENT,
							Id:               123,
							UncompressedSize: 345,
							CompressedSize:   234,
						},
					},
				},
			},
		},
	}

	ocSpan3 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationD"},
		StartTime: timestampP,
		EndTime:   timestampP,
	}

	tests := []struct {
		name     string
		internal data.TraceData
		oc       []consumerdata.TraceData
	}{
		{
			name:     "empty",
			internal: data.TraceData{},
			oc:       []consumerdata.TraceData{},
		},

		{
			name: "no-spans",
			internal: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(data.NewResource(), []*data.InstrumentationLibrarySpans{
					data.NewInstrumentationLibrarySpans(data.NewInstrumentationLibrary(), []*data.Span{})}),
			}),
			oc: []consumerdata.TraceData{
				{
					Node:         ocNode,
					Resource:     &ocresource.Resource{},
					Spans:        []*octrace.Span{},
					SourceFormat: sourceFormat,
				},
			},
		},

		{
			name: "one-spans",
			internal: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(resource, []*data.InstrumentationLibrarySpans{
					data.NewInstrumentationLibrarySpans(data.NewInstrumentationLibrary(), []*data.Span{span1})}),
			}),
			oc: []consumerdata.TraceData{
				{
					Node:         ocNode,
					Resource:     ocResource,
					Spans:        []*octrace.Span{ocSpan1},
					SourceFormat: sourceFormat,
				},
			},
		},

		{
			name: "two-spans",
			internal: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(data.NewResource(), []*data.InstrumentationLibrarySpans{
					data.NewInstrumentationLibrarySpans(data.NewInstrumentationLibrary(), []*data.Span{span1, span2})}),
			}),
			oc: []consumerdata.TraceData{
				{
					Node:         ocNode,
					Resource:     &ocresource.Resource{},
					Spans:        []*octrace.Span{ocSpan1, ocSpan2},
					SourceFormat: sourceFormat,
				},
			},
		},

		{
			name: "two-spans-plus-one-separate",
			internal: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(resource, []*data.InstrumentationLibrarySpans{
					data.NewInstrumentationLibrarySpans(data.NewInstrumentationLibrary(), []*data.Span{span1, span2})}),
				resourceSpans3,
			}),
			oc: []consumerdata.TraceData{
				{
					Node:         ocNode,
					Resource:     ocResource,
					Spans:        []*octrace.Span{ocSpan1, ocSpan2},
					SourceFormat: sourceFormat,
				},
				{
					Node: ocNode,
					Resource: &ocresource.Resource{
						Type:   span3ResourceType,
						Labels: map[string]string{},
					},
					Spans:        []*octrace.Span{ocSpan3},
					SourceFormat: sourceFormat,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := TraceDataToOC(test.internal)
			assert.EqualValues(t, test.oc, got)
		})
	}
}
