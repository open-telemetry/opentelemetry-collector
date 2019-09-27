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

package zipkin

import (
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"math"
	"reflect"
	"sort"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"

	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestZipkinThriftFallbackToLocalComponent(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_thrift_local_component.json")
	if err != nil {
		t.Fatalf("failed to load test data: %v", err)
	}
	var ztSpans []*zipkincore.Span
	err = json.Unmarshal(blob, &ztSpans)
	if err != nil {
		t.Fatalf("failed to unmarshal json into zipkin v1 thrift: %v", err)
	}

	reqs, err := V1ThriftBatchToOCProto(ztSpans)
	if err != nil {
		t.Fatalf("failed to translate zipkinv1 thrift to OC proto: %v", err)
	}

	if len(reqs) != 2 {
		t.Fatalf("got %d trace service request(s), want 2", len(reqs))
	}

	// Ensure the order of nodes
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].Node.ServiceInfo.Name < reqs[j].Node.ServiceInfo.Name
	})

	// First span didn't have a host/endpoint to give service name, use the local component.
	got := reqs[0].Node.ServiceInfo.Name
	want := "myLocalComponent"
	if got != want {
		t.Fatalf("got %q for service name, want %q", got, want)
	}

	// Second span have a host/endpoint to give service name, do not use local component.
	got = reqs[1].Node.ServiceInfo.Name
	want = "myServiceName"
	if got != want {
		t.Fatalf("got %q for service name, want %q", got, want)
	}
}

func TestV1ThriftToOCProto(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_thrift_single_batch.json")
	if err != nil {
		t.Fatalf("failed to load test data: %v", err)
	}

	var ztSpans []*zipkincore.Span
	err = json.Unmarshal(blob, &ztSpans)
	if err != nil {
		t.Fatalf("failed to unmarshal json into zipkin v1 thrift: %v", err)
	}

	got, err := V1ThriftBatchToOCProto(ztSpans)
	if err != nil {
		t.Fatalf("failed to translate zipkinv1 thrift to OC proto: %v", err)
	}

	want := ocBatchesFromZipkinV1
	sortTraceByNodeName(want)
	sortTraceByNodeName(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got different data than want")
	}
}

func BenchmarkV1ThriftToOCProto(b *testing.B) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_thrift_single_batch.json")
	if err != nil {
		b.Fatalf("failed to load test data: %v", err)
	}

	var ztSpans []*zipkincore.Span
	err = json.Unmarshal(blob, &ztSpans)
	if err != nil {
		b.Fatalf("failed to unmarshal json into zipkin v1 thrift: %v", err)
	}

	for n := 0; n < b.N; n++ {
		V1ThriftBatchToOCProto(ztSpans)
	}
}

