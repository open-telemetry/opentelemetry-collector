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

package internaldata

import (
	"strconv"
	"testing"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

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

func TestInitAttributeMapFromOC(t *testing.T) {
	attrs := pdata.NewAttributeMap()
	initAttributeMapFromOC(nil, attrs)
	assert.EqualValues(t, pdata.NewAttributeMap(), attrs)
	assert.EqualValues(t, 0, ocAttrsToDroppedAttributes(nil))

	ocAttrs := &octrace.Span_Attributes{}
	attrs = pdata.NewAttributeMap()
	initAttributeMapFromOC(ocAttrs, attrs)
	assert.EqualValues(t, pdata.NewAttributeMap(), attrs)
	assert.EqualValues(t, 0, ocAttrsToDroppedAttributes(ocAttrs))

	ocAttrs = &octrace.Span_Attributes{
		DroppedAttributesCount: 123,
	}
	attrs = pdata.NewAttributeMap()
	initAttributeMapFromOC(ocAttrs, attrs)
	assert.EqualValues(t, pdata.NewAttributeMap(), attrs)
	assert.EqualValues(t, 123, ocAttrsToDroppedAttributes(ocAttrs))

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap:           map[string]*octrace.AttributeValue{},
		DroppedAttributesCount: 234,
	}
	attrs = pdata.NewAttributeMap()
	initAttributeMapFromOC(ocAttrs, attrs)
	assert.EqualValues(t, pdata.NewAttributeMap(), attrs)
	assert.EqualValues(t, 234, ocAttrsToDroppedAttributes(ocAttrs))

	ocAttrs = &octrace.Span_Attributes{
		AttributeMap: map[string]*octrace.AttributeValue{
			"abc": {
				Value: &octrace.AttributeValue_StringValue{StringValue: &octrace.TruncatableString{Value: "def"}},
			},
		},
		DroppedAttributesCount: 234,
	}
	attrs = pdata.NewAttributeMap()
	initAttributeMapFromOC(ocAttrs, attrs)
	assert.EqualValues(t,
		pdata.NewAttributeMap().InitFromMap(
			map[string]pdata.AttributeValue{
				"abc": pdata.NewAttributeValueString("def"),
			}),
		attrs)
	assert.EqualValues(t, 234, ocAttrsToDroppedAttributes(ocAttrs))

	ocAttrs.AttributeMap["intval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_IntValue{IntValue: 345},
	}
	ocAttrs.AttributeMap["boolval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_BoolValue{BoolValue: true},
	}
	ocAttrs.AttributeMap["doubleval"] = &octrace.AttributeValue{
		Value: &octrace.AttributeValue_DoubleValue{DoubleValue: 4.5},
	}
	attrs = pdata.NewAttributeMap()
	initAttributeMapFromOC(ocAttrs, attrs)

	expectedAttr := pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
		"abc":       pdata.NewAttributeValueString("def"),
		"intval":    pdata.NewAttributeValueInt(345),
		"boolval":   pdata.NewAttributeValueBool(true),
		"doubleval": pdata.NewAttributeValueDouble(4.5),
	})
	assert.EqualValues(t, expectedAttr.Sort(), attrs.Sort())
	assert.EqualValues(t, 234, ocAttrsToDroppedAttributes(ocAttrs))
}

