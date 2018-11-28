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

package tracetranslator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/internal/testutils"
)

func TestJaegerThriftBatchToOCProto(t *testing.T) {
	const numOfFiles = 2
	for i := 1; i <= 2; i++ {
		thriftInFile := fmt.Sprintf("./testdata/thrift_batch_%02d.json", i)
		jb := &jaeger.Batch{}
		if err := loadFromJSON(thriftInFile, jb); err != nil {
			t.Errorf("Failed load Jaeger Thrift from %q. Error: %v", thriftInFile, err)
			continue
		}

		octrace, err := JaegerThriftBatchToOCProto(jb)
		if err != nil {
			t.Errorf("Failed to handled Jaeger Thrift Batch from %q. Error: %v", thriftInFile, err)
			continue
		}

		wantSpanCount, gotSpanCount := len(jb.Spans), len(octrace.Spans)
		if wantSpanCount != gotSpanCount {
			t.Errorf("Different number of spans in the batches on pass #%d (want %d, got %d)", i, wantSpanCount, gotSpanCount)
			continue
		}

		gb, err := json.MarshalIndent(octrace, "", "  ")
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

		gj, wj := testutils.GenerateNormalizedJSON(string(gb)), testutils.GenerateNormalizedJSON(string(wb))
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

	var got []*agenttracepb.ExportTraceServiceRequest
	for i, batch := range batches {
		gb, err := JaegerThriftBatchToOCProto(batch)
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		got = append(got, gb)
	}

	want := []*agenttracepb.ExportTraceServiceRequest{
		{},
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: "testHere"},
				LibraryInfo: new(commonpb.LibraryInfo),
				Identifier:  new(commonpb.ProcessIdentifier),
				Attributes: map[string]string{
					"storage_version": "13",
				},
			},
			Spans: []*tracepb.Span{},
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
						Code:    403,
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
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"status.code": {
								Value: &tracepb.AttributeValue_IntValue{
									IntValue: 13,
								},
							},
							"status.message": {
								Value: &tracepb.AttributeValue_StringValue{
									StringValue: &tracepb.TruncatableString{Value: "proxy crashed"},
								},
							},
						},
					},
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
		gj, wj := jsonify(got), jsonify(want)
		if !bytes.Equal(gj, wj) {
			t.Fatalf("Unsuccessful conversion\nGot:\n\t%v\nWant:\n\t%v", got, want)
		}
	}
}

func jsonify(v interface{}) []byte {
	jb, _ := json.MarshalIndent(v, "", "   ")
	return jb
}
