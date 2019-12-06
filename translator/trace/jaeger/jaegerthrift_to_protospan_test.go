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
	"io/ioutil"
	"reflect"
	"sort"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestThriftBatchToOCProto_Roundtrip(t *testing.T) {
	const numOfFiles = 2
	for i := 0; i < numOfFiles; i++ {
		// Jaeger binary tags do not round trip from Jaeger -> OCProto -> Jaeger.
		// For tests use data without binary tags.
		thriftFile := fmt.Sprintf("./testdata/thrift_batch_no_binary_tags_%02d.json", i+1)
		wantJBatch := &jaeger.Batch{}
		if err := loadFromJSON(thriftFile, wantJBatch); err != nil {
			t.Errorf("Failed load Jaeger Thrift from %q: %v", thriftFile, err)
			continue
		}

		// Remove process tag types that are known to not roundtrip (OC Proto has all of these as strings).
		cleanTags := make([]*jaeger.Tag, 0, len(wantJBatch.Process.Tags))
		for _, jTag := range wantJBatch.Process.Tags {
			if jTag.GetVType() == jaeger.TagType_STRING {
				cleanTags = append(cleanTags, jTag)
			}
		}
		wantJBatch.Process.Tags = cleanTags

		ocBatch, err := ThriftBatchToOCProto(wantJBatch)
		if err != nil {
			t.Errorf("Failed to read to read Jaeger Thrift from %q: %v", thriftFile, err)
			continue
		}

		gotJBatch, err := OCProtoToJaegerThrift(ocBatch)
		if err != nil {
			t.Errorf("Failed to translate OC batch to Jaeger Thrift: %v", err)
			continue
		}

		wantSpanCount, gotSpanCount := len(ocBatch.Spans), len(gotJBatch.Spans)
		if wantSpanCount != gotSpanCount {
			t.Errorf("Different number of spans in the batches on pass #%d (want %d, got %d)", i, wantSpanCount, gotSpanCount)
			continue
		}

		// Sort tags to help with comparison, not only for jaeger.Process but also on each span.
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
			sort.Slice(jSpan.Logs, func(i, j int) bool {
				sort.Slice(jSpan.Logs[i].Fields, func(k, l int) bool {
					return jSpan.Logs[i].Fields[k].Key < jSpan.Logs[i].Fields[l].Key
				})
				sort.Slice(jSpan.Logs[j].Fields, func(k, l int) bool {
					return jSpan.Logs[j].Fields[k].Key < jSpan.Logs[j].Fields[l].Key
				})
				return jSpan.Logs[i].Timestamp < jSpan.Logs[j].Timestamp
			})
		}

		gjson, _ := json.MarshalIndent(gotJBatch, "", "  ")
		wjson, _ := json.MarshalIndent(wantJBatch, "", "  ")
		gjsonStr := testutils.GenerateNormalizedJSON(t, string(gjson))
		wjsonStr := testutils.GenerateNormalizedJSON(t, string(wjson))
		if gjsonStr != wjsonStr {
			t.Errorf("OC Proto to Jaeger Thrift failed.\nGot:\n%s\nWant:\n%s\n", gjson, wjson)
		}
	}
}

