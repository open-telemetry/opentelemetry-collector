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
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestThriftInvalidOCProtoIDs(t *testing.T) {
	fakeTraceID := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	tests := []struct {
		name         string
		ocSpans      []*tracepb.Span
		wantErr      error // nil means that we check for the message of the wrapped error
		wrappedError error // when wantErr is nil we expect this error to have been wrapped by the one received
	}{
		{
			name:         "nil TraceID",
			ocSpans:      []*tracepb.Span{{}},
			wantErr:      nil,
			wrappedError: tracetranslator.ErrNilTraceID,
		},
		{
			name:         "empty TraceID",
			ocSpans:      []*tracepb.Span{{TraceId: []byte{}}},
			wantErr:      nil,
			wrappedError: tracetranslator.ErrWrongLenTraceID,
		},
		{
			name:    "zero TraceID",
			ocSpans: []*tracepb.Span{{TraceId: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}},
			wantErr: errZeroTraceID,
		},
		{
			name:         "nil SpanID",
			ocSpans:      []*tracepb.Span{{TraceId: fakeTraceID}},
			wantErr:      nil,
			wrappedError: tracetranslator.ErrNilSpanID,
		},
		{
			name:         "empty SpanID",
			ocSpans:      []*tracepb.Span{{TraceId: fakeTraceID, SpanId: []byte{}}},
			wantErr:      nil,
			wrappedError: tracetranslator.ErrWrongLenSpanID,
		},
		{
			name:    "zero SpanID",
			ocSpans: []*tracepb.Span{{TraceId: fakeTraceID, SpanId: []byte{0, 0, 0, 0, 0, 0, 0, 0}}},
			wantErr: errZeroSpanID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ocSpansToJaegerSpans(tt.ocSpans)
			if err == nil {
				t.Error("ocSpansToJaegerSpans() no error, want error")
				return
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("ocSpansToJaegerSpans() = %v, want %v", err, tt.wantErr)
			}
			if tt.wrappedError != nil && !strings.Contains(err.Error(), tt.wrappedError.Error()) {
				t.Errorf("ocSpansToJaegerSpans() = %v, want it to wrap error %v", err, tt.wrappedError)
			}
		})
	}
}

func TestNilOCProtoNodeToJaegerThrift(t *testing.T) {
	nilNodeBatch := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				TraceId: []byte("0123456789abcdef"),
				SpanId:  []byte("01234567"),
			},
		},
	}
	got, err := OCProtoToJaegerThrift(nilNodeBatch)
	if err != nil {
		t.Fatalf("Failed to translate OC batch to Jaeger Thrift: %v", err)
	}
	if got.Process == nil {
		t.Fatalf("Jaeger requires a non-nil Process field")
	}
	if got.Process != unknownProcess {
		t.Fatalf("got unexpected Jaeger Process field")
	}
}

func TestOCProtoToJaegerThrift(t *testing.T) {
	const numOfFiles = 2
	for i := 0; i < numOfFiles; i++ {
		td := tds[i]

		gotJBatch, err := OCProtoToJaegerThrift(td)
		if err != nil {
			t.Errorf("Failed to translate OC batch to Jaeger Thrift: %v", err)
			continue
		}

		wantSpanCount, gotSpanCount := len(td.Spans), len(gotJBatch.Spans)
		if wantSpanCount != gotSpanCount {
			t.Errorf("Different number of spans in the batches on pass #%d (want %d, got %d)", i, wantSpanCount, gotSpanCount)
			continue
		}

		// Jaeger binary tags do not round trip from Jaeger -> OCProto -> Jaeger.
		// For tests use data without binary tags.
		thriftFile := fmt.Sprintf("./testdata/thrift_batch_no_binary_tags_%02d.json", i+1)
		wantJBatch := &jaeger.Batch{}
		if err := loadFromJSON(thriftFile, wantJBatch); err != nil {
			t.Errorf("Failed load Jaeger Thrift from %q: %v", thriftFile, err)
			continue
		}

		// Sort objects for comparison purposes.
		sortJaegerThriftBatch(wantJBatch)
		sortJaegerThriftBatch(gotJBatch)

		if diff := cmp.Diff(gotJBatch, wantJBatch); diff != "" {
			// Note: Lines with "-" at the beginning refer to values in gotJBatch. Lines with "+" refer to values in
			// wantJBatch.
			t.Errorf("OC Proto to Jaeger Thrift failed with following difference: %s", diff)
		}
	}
}

