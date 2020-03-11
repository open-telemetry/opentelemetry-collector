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

package tracetranslator

import (
	"encoding/base64"
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
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestResourceSpansToTraceData(t *testing.T) {
	unixnanos := uint64(12578940000000012345)

	traceID, err := base64.StdEncoding.DecodeString("SEhaOVO7YSQ=")
	assert.NoError(t, err)

	spanID, err := base64.StdEncoding.DecodeString("QuHicGYRg4U=")
	assert.NoError(t, err)

	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	otlpSpan1 := &otlptrace.Span{
		TraceId:           traceID,
		SpanId:            spanID,
		Name:              "operationB",
		Kind:              otlptrace.Span_SERVER,
		StartTimeUnixnano: unixnanos,
		EndTimeUnixnano:   unixnanos,
		Events: []*otlptrace.Span_Event{
			{
				TimeUnixnano: unixnanos,
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
		Links: []*otlptrace.Span_Link{
			{
				TraceId: traceID,
				SpanId:  spanID,
			},
		},
		DroppedAttributesCount: 1,
		DroppedEventsCount:     2,
		Status:                 &otlptrace.Status{Message: "status-cancelled", Code: otlptrace.Status_Cancelled},
		Tracestate:             "a=text,b=123",
	}

	otlpSpan2 := &otlptrace.Span{
		Name:              "operationC",
		StartTimeUnixnano: unixnanos,
		EndTimeUnixnano:   unixnanos,
		Kind:              otlptrace.Span_CONSUMER,
		Links:             []*otlptrace.Span_Link{{}},
		DroppedLinksCount: 1,
	}

	otlpAttributes := []*otlpcommon.AttributeKeyValue{
		{
			Key:         conventions.OCAttributeResourceType,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "good-resource",
		},
		{
			Key:         conventions.OCAttributeProcessStartTime,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "2020-02-11T20:26:00Z",
		},
		{
			Key:         conventions.AttributeHostHostname,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "host1",
		},
		{
			Key:         conventions.OCAttributeProcessID,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "123",
		},
		{
			Key:         conventions.AttributeLibraryVersion,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "v2.0.1",
		},
		{
			Key:         conventions.OCAttributeExporterVersion,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "v1.2.0",
		},
		{
			Key:         conventions.AttributeLibraryLanguage,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "CPP",
		},
		{
			Key:         "str1",
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "text",
		},
		{
			Key:      "int2",
			Type:     otlpcommon.AttributeKeyValue_INT,
			IntValue: 123,
		},
	}

	ocSpan1 := &octrace.Span{
		TraceId:   traceID,
		SpanId:    spanID,
		Name:      &octrace.TruncatableString{Value: "operationB"},
		Kind:      octrace.Span_SERVER,
		StartTime: internal.UnixnanoToTimestamp(data.TimestampUnixNano(unixnanos)),
		EndTime:   internal.UnixnanoToTimestamp(data.TimestampUnixNano(unixnanos)),
		TimeEvents: &octrace.Span_TimeEvents{
			TimeEvent: []*octrace.Span_TimeEvent{
				{
					Time: internal.UnixnanoToTimestamp(data.TimestampUnixNano(unixnanos)),
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
			DroppedMessageEventsCount: 2,
		},
		Links: &octrace.Span_Links{
			Link: []*octrace.Span_Link{
				{
					TraceId: traceID,
					SpanId:  spanID,
				},
			},
		},
		Attributes: &octrace.Span_Attributes{
			DroppedAttributesCount: 1,
		},
		Status: &octrace.Status{Message: "status-cancelled", Code: 1},
		Tracestate: &octrace.Span_Tracestate{
			Entries: []*octrace.Span_Tracestate_Entry{
				{
					Key:   "a",
					Value: "text",
				},
				{
					Key:   "b",
					Value: "123",
				},
			},
		},
	}

	ocSpan2 := &octrace.Span{
		Name:      &octrace.TruncatableString{Value: "operationC"},
		Kind:      octrace.Span_SPAN_KIND_UNSPECIFIED,
		StartTime: internal.UnixnanoToTimestamp(data.TimestampUnixNano(unixnanos)),
		EndTime:   internal.UnixnanoToTimestamp(data.TimestampUnixNano(unixnanos)),
		Attributes: &octrace.Span_Attributes{
			AttributeMap: map[string]*octrace.AttributeValue{
				TagSpanKind: {
					Value: &octrace.AttributeValue_StringValue{
						StringValue: &octrace.TruncatableString{
							Value: string(OpenTracingSpanKindConsumer),
						},
					},
				},
			},
			DroppedAttributesCount: 0,
		},
		Links: &octrace.Span_Links{
			Link:              []*octrace.Span_Link{{}},
			DroppedLinksCount: 1,
		},
	}

	ocAttributes := map[string]string{
		"str1": "text",
		"int2": "123",
	}

	tests := []struct {
		name string
		oc   consumerdata.TraceData
		otlp otlptrace.ResourceSpans
	}{
		{
			name: "empty",
			otlp: otlptrace.ResourceSpans{},
			oc: consumerdata.TraceData{
				Spans:        nil,
				SourceFormat: sourceFormat,
			},
		},

		{
			name: "nil-resource",
			otlp: otlptrace.ResourceSpans{
				Resource: nil,
			},
			oc: consumerdata.TraceData{
				Node:         nil,
				Resource:     nil,
				Spans:        nil,
				SourceFormat: sourceFormat,
			},
		},

		{
			name: "no-spans",
			otlp: otlptrace.ResourceSpans{
				Resource: &otlpresource.Resource{},
				Spans:    []*otlptrace.Span{},
			},
			oc: consumerdata.TraceData{
				Node:         &occommon.Node{},
				Resource:     &ocresource.Resource{},
				Spans:        nil,
				SourceFormat: sourceFormat,
			},
		},

		{
			name: "one-span",
			otlp: otlptrace.ResourceSpans{
				Resource: &otlpresource.Resource{
					Attributes: otlpAttributes,
				},
				Spans: []*otlptrace.Span{otlpSpan1},
			},
			oc: consumerdata.TraceData{
				Node: &occommon.Node{
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
				},
				Resource: &ocresource.Resource{
					Type:   "good-resource",
					Labels: ocAttributes,
				},
				Spans:        []*octrace.Span{ocSpan1},
				SourceFormat: sourceFormat,
			},
		},

		{
			name: "two-spans",
			otlp: otlptrace.ResourceSpans{
				Resource: &otlpresource.Resource{},
				Spans:    []*otlptrace.Span{otlpSpan1, otlpSpan2},
			},
			oc: consumerdata.TraceData{
				Node:         &occommon.Node{},
				Resource:     &ocresource.Resource{},
				Spans:        []*octrace.Span{ocSpan1, ocSpan2},
				SourceFormat: sourceFormat,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResourceSpansToTraceData(&test.otlp)
			assert.EqualValues(t, test.oc, got)
		})
	}
}