func TestThriftBatchToOCProto(t *testing.T) {
	const numOfFiles = 2
	for i := 1; i <= numOfFiles; i++ {
		thriftInFile := fmt.Sprintf("./testdata/thrift_batch_%02d.json", i)
		jb := &jaeger.Batch{}
		if err := loadFromJSON(thriftInFile, jb); err != nil {
			t.Errorf("Failed load Jaeger Thrift from %q. Error: %v", thriftInFile, err)
			continue
		}

		td, err := ThriftBatchToOCProto(jb)
		if err != nil {
			t.Errorf("Failed to handled Jaeger Thrift Batch from %q. Error: %v", thriftInFile, err)
			continue
		}

		wantSpanCount, gotSpanCount := len(jb.Spans), len(td.Spans)
		if wantSpanCount != gotSpanCount {
			t.Errorf("Different number of spans in the batches on pass #%d (want %d, got %d)", i, wantSpanCount, gotSpanCount)
			continue
		}

		gb, err := json.MarshalIndent(td, "", "  ")
		if err != nil {
			t.Errorf("Failed to convert received OC proto to json. Error: %v", err)
			continue
		}

		protoFile := fmt.Sprintf("./testdata/ocproto_batch_%02d.json", i)
		wb, err := ioutil.ReadFile(protoFile)
		if err != nil {
			t.Errorf("Failed to read file %q with expected OC proto in JSON format. Error: %v", protoFile, err)
			continue
		}

		gj, wj := testutils.GenerateNormalizedJSON(t, string(gb)), testutils.GenerateNormalizedJSON(t, string(wb))
		if gj != wj {
			t.Errorf("The roundtrip JSON doesn't match the JSON that we want\nGot:\n%s\nWant:\n%s", gj, wj)
		}
	}
}

func loadFromJSON(file string, obj interface{}) error {
	blob, err := ioutil.ReadFile(file)
	if err == nil {
		err = json.Unmarshal(blob, obj)
	}

	return err
}

