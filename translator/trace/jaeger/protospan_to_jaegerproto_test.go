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

package jaeger

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	jaeger "github.com/jaegertracing/jaeger/model"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/internal/testutils"
	tracetranslator "github.com/open-telemetry/opentelemetry-service/translator/trace"
)

func TestNilOCProtoNodeToJaegerProto(t *testing.T) {
	nilNodeBatch := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				TraceId: []byte("0123456789abcdef"),
				SpanId:  []byte("01234567"),
			},
		},
	}
	got, err := OCProtoToJaegerProto(nilNodeBatch)
	if err != nil {
		t.Fatalf("Failed to translate OC batch to Jaeger Proto: %v", err)
	}
	if got.Process == nil {
		t.Fatalf("Jaeger requires a non-nil Process field")
	}
}

func TestOCProtoToJaegerProto(t *testing.T) {
	const numOfFiles = 2
	for i := 0; i < numOfFiles; i++ {
		ocBatch := ocBatches[i]

		gotJBatch, err := OCProtoToJaegerProto(ocBatch)
		if err != nil {
			t.Errorf("Failed to translate OC batch to Jaeger Proto: %v", err)
			continue
		}

		wantSpanCount, gotSpanCount := len(ocBatch.Spans), len(gotJBatch.Spans)
		if wantSpanCount != gotSpanCount {
			t.Errorf("Different number of spans in the batches on pass #%d (want %d, got %d)", i, wantSpanCount, gotSpanCount)
			continue
		}

		// Jaeger binary tags do not round trip from Jaeger -> OCProto -> Jaeger.
		// For tests use data without binary tags.
		protoFile := fmt.Sprintf("./testdata/proto_batch_no_binary_tags_%02d.json", i+1)
		wantJBatch := &jaeger.Batch{}
		if err := loadFromJSON(protoFile, wantJBatch); err != nil {
			t.Errorf("Failed to load Jaeger Proto from %q: %v", protoFile, err)
			continue
		}

		// Sort tags to help with comparison, not only for jaeger.Process but also
		// on each span.
		sort.Slice(gotJBatch.Process.Tags, func(i, j int) bool {
			return gotJBatch.Process.Tags[i].Key < gotJBatch.Process.Tags[j].Key
		})
		sort.Slice(wantJBatch.Process.Tags, func(i, j int) bool {
			return wantJBatch.Process.Tags[i].Key < wantJBatch.Process.Tags[j].Key
		})
		var jSpans []*jaeger.Span
		jSpans = append(jSpans, gotJBatch.Spans...)
		jSpans = append(jSpans, wantJBatch.Spans...)
		for _, jSpan := range jSpans {
			sort.Slice(jSpan.Tags, func(i, j int) bool {
				return jSpan.Tags[i].Key < jSpan.Tags[j].Key
			})
		}
		for _, jSpan := range jSpans {
			for _, jSpanLog := range jSpan.Logs {
				sort.Slice(jSpanLog.Fields, func(i, j int) bool {
					return jSpanLog.Fields[i].Key < jSpanLog.Fields[j].Key
				})
			}
		}

		gjson, _ := json.Marshal(gotJBatch)
		wjson, _ := json.Marshal(wantJBatch)
		gjsonStr := testutils.GenerateNormalizedJSON(string(gjson))
		wjsonStr := testutils.GenerateNormalizedJSON(string(wjson))
		if gjsonStr != wjsonStr {
			t.Errorf("OC Proto to Jaeger Proto failed.\nGot:\n%s\nWant:\n%s\n", gjsonStr, wjsonStr)
		}
	}
}

func TestOCStatusToJaegerProtoTags(t *testing.T) {

	type test struct {
		haveAttributes *tracepb.Span_Attributes
		haveStatus     *tracepb.Status
		wantTags       []jaeger.KeyValue
	}

	cases := []test{
		// only status.code
		{
			haveAttributes: nil,
			haveStatus: &tracepb.Status{
				Code: 10,
			},
			wantTags: []jaeger.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(10),
					VType:  jaeger.ValueType_INT64,
				},
			},
		},
		// only status.message
		{
			haveAttributes: nil,
			haveStatus: &tracepb.Status{
				Message: "Forbidden",
			},
			wantTags: []jaeger.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(0),
					VType:  jaeger.ValueType_INT64,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  "Forbidden",
					VType: jaeger.ValueType_STRING,
				},
			},
		},
		// both status.code and status.message
		{
			haveAttributes: nil,
			haveStatus: &tracepb.Status{
				Code:    12,
				Message: "Forbidden",
			},
			wantTags: []jaeger.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(12),
					VType:  jaeger.ValueType_INT64,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  "Forbidden",
					VType: jaeger.ValueType_STRING,
				},
			},
		},

		// status and existing tags
		{
			haveStatus: &tracepb.Status{
				Code:    404,
				Message: "NotFound",
			},
			haveAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"status.code": {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 13,
						},
					},
					"status.message": {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "Error"},
						},
					},
				},
			},
			wantTags: []jaeger.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(13),
					VType:  jaeger.ValueType_INT64,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  "Error",
					VType: jaeger.ValueType_STRING,
				},
			},
		},

		// partial existing tag

		{
			haveStatus: &tracepb.Status{
				Code:    404,
				Message: "NotFound",
			},
			haveAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"status.code": {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 13,
						},
					},
				},
			},
			wantTags: []jaeger.KeyValue{
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(13),
					VType:  jaeger.ValueType_INT64,
				},
			},
		},

		{
			haveStatus: &tracepb.Status{
				Code:    404,
				Message: "NotFound",
			},
			haveAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"status.message": {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "Error"},
						},
					},
				},
			},
			wantTags: []jaeger.KeyValue{
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  "Error",
					VType: jaeger.ValueType_STRING,
				},
			},
		},
		// both status and tags
		{
			haveStatus: &tracepb.Status{
				Code:    13,
				Message: "Forbidden",
			},
			haveAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"http.status_code": {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					"http.status_message": {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantTags: []jaeger.KeyValue{
				{
					Key:    tracetranslator.TagHTTPStatusCode,
					VInt64: int64(404),
					VType:  jaeger.ValueType_INT64,
				},
				{
					Key:   tracetranslator.TagHTTPStatusMsg,
					VStr:  "NotFound",
					VType: jaeger.ValueType_STRING,
				},
				{
					Key:    tracetranslator.TagStatusCode,
					VInt64: int64(13),
					VType:  jaeger.ValueType_INT64,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  "Forbidden",
					VType: jaeger.ValueType_STRING,
				},
			},
		},
	}

	fakeTraceID := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	fakeSpanID := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	for i, c := range cases {
		gb, err := OCProtoToJaegerProto(consumerdata.TraceData{
			Spans: []*tracepb.Span{{
				TraceId:    fakeTraceID,
				SpanId:     fakeSpanID,
				Status:     c.haveStatus,
				Attributes: c.haveAttributes,
			}},
		})

		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		gs := gb.Spans[0]
		sort.Slice(gs.Tags, func(i, j int) bool {
			return gs.Tags[i].Key < gs.Tags[j].Key
		})
		if !reflect.DeepEqual(c.wantTags, gs.Tags) {
			t.Fatalf("%d: Unsuccessful conversion\nGot:\n\t%v\nWant:\n\t%v", i, gs.Tags, c.wantTags)
		}
	}
}

