// Copyright 2018, OpenCensus Authors
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

package main

import (
	"reflect"
	"testing"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/traceproto"
	"github.com/golang/protobuf/ptypes"
	"go.opencensus.io/trace"
)

func TestProtoToSpanData(t *testing.T) {
	start := time.Now()
	end := time.Now()

	startProto, _ := ptypes.TimestampProto(start)
	endProto, _ := ptypes.TimestampProto(end)

	tests := []struct {
		name string
		s    *tracepb.Span
		want *trace.SpanData
	}{
		{
			name: "zero",
			s:    &tracepb.Span{},
			want: &trace.SpanData{
				StartTime: time.Unix(0, 0).UTC(),
				EndTime:   time.Unix(0, 0).UTC(),
			},
		},
		{
			name: "non-zero",
			s: &tracepb.Span{
				TraceId:      []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
				SpanId:       []byte{0, 1, 2, 3, 4, 5, 6, 7},
				ParentSpanId: []byte{8, 9, 10, 11, 12, 13, 14, 15},
				Name: &tracepb.TruncatableString{
					Value: "name",
				},
				Kind:      tracepb.Span_SERVER,
				StartTime: startProto,
				EndTime:   endProto,

				Status: &tracepb.Status{
					Code:    2,
					Message: "unknown",
				},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key3": {
							Value: &tracepb.AttributeValue_BoolValue{
								BoolValue: true,
							},
						},
					},
				},
				Links: &tracepb.Span_Links{
					Link: []*tracepb.Span_Link{
						{
							TraceId: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
							SpanId:  []byte{0, 1, 2, 3, 4, 5, 6, 7},
							Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
							Attributes: &tracepb.Span_Attributes{
								AttributeMap: map[string]*tracepb.AttributeValue{
									"key1": {
										Value: &tracepb.AttributeValue_StringValue{
											StringValue: &tracepb.TruncatableString{
												Value: "value1",
											},
										},
									},
								},
							},
						},
						{
							TraceId: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
							SpanId:  []byte{0, 1, 2, 3, 4, 5, 6, 7},
							Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
							Attributes: &tracepb.Span_Attributes{
								AttributeMap: map[string]*tracepb.AttributeValue{
									"key2": {
										Value: &tracepb.AttributeValue_IntValue{
											IntValue: 5,
										},
									},
								},
							},
						},
					},
				},
			},
			want: &trace.SpanData{
				SpanContext: trace.SpanContext{
					TraceID: trace.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
					SpanID:  trace.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
				},
				ParentSpanID: trace.SpanID{8, 9, 10, 11, 12, 13, 14, 15},
				Name:         "name",
				SpanKind:     trace.SpanKindServer,
				StartTime:    start.UTC(),
				EndTime:      end.UTC(),
				Status: trace.Status{
					Code:    2,
					Message: "unknown",
				},
				Attributes: map[string]interface{}{
					"key3": true,
				},
				Links: []trace.Link{
					{
						TraceID: trace.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
						SpanID:  trace.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
						Type:    trace.LinkTypeParent,
						Attributes: map[string]interface{}{
							"key1": "value1",
						},
					},
					{
						TraceID: trace.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
						SpanID:  trace.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
						Type:    trace.LinkTypeParent,
						Attributes: map[string]interface{}{
							"key2": int64(5),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := protoToSpanData(tt.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("protoToSpanData() = %v, want %v", got, tt.want)
			}
		})
	}
}