// This test ensures that we conservatively allocate, only creating memory when necessary.
func TestConservativeConversions(t *testing.T) {
	batches := []*jaeger.Batch{
		{
			Process: nil,
			Spans: []*jaeger.Span{
				{}, // Blank span
			},
		},
		{
			Process: &jaeger.Process{
				ServiceName: "testHere",
				Tags: []*jaeger.Tag{
					{
						Key:   "storage_version",
						VLong: func() *int64 { v := int64(13); return &v }(),
						VType: jaeger.TagType_LONG,
					},
				},
			},
			Spans: []*jaeger.Span{
				{}, // Blank span
			},
		},
		{
			Process: &jaeger.Process{
				ServiceName: "test2",
			},
			Spans: []*jaeger.Span{
				{TraceIdLow: 0, TraceIdHigh: 0},
				{TraceIdLow: 0x01111111FFFFFFFF, TraceIdHigh: 0x0011121314111111},
				{
					OperationName: "HTTP call",
					TraceIdLow:    0x01111111FFFFFFFF, TraceIdHigh: 0x0011121314111111,
					Tags: []*jaeger.Tag{
						{
							Key:   "http.status_code",
							VLong: func() *int64 { v := int64(403); return &v }(),
							VType: jaeger.TagType_LONG,
						},
						{
							Key:   "http.status_message",
							VStr:  func() *string { v := "Forbidden"; return &v }(),
							VType: jaeger.TagType_STRING,
						},
					},
				},
				{
					OperationName: "RPC call",
					TraceIdLow:    0x01111111FFFFFFFF, TraceIdHigh: 0x0011121314111111,
					Tags: []*jaeger.Tag{
						{
							Key:   "status.code",
							VLong: func() *int64 { v := int64(13); return &v }(),
							VType: jaeger.TagType_LONG,
						},
						{
							Key:   "status.message",
							VStr:  func() *string { v := "proxy crashed"; return &v }(),
							VType: jaeger.TagType_STRING,
						},
					},
				},
			},
		},
		{
			Spans: []*jaeger.Span{
				{
					OperationName: "ThisOne",
					TraceIdLow:    0x1001021314151617,
					TraceIdHigh:   0x100102F3F4F5F6F7,
					SpanId:        0x1011121314151617,
					ParentSpanId:  0x10F1F2F3F4F5F6F7,
					Tags: []*jaeger.Tag{
						{
							Key:   "cache_miss",
							VBool: func() *bool { v := true; return &v }(),
							VType: jaeger.TagType_BOOL,
						},
					},
				},
			},
		},
	}

	got := make([]consumerdata.TraceData, 0, len(batches))
	for i, batch := range batches {
		gb, err := ThriftBatchToOCProto(batch)
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		got = append(got, gb)
	}

	want := []consumerdata.TraceData{
		{
			// The conversion returns a slice with capacity equals to the number of elements in the
			// jager batch spans, even if the element is nil.
			Spans: make([]*tracepb.Span, 0, 1),
		},
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: "testHere"},
				LibraryInfo: new(commonpb.LibraryInfo),
				Identifier:  new(commonpb.ProcessIdentifier),
				Attributes: map[string]string{
					"storage_version": "13",
				},
			},
			// The conversion returns a slice with capacity equals to the number of elements in the
			// jager batch spans, even if the element is nil.
			Spans: make([]*tracepb.Span, 0, 1),
		},
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: "test2"},
				LibraryInfo: new(commonpb.LibraryInfo),
				Identifier:  new(commonpb.ProcessIdentifier),
			},
			Spans: []*tracepb.Span{
				// The first span should be missing
				{
					TraceId: []byte{0x00, 0x11, 0x12, 0x13, 0x14, 0x11, 0x11, 0x11, 0x01, 0x11, 0x11, 0x11, 0xFF, 0xFF, 0xFF, 0xFF},
				},
				{
					Name:    &tracepb.TruncatableString{Value: "HTTP call"},
					TraceId: []byte{0x00, 0x11, 0x12, 0x13, 0x14, 0x11, 0x11, 0x11, 0x01, 0x11, 0x11, 0x11, 0xFF, 0xFF, 0xFF, 0xFF},
					// Ensure that the status code was properly translated
					Status: &tracepb.Status{
						Code:    7,
						Message: "Forbidden",
					},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"http.status_code": {
								Value: &tracepb.AttributeValue_IntValue{
									IntValue: 403,
								},
							},
							"http.status_message": {
								Value: &tracepb.AttributeValue_StringValue{
									StringValue: &tracepb.TruncatableString{Value: "Forbidden"},
								},
							},
						},
					},
				},
				{
					Name:    &tracepb.TruncatableString{Value: "RPC call"},
					TraceId: []byte{0x00, 0x11, 0x12, 0x13, 0x14, 0x11, 0x11, 0x11, 0x01, 0x11, 0x11, 0x11, 0xFF, 0xFF, 0xFF, 0xFF},
					// Ensure that the status code was properly translated
					Status: &tracepb.Status{
						Code:    13,
						Message: "proxy crashed",
					},
					Attributes: nil,
				},
			},
		},
		{
			Spans: []*tracepb.Span{
				{
					Name:         &tracepb.TruncatableString{Value: "ThisOne"},
					SpanId:       []byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17},
					ParentSpanId: []byte{0x10, 0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7},
					TraceId:      []byte{0x10, 0x01, 0x02, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0x10, 0x01, 0x02, 0x13, 0x14, 0x15, 0x16, 0x17},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"cache_miss": {
								Value: &tracepb.AttributeValue_BoolValue{
									BoolValue: true,
								},
							},
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unsuccessful conversion\nGot:\n\t%v\nWant:\n\t%v", got, want)
	}
}

