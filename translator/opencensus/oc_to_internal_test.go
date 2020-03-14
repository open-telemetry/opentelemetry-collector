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

package opencensus

import (
	"strings"
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestOcNodeResourceToInternal(t *testing.T) {
	resource := ocNodeResourceToInternal(nil, nil)
	assert.EqualValues(t, data.NewResource(), resource)

	ocNode := &occommon.Node{}
	ocResource := &ocresource.Resource{}
	resource = ocNodeResourceToInternal(ocNode, ocResource)
	assert.EqualValues(t, data.NewResource(), resource)

	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	ocNode = &occommon.Node{
		Identifier: &occommon.ProcessIdentifier{
			HostName:       "host1",
			Pid:            123,
			StartTimestamp: ts,
		},
		LibraryInfo: &occommon.LibraryInfo{
			Language:           occommon.LibraryInfo_CPP,
			ExporterVersion:    "v1.2.0",
			CoreLibraryVersion: "v2.0.1",
		},
		ServiceInfo: &occommon.ServiceInfo{
			Name: "svcA",
		},
		Attributes: map[string]string{
			"node-attr": "val1",
		},
	}
	ocResource = &ocresource.Resource{
		Type: "good-resource",
		Labels: map[string]string{
			"resource-attr": "val2",
		},
	}
	resource = ocNodeResourceToInternal(ocNode, ocResource)

	expectedAttrs := data.AttributesMap{
		conventions.AttributeHostHostname:       data.NewAttributeValueString("host1"),
		conventions.OCAttributeProcessID:        data.NewAttributeValueInt(123),
		conventions.OCAttributeProcessStartTime: data.NewAttributeValueString("2020-02-11T20:26:00Z"),
		conventions.AttributeLibraryLanguage:    data.NewAttributeValueString("CPP"),
		conventions.OCAttributeExporterVersion:  data.NewAttributeValueString("v1.2.0"),
		conventions.AttributeLibraryVersion:     data.NewAttributeValueString("v2.0.1"),
		conventions.AttributeServiceName:        data.NewAttributeValueString("svcA"),
		"node-attr":                             data.NewAttributeValueString("val1"),
		conventions.OCAttributeResourceType:     data.NewAttributeValueString("good-resource"),
		"resource-attr":                         data.NewAttributeValueString("val2"),
	}

	assert.EqualValues(t, expectedAttrs, resource.Attributes())

	// Make sure hard-coded fields override same-name values in Attributes.
	// To do that add Attributes with same-name.
	for k := range expectedAttrs {
		// Set all except "attr1" which is not a hard-coded field to some bogus values.

		if strings.Index(k, "-attr") < 0 {
			ocNode.Attributes[k] = "this will be overridden 1"
		}
	}
	ocResource.Labels[conventions.OCAttributeResourceType] = "this will be overridden 2"

	// Convert again.
	resource = ocNodeResourceToInternal(ocNode, ocResource)

	// And verify that same-name attributes were ignored.
	assert.EqualValues(t, expectedAttrs, resource.Attributes())
}

func TestOcTraceStateToInternal(t *testing.T) {
	assert.EqualValues(t, "", ocTraceStateToInternal(nil))

	tracestate := &octrace.Span_Tracestate{
		Entries: []*octrace.Span_Tracestate_Entry{
			{
				Key:   "abc",
				Value: "def",
			},
		},
	}
	assert.EqualValues(t, "abc=def", ocTraceStateToInternal(tracestate))

	tracestate.Entries = append(tracestate.Entries,
		&octrace.Span_Tracestate_Entry{
			Key:   "123",
			Value: "4567",
		})
	assert.EqualValues(t, "abc=def,123=4567", ocTraceStateToInternal(tracestate))
}

func TestOcAttrsToInternal(t *testing.T) {
	attrs := ocAttrsToInternal(nil)
	assert.EqualValues(t, data.NewAttributes(nil, 0), attrs)

	ocAttrs := &octrace.Span_Attributes{}
	attrs = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t, data.NewAttributes(data.AttributesMap{}, 0), attrs)

	ocAttrs = &octrace.Span_Attributes{
		DroppedAttributesCount: 123,
	}
	attrs = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t, data.NewAttributes(data.AttributesMap{}, 123), attrs)

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap:           map[string]*octrace.AttributeValue{},
		DroppedAttributesCount: 234,
	}
	attrs = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t, data.NewAttributes(data.AttributesMap{}, 234), attrs)

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap: map[string]*octrace.AttributeValue{
			"abc": {
				Value: &octrace.AttributeValue_StringValue{StringValue: &octrace.TruncatableString{Value: "def"}},
			},
		},
		DroppedAttributesCount: 234,
	}
	attrs = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t,
		data.NewAttributes(
			data.AttributesMap{
				"abc": data.NewAttributeValueString("def"),
			},
			234),
		attrs)

	ocAttrs.AttributeMap["intval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_IntValue{IntValue: 345},
	}
	ocAttrs.AttributeMap["boolval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_BoolValue{BoolValue: true},
	}
	ocAttrs.AttributeMap["doubleval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_DoubleValue{DoubleValue: 4.5},
	}
	attrs = ocAttrsToInternal(ocAttrs)

	expected := data.NewAttributes(data.AttributesMap{
		"abc":       data.NewAttributeValueString("def"),
		"intval":    data.NewAttributeValueInt(345),
		"boolval":   data.NewAttributeValueBool(true),
		"doubleval": data.NewAttributeValueDouble(4.5),
	}, 234)
	assert.EqualValues(t, expected, attrs)
}