// Helper method to sort jaeger.Batch for cmp.Diff.
func sortJaegerThriftBatch(batch *jaeger.Batch) {
	// First sort the process tags.
	sort.Slice(batch.Process.Tags, func(i, j int) bool {
		return batch.Process.Tags[i].Key < batch.Process.Tags[j].Key
	})
	// Sort the span tags and the span log fields.
	for _, jSpan := range batch.Spans {

		sort.Slice(jSpan.Tags, func(i, j int) bool {
			return jSpan.Tags[i].Key < jSpan.Tags[j].Key
		})
		for _, jSpanLog := range jSpan.Logs {
			sort.Slice(jSpanLog.Fields, func(i, j int) bool {
				return jSpanLog.Fields[i].Key < jSpanLog.Fields[j].Key
			})
		}
	}

}

func TestOCStatusToJaegerThriftTags(t *testing.T) {

	type test struct {
		haveAttributes *tracepb.Span_Attributes
		haveStatus     *tracepb.Status
		wantTags       []*jaeger.Tag
	}

	cases := []test{
		// only status.code
		{
			haveAttributes: nil,
			haveStatus: &tracepb.Status{
				Code: 10,
			},
			wantTags: []*jaeger.Tag{
				{
					Key:   tracetranslator.TagStatusCode,
					VLong: func() *int64 { v := int64(10); return &v }(),
					VType: jaeger.TagType_LONG,
				},
			},
		},
		// only status.message
		{
			haveAttributes: nil,
			haveStatus: &tracepb.Status{
				Message: "Message",
			},
			wantTags: []*jaeger.Tag{
				{
					Key:   tracetranslator.TagStatusCode,
					VLong: func() *int64 { v := int64(0); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  func() *string { v := "Message"; return &v }(),
					VType: jaeger.TagType_STRING,
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
			wantTags: []*jaeger.Tag{
				{
					Key:   tracetranslator.TagStatusCode,
					VLong: func() *int64 { v := int64(12); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  func() *string { v := "Forbidden"; return &v }(),
					VType: jaeger.TagType_STRING,
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
			wantTags: []*jaeger.Tag{
				{
					Key:   tracetranslator.TagStatusCode,
					VLong: func() *int64 { v := int64(13); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  func() *string { v := "Error"; return &v }(),
					VType: jaeger.TagType_STRING,
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
			wantTags: []*jaeger.Tag{
				{
					Key:   tracetranslator.TagStatusCode,
					VLong: func() *int64 { v := int64(13); return &v }(),
					VType: jaeger.TagType_LONG,
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
			wantTags: []*jaeger.Tag{
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  func() *string { v := "Error"; return &v }(),
					VType: jaeger.TagType_STRING,
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
			wantTags: []*jaeger.Tag{
				{
					Key:   tracetranslator.TagHTTPStatusCode,
					VLong: func() *int64 { v := int64(404); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   tracetranslator.TagHTTPStatusMsg,
					VStr:  func() *string { v := "NotFound"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
				{
					Key:   tracetranslator.TagStatusCode,
					VLong: func() *int64 { v := int64(13); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   tracetranslator.TagStatusMsg,
					VStr:  func() *string { v := "Forbidden"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
			},
		},
	}

	fakeTraceID := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	fakeSpanID := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	for i, c := range cases {
		gb, err := OCProtoToJaegerThrift(consumerdata.TraceData{
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

// tds has the TraceData proto used in the test. They are hard coded because
// structs like tracepb.AttributeMap cannot be ready from JSON.
var tds = []consumerdata.TraceData{
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
		Resource: &resourcepb.Resource{
			Type:   "k8s.io/container",
			Labels: map[string]string{"resource_key1": "resource_val1"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52, 0x96, 0x9A, 0x89, 0x55, 0x57, 0x1A, 0x3F},
				SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
				ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x68, 0xC4, 0xE3},
				Name:         &tracepb.TruncatableString{Value: "get"},
				Kind:         tracepb.Span_CLIENT,
				StartTime:    &timestamp.Timestamp{Seconds: 1485467191, Nanos: 639875000},
				EndTime:      &timestamp.Timestamp{Seconds: 1485467191, Nanos: 662813000},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"http.url": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "http://localhost:15598/client_transactions"}},
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
						"span.kind": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "client"}},
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
				SpanId:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x7D, 0x98},
				ParentSpanId: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				Name:         &tracepb.TruncatableString{Value: "get2"},
				StartTime:    &timestamp.Timestamp{Seconds: 1485467192, Nanos: 639875000},
				EndTime:      &timestamp.Timestamp{Seconds: 1485467192, Nanos: 662813000},
			},
		},
	},
}