func TestJaegerStatusTagsToOCStatus(t *testing.T) {
	type test struct {
		haveTags       []*jaeger.Tag
		wantAttributes *tracepb.Span_Attributes
		wantStatus     *tracepb.Status
	}

	cases := []test{
		// only status.code tag
		{
			haveTags: []*jaeger.Tag{
				{
					Key:   "status.code",
					VLong: func() *int64 { v := int64(13); return &v }(),
					VType: jaeger.TagType_LONG,
				},
			},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code: 13,
			},
		},
		// only status.message tag
		{
			haveTags: []*jaeger.Tag{
				{
					Key:   "status.message",
					VStr:  func() *string { v := "Forbidden"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
			},
			wantAttributes: nil,
			wantStatus:     nil,
		},
		// both status.code and status.message
		{
			haveTags: []*jaeger.Tag{
				{
					Key:   "status.code",
					VLong: func() *int64 { v := int64(13); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   "status.message",
					VStr:  func() *string { v := "Forbidden"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
			},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code:    13,
				Message: "Forbidden",
			},
		},

		// http status.code
		{
			haveTags: []*jaeger.Tag{
				{
					Key:   "http.status_code",
					VLong: func() *int64 { v := int64(404); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   "http.status_message",
					VStr:  func() *string { v := "NotFound"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code:    5,
				Message: "NotFound",
			},
		},

		// http and oc
		{
			haveTags: []*jaeger.Tag{
				{
					Key:   "http.status_code",
					VLong: func() *int64 { v := int64(404); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   "http.status_message",
					VStr:  func() *string { v := "NotFound"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
				{
					Key:   "status.code",
					VLong: func() *int64 { v := int64(13); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   "status.message",
					VStr:  func() *string { v := "Forbidden"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code:    13,
				Message: "Forbidden",
			},
		},
		// http and only oc code
		{
			haveTags: []*jaeger.Tag{
				{
					Key:   "http.status_code",
					VLong: func() *int64 { v := int64(404); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   "http.status_message",
					VStr:  func() *string { v := "NotFound"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
				{
					Key:   "status.code",
					VLong: func() *int64 { v := int64(14); return &v }(),
					VType: jaeger.TagType_LONG,
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code: 14,
			},
		},
		// http and only oc message
		{
			haveTags: []*jaeger.Tag{
				{
					Key:   "http.status_code",
					VLong: func() *int64 { v := int64(404); return &v }(),
					VType: jaeger.TagType_LONG,
				},
				{
					Key:   "http.status_message",
					VStr:  func() *string { v := "NotFound"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
				{
					Key:   "status.message",
					VStr:  func() *string { v := "Forbidden"; return &v }(),
					VType: jaeger.TagType_STRING,
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code:    5,
				Message: "NotFound",
			},
		},
	}

	for i, c := range cases {
		gb, err := ThriftBatchToOCProto(&jaeger.Batch{
			Process: nil,
			Spans: []*jaeger.Span{{
				TraceIdLow:  0x1001021314151617,
				TraceIdHigh: 0x100102F3F4F5F6F7,
				SpanId:      0x1011121314151617,
				Tags:        c.haveTags,
			}},
		})
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		gs := gb.Spans[0]
		if !reflect.DeepEqual(gs.Attributes, c.wantAttributes) {
			t.Fatalf("Unsuccessful conversion: %d\nGot:\n\t%v\nWant:\n\t%v", i, gs.Attributes, c.wantAttributes)
		}
		if !reflect.DeepEqual(gs.Status, c.wantStatus) {
			t.Fatalf("Unsuccessful conversion: %d\nGot:\n\t%v\nWant:\n\t%v", i, gs.Status, c.wantStatus)
		}
	}
}

func TestHTTPToGRPCStatusCode(t *testing.T) {
	for i := int64(100); i <= 600; i++ {
		wantStatus := tracetranslator.OCStatusCodeFromHTTP(int32(i))
		gb, err := ThriftBatchToOCProto(&jaeger.Batch{
			Process: nil,
			Spans: []*jaeger.Span{{
				TraceIdLow:  0x1001021314151617,
				TraceIdHigh: 0x100102F3F4F5F6F7,
				SpanId:      0x1011121314151617,
				Tags: []*jaeger.Tag{{
					Key:   "http.status_code",
					VLong: &i,
					VType: jaeger.TagType_LONG,
				}},
			}},
		})
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		gs := gb.Spans[0]
		if !reflect.DeepEqual(gs.Status.Code, wantStatus) {
			t.Fatalf("Unsuccessful conversion: %d\nGot:\n\t%v\nWant:\n\t%v", i, gs.Status, wantStatus)
		}
	}
}