// ocBatches has the OpenCensus proto batches used in the test. They are hard coded because
// structs like tracepb.AttributeMap cannot be read from JSON.
var ocBatches = []consumerdata.TraceData{
	{
		Node: &commonpb.Node{
			Identifier: &commonpb.ProcessIdentifier{
				HostName:       "api246-sjc1",
				Pid:            13,
				StartTimestamp: &timestamp.Timestamp{Seconds: 1485467190, Nanos: 639875000},
			},
			LibraryInfo: &commonpb.LibraryInfo{ExporterVersion: "someVersion"},
			ServiceInfo: &commonpb.ServiceInfo{Name: "api"},
			Attributes: map[string]string{
				"a.binary": "AQIDBAMCAQ==",
				"a.bool":   "true",
				"a.double": "1234.56789",
				"a.long":   "123456789",
				"ip":       "10.53.69.61",
			},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
				SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
				ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x68, 0xC4, 0xE3},
				Name:         &tracepb.TruncatableString{Value: "get"},
				Kind:         tracepb.Span_SERVER,
				StartTime:    &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
				EndTime:      &timestamp.Timestamp{Seconds: 1485467191, Nanos: 662813000},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"http.url": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "http://127.0.0.1:15598/client_transactions"}},
						},
						"peer.ipv4": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 3224716605},
						},
						"peer.port": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 53931},
						},
						"peer.service": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "rtapi"}},
						},
						"someBool": {
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
						"someDouble": {
							Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 129.8},
						},
					},
				},
				TimeEvents: &tracepb.Span_TimeEvents{
					TimeEvent: []*tracepb.Span_TimeEvent{
						{
							Time: &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639874000},
							Value: &tracepb.Span_TimeEvent_MessageEvent_{
								MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
									Type: tracepb.Span_TimeEvent_MessageEvent_SENT, UncompressedSize: 1024, CompressedSize: 512,
								},
							},
						},
						{
							Time: &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
							Value: &tracepb.Span_TimeEvent_Annotation_{
								Annotation: &tracepb.Span_TimeEvent_Annotation{
									Attributes: &tracepb.Span_Attributes{
										AttributeMap: map[string]*tracepb.AttributeValue{
											"key1": {
												Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "value1"}},
											},
										},
									},
								},
							},
						},
						{
							Time: &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
							Value: &tracepb.Span_TimeEvent_Annotation_{
								Annotation: &tracepb.Span_TimeEvent_Annotation{
									Description: &tracepb.TruncatableString{Value: "annotation description"},
									Attributes: &tracepb.Span_Attributes{
										AttributeMap: map[string]*tracepb.AttributeValue{
											"event": {
												Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "nothing"}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
	{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "api"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
				SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
				ParentSpanId: nil,
				Name:         &tracepb.TruncatableString{Value: "get"},
				Kind:         tracepb.Span_SERVER,
				StartTime:    &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
				EndTime:      &timestamp.Timestamp{Seconds: 1485467191, Nanos: 662813000},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"peer.service": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "AAAAAAAAMDk="}},
						},
					},
				},
			},
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
				SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x99},
				ParentSpanId: []byte{},
				Name:         &tracepb.TruncatableString{Value: "get"},
				Kind:         tracepb.Span_SERVER,
				StartTime:    &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
				EndTime:      &timestamp.Timestamp{Seconds: 1485467191, Nanos: 662813000},
				Links: &tracepb.Span_Links{
					Link: []*tracepb.Span_Link{
						{
							TraceId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
							SpanId:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
							Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
						},
						{
							TraceId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
							SpanId:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x68, 0xC4, 0xE3},
						},
					},
				},
			},
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
				SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x90},
				ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
				Name:         &tracepb.TruncatableString{Value: "get2"},
				StartTime:    &timestamp.Timestamp{Seconds: 1485467192, Nanos: 639875000},
				EndTime:      &timestamp.Timestamp{Seconds: 1485467192, Nanos: 662813000},
			},
		},
	},
}
