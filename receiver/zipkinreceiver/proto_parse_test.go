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

package zipkinreceiver

import (
	"net/http"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/proto"
	zipkin_proto3 "github.com/openzipkin/zipkin-go/proto/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/translator/trace/zipkin"
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
	require.NoError(t, err, "Failed to protobuf serialize payload: %v", err)
	zi := new(ZipkinReceiver)
	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/x-protobuf")

	// 3. Get that payload converted to OpenCensus proto spans.
	reqs, err := zi.v2ToTraceSpans(protoBlob, hdr)
	require.NoError(t, err, "Failed to parse convert Zipkin spans in Protobuf to Trace spans: %v", err)
	require.Len(t, reqs, 2, "Expecting exactly 2 requests since spans have different node/localEndpoint: %v", len(reqs))

	want := []consumerdata.TraceData{
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: "svc-1",
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
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							zipkin.LocalEndpointIPv4:         {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "192.168.0.1"}}},
							zipkin.LocalEndpointPort:         {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "8009"}}},
							zipkin.RemoteEndpointServiceName: {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "memcached"}}},
							zipkin.RemoteEndpointIPv6:        {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "fe80::1453:a77c:da4d:d21b"}}},
							zipkin.RemoteEndpointPort:        {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "11211"}}},
						},
					},
				},
			},
		},
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: "search",
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
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							zipkin.LocalEndpointIPv4:         {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "10.0.0.13"}}},
							zipkin.LocalEndpointPort:         {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "8009"}}},
							zipkin.RemoteEndpointServiceName: {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "redis"}}},
							zipkin.RemoteEndpointIPv6:        {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "fe80::1453:a77c:da4d:d21b"}}},
							zipkin.RemoteEndpointPort:        {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "6379"}}},
						},
					},
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

	assert.Equal(t, want, reqs)
}