func TestOcSpanKindToInternal(t *testing.T) {
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
			got := ocSpanKindToInternal(test.ocKind, test.ocAttrs)
			assert.EqualValues(t, test.otlpKind, got, "Expected "+test.otlpKind.String()+", got "+got.String())
		})
	}
}

func TestOcToInternal(t *testing.T) {
	ocNode := &occommon.Node{Attributes: map[string]string{"nodeattr": "attrval123"}}
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
				4,
			),
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

	span3 := data.NewSpan()
	span3.SetName("operationD")
	span3.SetStartTime(internal.TimestampToUnixnano(timestampP))
	span3.SetEndTime(internal.TimestampToUnixnano(timestampP))

	internalResource := data.NewResource()
	internalResource.SetAttributes(
		map[string]data.AttributeValue{"nodeattr": data.NewAttributeValueString("attrval123")},
	)

	tests := []struct {
		name string
		oc   consumerdata.TraceData
		itd  data.TraceData
	}{
		{
			name: "empty",
			oc:   consumerdata.TraceData{},
			itd:  data.TraceData{},
		},

		{
			name: "nil-resource",
			oc:   consumerdata.TraceData{Node: ocNode},
			itd: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(internalResource, nil),
			}),
		},

		{
			name: "nil-node",
			oc:   consumerdata.TraceData{Resource: ocResource},
			itd: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(data.NewResource(), nil),
			}),
		},

		{
			name: "no-spans",
			oc:   consumerdata.TraceData{Node: ocNode, Resource: ocResource},
			itd: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(internalResource, nil),
			}),
		},

		{
			name: "one-spans",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1},
			},
			itd: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(internalResource, []*data.Span{span1}),
			}),
		},

		{
			name: "two-spans",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1, nil, ocSpan2},
			},
			itd: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(internalResource, []*data.Span{span1, span2}),
			}),
		},

		{
			name: "two-spans-plus-one-separate",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1, ocSpan2, ocSpan3},
			},
			itd: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(internalResource, []*data.Span{span1, span2}),
				data.NewResourceSpans(internalResource, []*data.Span{span3}),
			}),
		},

		{
			name: "two-spans-and-separate-in-the-middle",
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource,
				Spans:    []*octrace.Span{ocSpan1, ocSpan3, ocSpan2},
			},
			itd: data.NewTraceData([]*data.ResourceSpans{
				data.NewResourceSpans(internalResource, []*data.Span{span1, span2}),
				data.NewResourceSpans(internalResource, []*data.Span{span3}),
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ocToInternal(test.oc)
			assert.EqualValues(t, test.itd, got)
		})
	}
}
