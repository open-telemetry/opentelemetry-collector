// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spandata

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/tracestate"

	"github.com/open-telemetry/opentelemetry-collector/internal"
)

func TestProtoSpanToOCSpanData_endToEnd(t *testing.T) {
	// The goal of this test is to ensure that each
	// tracepb.Span is transformed to its *trace.SpanData correctly!

	endTime := time.Now().Round(time.Second)
	startTime := endTime.Add(-90 * time.Second)

	protoSpan := &tracepb.Span{
		TraceId:      []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
		SpanId:       []byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8},
		ParentSpanId: []byte{0xEF, 0xEE, 0xED, 0xEC, 0xEB, 0xEA, 0xE9, 0xE8},
		Name:         &tracepb.TruncatableString{Value: "End-To-End Here"},
		Kind:         tracepb.Span_SERVER,
		StartTime:    internal.TimeToTimestamp(startTime),
		EndTime:      internal.TimeToTimestamp(endTime),
		Status: &tracepb.Status{
			Code:    13,
			Message: "This is not a drill!",
		},
		SameProcessAsParentSpan: &wrappers.BoolValue{Value: false},
		TimeEvents: &tracepb.Span_TimeEvents{
			TimeEvent: []*tracepb.Span_TimeEvent{
				{
					Time: internal.TimeToTimestamp(startTime),
					Value: &tracepb.Span_TimeEvent_MessageEvent_{
						MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
							Type: tracepb.Span_TimeEvent_MessageEvent_SENT, UncompressedSize: 1024, CompressedSize: 512,
						},
					},
				},
				{
					Time: internal.TimeToTimestamp(endTime),
					Value: &tracepb.Span_TimeEvent_MessageEvent_{
						MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
							Type: tracepb.Span_TimeEvent_MessageEvent_RECEIVED, UncompressedSize: 1024, CompressedSize: 1000,
						},
					},
				},
			},
		},
		Links: &tracepb.Span_Links{
			Link: []*tracepb.Span_Link{
				{
					TraceId: []byte{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF},
					SpanId:  []byte{0xB0, 0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7},
					Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
				},
				{
					TraceId: []byte{0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF},
					SpanId:  []byte{0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7},
					Type:    tracepb.Span_Link_CHILD_LINKED_SPAN,
				},
			},
		},
		Tracestate: &tracepb.Span_Tracestate{
			Entries: []*tracepb.Span_Tracestate_Entry{
				{Key: "foo", Value: "bar"},
				{Key: "a", Value: "b"},
			},
		},
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: map[string]*tracepb.AttributeValue{
				"cache_hit":  {Value: &tracepb.AttributeValue_BoolValue{BoolValue: true}},
				"timeout_ns": {Value: &tracepb.AttributeValue_IntValue{IntValue: 12e9}},
				"ping_count": {Value: &tracepb.AttributeValue_IntValue{IntValue: 25}},
				"agent": {Value: &tracepb.AttributeValue_StringValue{
					StringValue: &tracepb.TruncatableString{Value: "ocagent"},
				}},
			},
		},
	}

	gotOCSpanData, err := ProtoSpanToOCSpanData(protoSpan)
	if err != nil {
		t.Fatalf("Failed to convert from ProtoSpan to OCSpanData: %v", err)
	}

	ocTracestate, err := tracestate.New(new(tracestate.Tracestate), tracestate.Entry{Key: "foo", Value: "bar"},
		tracestate.Entry{Key: "a", Value: "b"})
	if err != nil || ocTracestate == nil {
		t.Fatalf("Failed to create ocTracestate: %v", err)
	}

	wantOCSpanData := &trace.SpanData{
		SpanContext: trace.SpanContext{
			TraceID:    trace.TraceID{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
			SpanID:     trace.SpanID{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8},
			Tracestate: ocTracestate,
		},
		SpanKind:     trace.SpanKindServer,
		ParentSpanID: trace.SpanID{0xEF, 0xEE, 0xED, 0xEC, 0xEB, 0xEA, 0xE9, 0xE8},
		Name:         "End-To-End Here",
		StartTime:    startTime,
		EndTime:      endTime,
		MessageEvents: []trace.MessageEvent{
			{Time: startTime, EventType: trace.MessageEventTypeSent, UncompressedByteSize: 1024, CompressedByteSize: 512},
			{Time: endTime, EventType: trace.MessageEventTypeRecv, UncompressedByteSize: 1024, CompressedByteSize: 1000},
		},
		Links: []trace.Link{
			{
				TraceID: trace.TraceID{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF},
				SpanID:  trace.SpanID{0xB0, 0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7},
				Type:    trace.LinkTypeParent,
			},
			{
				TraceID: trace.TraceID{0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF},
				SpanID:  trace.SpanID{0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7},
				Type:    trace.LinkTypeChild,
			},
		},
		Status: trace.Status{
			Code:    trace.StatusCodeInternal,
			Message: "This is not a drill!",
		},
		HasRemoteParent: true,
		Attributes: map[string]interface{}{
			"timeout_ns": int64(12e9),
			"agent":      "ocagent",
			"cache_hit":  true,
			"ping_count": int(25), // Should be transformed into int64
		},
	}

	if g, w := gotOCSpanData, wantOCSpanData; !reflect.DeepEqual(g, w) {
		// As a last resort now compare by bytes here since reflect.DeepEqual might have been
		// tripped out perhaps due to pointer differences(although this shouldn't hinder it).
		gBlob, _ := json.MarshalIndent(g, "", "  ")
		wBlob, _ := json.MarshalIndent(w, "", "  ")
		if !bytes.Equal(wBlob, gBlob) {
			t.Fatalf("End-to-end transformed span\n\tGot  %+v\n\tWant %+v", g, w)
		}
	}
}
