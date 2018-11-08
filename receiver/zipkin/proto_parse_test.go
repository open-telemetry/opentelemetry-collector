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

package zipkinreceiver

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	zipkin_proto3 "github.com/openzipkin/zipkin-go/proto/v2"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/internal"
)

func TestConvertSpansToTraceSpans_protobuf(t *testing.T) {
	// TODO: (@odeke-em) examine the entire codepath that time goes through
	//  in Zipkin-Go to ensure that all rounding matches. Otherwise
	// for now we need to round Annotation times to seconds for comparison.
	cmpTimestamp := func(t time.Time) time.Time {
		return t.Round(time.Second)
	}

	now := cmpTimestamp(time.Date(2018, 10, 31, 19, 43, 35, 789, time.UTC))
	minus10hr5ms := cmpTimestamp(now.Add(-(10*time.Hour + 5*time.Millisecond)))

	// 1. Generate some spans then serialize them with protobuf
	payloadFromWild := &zipkin_proto3.ListOfSpans{
		Spans: []*zipkin_proto3.Span{
			{
				TraceId:   []byte{0x7F, 0x6F, 0x5F, 0x4F, 0x3F, 0x2F, 0x1F, 0x0F, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
				Id:        []byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
				ParentId:  []byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
				Name:      "ProtoSpan1",
				Kind:      zipkin_proto3.Span_CONSUMER,
				Timestamp: uint64(now.UnixNano() / 1e3),
				Duration:  12e6, // 12 seconds
				LocalEndpoint: &zipkin_proto3.Endpoint{
					ServiceName: "svc-1",
					Ipv4:        []byte{0xC0, 0xA8, 0x00, 0x01},
					Port:        8009,
				},
				RemoteEndpoint: &zipkin_proto3.Endpoint{
					ServiceName: "memcached",
					Ipv6:        []byte{0xFE, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14, 0x53, 0xa7, 0x7c, 0xda, 0x4d, 0xd2, 0x1b},
					Port:        11211,
				},
			},
			{
				TraceId:   []byte{0x7A, 0x6A, 0x5A, 0x4A, 0x3A, 0x2A, 0x1A, 0x0A, 0xC7, 0xC6, 0xC5, 0xC4, 0xC3, 0xC2, 0xC1, 0xC0},
				Id:        []byte{0x67, 0x66, 0x65, 0x64, 0x63, 0x62, 0x61, 0x60},
				ParentId:  []byte{0x17, 0x16, 0x15, 0x14, 0x13, 0x12, 0x11, 0x10},
				Name:      "CacheWarmUp",
				Kind:      zipkin_proto3.Span_PRODUCER,
				Timestamp: uint64(minus10hr5ms.UnixNano() / 1e3),
				Duration:  7e6, // 7 seconds
				LocalEndpoint: &zipkin_proto3.Endpoint{
					ServiceName: "search",
					Ipv4:        []byte{0x0A, 0x00, 0x00, 0x0D},
					Port:        8009,
				},
				RemoteEndpoint: &zipkin_proto3.Endpoint{
					ServiceName: "redis",
					Ipv6:        []byte{0xFE, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x14, 0x53, 0xa7, 0x7c, 0xda, 0x4d, 0xd2, 0x1b},
					Port:        6379,
				},
				Annotations: []*zipkin_proto3.Annotation{
					{
						Timestamp: uint64(minus10hr5ms.UnixNano() / 1e3),
						Value:     "DB reset",
					},
					{
						Timestamp: uint64(minus10hr5ms.UnixNano() / 1e3),
						Value:     "GC Cycle 39",
					},
				},
			},
		},
	}

	// 2. Serialize it
	protoBlob, err := proto.Marshal(payloadFromWild)
	if err != nil {
		t.Fatalf("Failed to protobuf serialize payload: %v", err)
	}
	zi := new(ZipkinReceiver)
	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/x-protobuf")

	// 3. Get that payload converted to OpenCensus proto spans.
	reqs, err := zi.parseAndConvertToTraceSpans(protoBlob, hdr)
	if err != nil {
		t.Fatalf("Failed to parse convert Zipkin spans in Protobuf to Trace spans: %v", err)
	}

	if g, w := len(reqs), 2; g != w {
		t.Fatalf("Expecting exactly 2 requests since spans have different node/localEndpoint: %v", g)
	}

	want := []*agenttracepb.ExportTraceServiceRequest{
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: "svc-1",
				},
				Attributes: map[string]string{
					"ipv4":                              "192.168.0.1",
					"serviceName":                       "svc-1",
					"port":                              "8009",
					"zipkin.remoteEndpoint.serviceName": "memcached",
					"zipkin.remoteEndpoint.ipv6":        "fe80::1453:a77c:da4d:d21b",
					"zipkin.remoteEndpoint.port":        "11211",
				},
			},
			Spans: []*tracepb.Span{
				{
					TraceId:      []byte{0x7F, 0x6F, 0x5F, 0x4F, 0x3F, 0x2F, 0x1F, 0x0F, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
					SpanId:       []byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
					ParentSpanId: []byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
					Name:         &tracepb.TruncatableString{Value: "ProtoSpan1"},
					StartTime:    internal.TimeToTimestamp(now),
					EndTime:      internal.TimeToTimestamp(now.Add(12 * time.Second)),
				},
			},
		},
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: "search",
				},
				Attributes: map[string]string{
					"ipv4":                              "10.0.0.13",
					"serviceName":                       "search",
					"port":                              "8009",
					"zipkin.remoteEndpoint.serviceName": "redis",
					"zipkin.remoteEndpoint.ipv6":        "fe80::1453:a77c:da4d:d21b",
					"zipkin.remoteEndpoint.port":        "6379",
				},
			},
			Spans: []*tracepb.Span{
				{
					TraceId:      []byte{0x7A, 0x6A, 0x5A, 0x4A, 0x3A, 0x2A, 0x1A, 0x0A, 0xC7, 0xC6, 0xC5, 0xC4, 0xC3, 0xC2, 0xC1, 0xC0},
					SpanId:       []byte{0x67, 0x66, 0x65, 0x64, 0x63, 0x62, 0x61, 0x60},
					ParentSpanId: []byte{0x17, 0x16, 0x15, 0x14, 0x13, 0x12, 0x11, 0x10},
					Name:         &tracepb.TruncatableString{Value: "CacheWarmUp"},
					StartTime:    internal.TimeToTimestamp(now.Add(-10 * time.Hour)),
					EndTime:      internal.TimeToTimestamp(now.Add(-10 * time.Hour).Add(7 * time.Second)),
					TimeEvents: &tracepb.Span_TimeEvents{
						TimeEvent: []*tracepb.Span_TimeEvent{
							{
								Time: internal.TimeToTimestamp(cmpTimestamp(now.Add(-10 * time.Hour))),
								Value: &tracepb.Span_TimeEvent_Annotation_{
									Annotation: &tracepb.Span_TimeEvent_Annotation{
										Description: &tracepb.TruncatableString{
											Value: "DB reset",
										},
									},
								},
							},
							{
								Time: internal.TimeToTimestamp(minus10hr5ms),
								Value: &tracepb.Span_TimeEvent_Annotation_{
									Annotation: &tracepb.Span_TimeEvent_Annotation{
										Description: &tracepb.TruncatableString{
											Value: "GC Cycle 39",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if g, w := reqs, want; !reflect.DeepEqual(g, w) {
		t.Errorf("Got:\n\t%v\nWant:\n\t%v", g, w)
	}
}