func TestZipkinThriftAnnotationsToOCStatus(t *testing.T) {
	type test struct {
		haveTags       []*zipkincore.BinaryAnnotation
		wantAttributes *tracepb.Span_Attributes
		wantStatus     *tracepb.Status
	}

	cases := []test{
		// too large code for OC
		{
			haveTags: []*zipkincore.BinaryAnnotation{{
				Key:            "status.code",
				Value:          uint64ToBytes(math.MaxInt64),
				AnnotationType: zipkincore.AnnotationType_I64,
			}},
			wantAttributes: nil,
			wantStatus:     nil,
		},
		// only status.code tag
		{
			haveTags: []*zipkincore.BinaryAnnotation{{
				Key:            "status.code",
				Value:          uint64ToBytes(5),
				AnnotationType: zipkincore.AnnotationType_I64,
			}},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code: 5,
			},
		},
		{
			haveTags: []*zipkincore.BinaryAnnotation{{
				Key:            "status.code",
				Value:          uint32ToBytes(6),
				AnnotationType: zipkincore.AnnotationType_I32,
			}},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code: 6,
			},
		},
		{
			haveTags: []*zipkincore.BinaryAnnotation{{
				Key:            "status.code",
				Value:          uint16ToBytes(7),
				AnnotationType: zipkincore.AnnotationType_I16,
			}},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code: 7,
			},
		},
		// only status.message tag
		{
			haveTags: []*zipkincore.BinaryAnnotation{{
				Key:            "status.message",
				Value:          []byte("Forbidden"),
				AnnotationType: zipkincore.AnnotationType_STRING,
			}},
			wantAttributes: nil,
			wantStatus:     nil,
		},
		// both status.code and status.message
		{
			haveTags: []*zipkincore.BinaryAnnotation{
				{
					Key:            "status.code",
					Value:          uint32ToBytes(13),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "status.message",
					Value:          []byte("Forbidden"),
					AnnotationType: zipkincore.AnnotationType_STRING,
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
			haveTags: []*zipkincore.BinaryAnnotation{
				{
					Key:            "http.status_code",
					Value:          uint32ToBytes(404),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "http.status_message",
					Value:          []byte("NotFound"),
					AnnotationType: zipkincore.AnnotationType_STRING,
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
			haveTags: []*zipkincore.BinaryAnnotation{
				{
					Key:            "http.status_code",
					Value:          uint32ToBytes(404),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "http.status_message",
					Value:          []byte("NotFound"),
					AnnotationType: zipkincore.AnnotationType_STRING,
				},
				{
					Key:            "status.code",
					Value:          uint32ToBytes(13),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "status.message",
					Value:          []byte("Forbidden"),
					AnnotationType: zipkincore.AnnotationType_STRING,
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
			haveTags: []*zipkincore.BinaryAnnotation{
				{
					Key:            "http.status_code",
					Value:          uint32ToBytes(404),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "http.status_message",
					Value:          []byte("NotFound"),
					AnnotationType: zipkincore.AnnotationType_STRING,
				},
				{
					Key:            "status.code",
					Value:          uint32ToBytes(14),
					AnnotationType: zipkincore.AnnotationType_I32,
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
			haveTags: []*zipkincore.BinaryAnnotation{
				{
					Key:            "http.status_code",
					Value:          uint32ToBytes(404),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "http.status_message",
					Value:          []byte("NotFound"),
					AnnotationType: zipkincore.AnnotationType_STRING,
				},
				{
					Key:            "status.message",
					Value:          []byte("Forbidden"),
					AnnotationType: zipkincore.AnnotationType_STRING,
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

		// census tags
		{
			haveTags: []*zipkincore.BinaryAnnotation{
				{
					Key:            "census.status_code",
					Value:          uint32ToBytes(18),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "census.status_description",
					Value:          []byte("RPCError"),
					AnnotationType: zipkincore.AnnotationType_STRING,
				},
			},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code:    18,
				Message: "RPCError",
			},
		},

		// census tags priority over others
		{
			haveTags: []*zipkincore.BinaryAnnotation{
				{
					Key:            "census.status_code",
					Value:          uint32ToBytes(18),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "census.status_description",
					Value:          []byte("RPCError"),
					AnnotationType: zipkincore.AnnotationType_STRING,
				},
				{
					Key:            "http.status_code",
					Value:          uint32ToBytes(404),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
				{
					Key:            "http.status_message",
					Value:          []byte("NotFound"),
					AnnotationType: zipkincore.AnnotationType_STRING,
				},
				{
					Key:            "status.message",
					Value:          []byte("Forbidden"),
					AnnotationType: zipkincore.AnnotationType_STRING,
				},
				{
					Key:            "status.code",
					Value:          uint32ToBytes(1),
					AnnotationType: zipkincore.AnnotationType_I32,
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
				Code:    18,
				Message: "RPCError",
			},
		},
	}

	for i, c := range cases {
		zSpans := []*zipkincore.Span{{
			ID:                1,
			TraceID:           1,
			BinaryAnnotations: c.haveTags,
		}}
		gb, err := V1ThriftBatchToOCProto(zSpans)
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		gs := gb[0].Spans[0]
		if !reflect.DeepEqual(gs.Attributes, c.wantAttributes) {
			t.Fatalf("Unsuccessful conversion %d\nGot:\n\t%v\nWant:\n\t%v", i, gs.Attributes, c.wantAttributes)
		}

		if !reflect.DeepEqual(gs.Status, c.wantStatus) {
			t.Fatalf("Unsuccessful conversion: %d\nGot:\n\t%v\nWant:\n\t%v", i, gs.Status, c.wantStatus)
		}
	}
}

func TestThirftHTTPToGRPCStatusCode(t *testing.T) {
	for i := int32(100); i <= 600; i++ {
		wantStatus := tracetranslator.OCStatusCodeFromHTTP(i)
		gb, err := V1ThriftBatchToOCProto([]*zipkincore.Span{{
			ID:      1,
			TraceID: 1,
			BinaryAnnotations: []*zipkincore.BinaryAnnotation{
				{
					Key:            "http.status_code",
					Value:          uint32ToBytes(uint32(i)),
					AnnotationType: zipkincore.AnnotationType_I32,
				},
			},
		}})
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		gs := gb[0].Spans[0]
		if !reflect.DeepEqual(gs.Status.Code, wantStatus) {
			t.Fatalf("Unsuccessful conversion: %d\nGot:\n\t%v\nWant:\n\t%v", i, gs.Status, wantStatus)
		}
	}
}

func Test_bytesInt16ToInt64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    int64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   nil,
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{0, 200},
			want:    200,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{0, 128, 200, 200},
			want:    128,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesInt16ToInt64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesInt16ToInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesInt16ToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bytesInt32ToInt64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    int64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   []byte{},
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{0, 0, 0, 202},
			want:    202,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{0, 0, 0, 128, 0, 0, 0, 0},
			want:    128,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesInt32ToInt64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesInt32ToInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesInt32ToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bytesInt64ToInt64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    int64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   []byte{0, 0, 0, 0},
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{0, 0, 0, 0, 0, 0, 0, 202},
			want:    202,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{0, 0, 0, 0, 0, 0, 0, 128, 0, 0, 0, 0},
			want:    128,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesInt64ToInt64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesInt64ToInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesInt64ToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bytesFloat64ToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    float64
		wantErr error
	}{
		{
			name:    "too short byte slice",
			bytes:   []byte{0, 0, 0, 0},
			want:    0,
			wantErr: errNotEnoughBytes,
		},
		{
			name:    "exact size byte slice",
			bytes:   []byte{64, 9, 33, 251, 84, 68, 45, 24},
			want:    3.141592653589793,
			wantErr: nil,
		},
		{
			name:    "large byte slice",
			bytes:   []byte{64, 9, 33, 251, 84, 68, 45, 24, 0, 0, 0, 0},
			want:    3.141592653589793,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesFloat64ToFloat64(tt.bytes)
			if err != tt.wantErr {
				t.Errorf("bytesFloat64ToFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("bytesFloat64ToFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func uint64ToBytes(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return b
}

func uint32ToBytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

func uint16ToBytes(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}
