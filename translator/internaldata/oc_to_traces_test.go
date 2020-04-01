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

package internaldata

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
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestOcNodeResourceToInternal(t *testing.T) {
	resource := data.NewResource()
	ocNodeResourceToInternal(nil, nil, resource)
	assert.EqualValues(t, true, resource.IsNil())

	ocNode := &occommon.Node{}
	ocResource := &ocresource.Resource{}
	ocNodeResourceToInternal(ocNode, ocResource, resource)
	assert.EqualValues(t, true, resource.IsNil())

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
	expectedAttrs := data.NewAttributeMap(map[string]data.AttributeValue{
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
	})

	ocNodeResourceToInternal(ocNode, ocResource, resource)
	assert.EqualValues(t, expectedAttrs.Sort(), resource.Attributes().Sort())

	// Make sure hard-coded fields override same-name values in Attributes.
	// To do that add Attributes with same-name.
	for i := 0; i < expectedAttrs.Len(); i++ {
		// Set all except "attr1" which is not a hard-coded field to some bogus values.

		if !strings.Contains(expectedAttrs.GetAttribute(i).Key(), "-attr") {
			ocNode.Attributes[expectedAttrs.GetAttribute(i).Key()] = "this will be overridden 1"
		}
	}
	ocResource.Labels[conventions.OCAttributeResourceType] = "this will be overridden 2"

	// Convert again.
	resource = data.NewResource()
	ocNodeResourceToInternal(ocNode, ocResource, resource)
	// And verify that same-name attributes were ignored.
	assert.EqualValues(t, expectedAttrs.Sort(), resource.Attributes().Sort())
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
	attrs, droppedAttr := ocAttrsToInternal(nil)
	assert.EqualValues(t, data.NewAttributeMap(nil), attrs)
	assert.EqualValues(t, 0, droppedAttr)

	ocAttrs := &octrace.Span_Attributes{}
	attrs, droppedAttr = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t, data.NewAttributeMap(data.AttributesMap{}), attrs)
	assert.EqualValues(t, 0, droppedAttr)

	ocAttrs = &octrace.Span_Attributes{
		DroppedAttributesCount: 123,
	}
	attrs, droppedAttr = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t, data.NewAttributeMap(data.AttributesMap{}), attrs)
	assert.EqualValues(t, 123, droppedAttr)

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap:           map[string]*octrace.AttributeValue{},
		DroppedAttributesCount: 234,
	}
	attrs, droppedAttr = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t, data.NewAttributeMap(data.AttributesMap{}), attrs)
	assert.EqualValues(t, 234, droppedAttr)

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap: map[string]*octrace.AttributeValue{
			"abc": {
				Value: &octrace.AttributeValue_StringValue{StringValue: &octrace.TruncatableString{Value: "def"}},
			},
		},
		DroppedAttributesCount: 234,
	}
	attrs, droppedAttr = ocAttrsToInternal(ocAttrs)
	assert.EqualValues(t,
		data.NewAttributeMap(
			data.AttributesMap{
				"abc": data.NewAttributeValueString("def"),
			}),
		attrs)
	assert.EqualValues(t, 234, droppedAttr)

	ocAttrs.AttributeMap["intval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_IntValue{IntValue: 345},
	}
	ocAttrs.AttributeMap["boolval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_BoolValue{BoolValue: true},
	}
	ocAttrs.AttributeMap["doubleval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_DoubleValue{DoubleValue: 4.5},
	}
	attrs, droppedAttr = ocAttrsToInternal(ocAttrs)

	expectedAttr := data.NewAttributeMap(data.AttributesMap{
		"abc":       data.NewAttributeValueString("def"),
		"intval":    data.NewAttributeValueInt(345),
		"boolval":   data.NewAttributeValueBool(true),
		"doubleval": data.NewAttributeValueDouble(4.5),
	})
	assert.EqualValues(t, expectedAttr.Sort(), attrs.Sort())
	assert.EqualValues(t, 234, droppedAttr)
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
	ocNode := &occommon.Node{}
	ocResource1 := &ocresource.Resource{Labels: map[string]string{"resource-attr": "resource-attr-val-1"}}
	ocResource2 := &ocresource.Resource{Labels: map[string]string{"resource-attr": "resource-attr-val-2"}}

	startTime, err := ptypes.TimestampProto(testdata.TestSpanStartTime)
	assert.NoError(t, err)
	eventTime, err := ptypes.TimestampProto(testdata.TestSpanEventTime)
	assert.NoError(t, err)
	endTime, err := ptypes.TimestampProto(testdata.TestSpanEndTime)
	assert.NoError(t, err)

	ocSpan1 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationA"},
		StartTime: startTime,
		EndTime:   endTime,
		TimeEvents: &octrace.Span_TimeEvents{
			TimeEvent: []*octrace.Span_TimeEvent{
				{
					Time: eventTime,
					Value: &octrace.Span_TimeEvent_Annotation_{
						Annotation: &octrace.Span_TimeEvent_Annotation{
							Description: &octrace.TruncatableString{Value: "event-with-attr"},
							Attributes: &octrace.Span_Attributes{
								AttributeMap: map[string]*octrace.AttributeValue{
									"span-event-attr": {
										Value: &octrace.AttributeValue_StringValue{
											StringValue: &octrace.TruncatableString{Value: "span-event-attr-val"},
										},
									},
								},
								DroppedAttributesCount: 2,
							},
						},
					},
				},
				{
					Time: eventTime,
					Value: &octrace.Span_TimeEvent_Annotation_{
						Annotation: &octrace.Span_TimeEvent_Annotation{
							Description: &octrace.TruncatableString{Value: "event"},
							Attributes: &octrace.Span_Attributes{
								DroppedAttributesCount: 2,
							},
						},
					},
				},
			},
			DroppedAnnotationsCount: 1,
		},
		Attributes: &octrace.Span_Attributes{
			DroppedAttributesCount: 1,
		},
		Status: &octrace.Status{Message: "status-cancelled", Code: 1},
	}

	ocSpan2 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationB"},
		StartTime: startTime,
		EndTime:   endTime,
		Links: &octrace.Span_Links{
			Link: []*octrace.Span_Link{
				{
					Attributes: &octrace.Span_Attributes{
						AttributeMap: map[string]*octrace.AttributeValue{
							"span-link-attr": {
								Value: &octrace.AttributeValue_StringValue{
									StringValue: &octrace.TruncatableString{Value: "span-link-attr-val"},
								},
							},
						},
						DroppedAttributesCount: 4,
					},
				},
				{
					Attributes: &octrace.Span_Attributes{
						DroppedAttributesCount: 4,
					},
				},
			},
			DroppedLinksCount: 3,
		},
	}

	ocSpan3 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationC"},
		StartTime: startTime,
		EndTime:   endTime,
		Resource:  ocResource2,
		Attributes: &octrace.Span_Attributes{
			AttributeMap: map[string]*octrace.AttributeValue{
				"span-attr": {
					Value: &octrace.AttributeValue_StringValue{
						StringValue: &octrace.TruncatableString{Value: "span-attr-val"},
					},
				},
			},
			DroppedAttributesCount: 5,
		},
	}

	tests := []struct {
		name string
		td   data.TraceData
		oc   consumerdata.TraceData
	}{
		{
			name: "empty",
			td:   testdata.GenerateTraceDataEmpty(),
			oc:   consumerdata.TraceData{},
		},

		{
			name: "one-empty-resource-spans",
			td:   testdata.GenerateTraceDataOneEmptyResourceSpans(),
			oc:   consumerdata.TraceData{Node: ocNode},
		},

		{
			name: "no-libraries",
			td:   testdata.GenerateTraceDataNoLibraries(),
			oc:   consumerdata.TraceData{Resource: ocResource1},
		},

		{
			name: "one-span-no-resource",
			td:   testdata.GenerateTraceDataOneSpanNoResource(),
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: &ocresource.Resource{},
				Spans:    []*octrace.Span{ocSpan1},
			},
		},

		{

			name: "one-span",
			td:   testdata.GenerateTraceDataOneSpan(),
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource1,
				Spans:    []*octrace.Span{ocSpan1},
			},
		},

		{
			name: "two-spans-same-resource",
			td:   testdata.GenerateTraceDataTwoSpansSameResource(),
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource1,
				Spans:    []*octrace.Span{ocSpan1, nil, ocSpan2},
			},
		},

		{
			name: "two-spans-same-resource-one-different",
			td:   testdata.GenerateTraceDataTwoSpansSameResourceOneDifferent(),
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource1,
				Spans:    []*octrace.Span{ocSpan1, ocSpan2, ocSpan3},
			},
		},

		{
			name: "two-spans-and-separate-in-the-middle",
			td:   testdata.GenerateTraceDataTwoSpansSameResourceOneDifferent(),
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource1,
				Spans:    []*octrace.Span{ocSpan1, ocSpan3, ocSpan2},
			},
		},
	}

	// Equal number of tests even though there is an extra test "two-spans-and-separate-in-the-middle"
	// but the test case GenerateTraceDataNoSpans it is impossible to get from OC data.
	assert.EqualValues(t, testdata.NumTraceTests, len(tests))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.EqualValues(t, test.td, OCToTraceData(test.oc))
		})
	}
}