func TestOcSpanKindToInternal(t *testing.T) {
	tests := []struct {
		ocAttrs  *octrace.Span_Attributes
		ocKind   octrace.Span_SpanKind
		otlpKind otlptrace.Span_SpanKind
	}{
		{
			ocKind:   octrace.Span_CLIENT,
			otlpKind: otlptrace.Span_SPAN_KIND_CLIENT,
		},
		{
			ocKind:   octrace.Span_SERVER,
			otlpKind: otlptrace.Span_SPAN_KIND_SERVER,
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
			otlpKind: otlptrace.Span_SPAN_KIND_CONSUMER,
		},
		{
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
			ocAttrs: &octrace.Span_Attributes{
				AttributeMap: map[string]*octrace.AttributeValue{
					"span.kind": {Value: &octrace.AttributeValue_StringValue{
						StringValue: &octrace.TruncatableString{Value: "producer"}}},
				},
			},
			otlpKind: otlptrace.Span_SPAN_KIND_PRODUCER,
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
			otlpKind: otlptrace.Span_SPAN_KIND_CLIENT,
		},
		{
			ocKind: octrace.Span_SPAN_KIND_UNSPECIFIED,
			ocAttrs: &octrace.Span_Attributes{
				AttributeMap: map[string]*octrace.AttributeValue{
					"span.kind": {Value: &octrace.AttributeValue_StringValue{
						StringValue: &octrace.TruncatableString{Value: "internal"}}},
				},
			},
			otlpKind: otlptrace.Span_SPAN_KIND_INTERNAL,
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

	startTime := timestamppb.New(testdata.TestSpanStartTime)
	eventTime := timestamppb.New(testdata.TestSpanEventTime)
	endTime := timestamppb.New(testdata.TestSpanEndTime)

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

	// TODO: Create another unit test fully covering ocSpanToInternal
	ocSpanZeroedParentID := proto.Clone(ocSpan1).(*octrace.Span)
	ocSpanZeroedParentID.ParentSpanId = []byte{0, 0, 0, 0, 0, 0, 0, 0}

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
		td   pdata.Traces
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
			name: "one-span-zeroed-parent-id",
			td:   testdata.GenerateTraceDataOneSpan(),
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource1,
				Spans:    []*octrace.Span{ocSpanZeroedParentID},
			},
		},

		{
			name: "one-span-one-nil",
			td:   testdata.GenerateTraceDataOneSpan(),
			oc: consumerdata.TraceData{
				Node:     ocNode,
				Resource: ocResource1,
				Spans:    []*octrace.Span{ocSpan1, nil},
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.EqualValues(t, test.td, OCToTraceData(test.oc))
		})
	}
}

func TestOcSameProcessAsParentSpanToInternal(t *testing.T) {
	span := pdata.NewSpan()
	ocSameProcessAsParentSpanToInternal(nil, span)
	assert.Equal(t, 0, span.Attributes().Len())

	ocSameProcessAsParentSpanToInternal(wrapperspb.Bool(false), span)
	assert.Equal(t, 1, span.Attributes().Len())
	v, ok := span.Attributes().Get(conventions.OCAttributeSameProcessAsParentSpan)
	assert.True(t, ok)
	assert.EqualValues(t, pdata.AttributeValueBOOL, v.Type())
	assert.False(t, v.BoolVal())

	ocSameProcessAsParentSpanToInternal(wrapperspb.Bool(true), span)
	assert.Equal(t, 1, span.Attributes().Len())
	v, ok = span.Attributes().Get(conventions.OCAttributeSameProcessAsParentSpan)
	assert.True(t, ok)
	assert.EqualValues(t, pdata.AttributeValueBOOL, v.Type())
	assert.True(t, v.BoolVal())
}

func BenchmarkSpansWithAttributesOCToInternal(b *testing.B) {
	ocSpan := generateSpanWithAttributes(15)

	ocTraceData := consumerdata.TraceData{
		Resource: generateOCTestResource(),
		Spans: []*octrace.Span{
			ocSpan,
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		OCToTraceData(ocTraceData)
	}
}

func BenchmarkSpansWithAttributesUnmarshal(b *testing.B) {
	ocSpan := generateSpanWithAttributes(15)

	bytes, err := proto.Marshal(ocSpan)
	if err != nil {
		b.Fail()
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		unmarshalOc := &octrace.Span{}
		if err := proto.Unmarshal(bytes, unmarshalOc); err != nil {
			b.Fail()
		}
		if len(unmarshalOc.Attributes.AttributeMap) != 15 {
			b.Fail()
		}
	}
}

func generateSpanWithAttributes(len int) *octrace.Span {
	startTime := timestamppb.New(testdata.TestSpanStartTime)
	endTime := timestamppb.New(testdata.TestSpanEndTime)
	ocSpan2 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationB"},
		StartTime: startTime,
		EndTime:   endTime,
		Attributes: &octrace.Span_Attributes{
			DroppedAttributesCount: 3,
		},
	}

	ocSpan2.Attributes.AttributeMap = make(map[string]*octrace.AttributeValue, len)
	ocAttr := ocSpan2.Attributes.AttributeMap
	for i := 0; i < len; i++ {
		ocAttr["span-link-attr_"+strconv.Itoa(i)] = &octrace.AttributeValue{
			Value: &octrace.AttributeValue_StringValue{
				StringValue: &octrace.TruncatableString{Value: "span-link-attr-val"},
			},
		}
	}
	return ocSpan2
}
