// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
	"go.opentelemetry.io/collector/translator/trace/zipkinv2"
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
	zi := newTestZipkinReceiver()
	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/x-protobuf")

	// 3. Get that payload converted to OpenCensus proto spans.
	reqs, err := zi.v2ToTraceSpans(protoBlob, hdr)
	require.NoError(t, err, "Failed to parse convert Zipkin spans in Protobuf to Trace spans: %v", err)
	require.Equal(t, reqs.ResourceSpans().Len(), 2, "Expecting exactly 2 requests since spans have different node/localEndpoint: %v", reqs.ResourceSpans().Len())

	want := pdata.NewTraces()
	want.ResourceSpans().Resize(2)

	// First span/resource
	want.ResourceSpans().At(0).Resource().Attributes().UpsertString(conventions.AttributeServiceName, "svc-1")
	span0 := want.ResourceSpans().At(0).InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	span0.SetTraceID(pdata.NewTraceID([16]byte{0x7F, 0x6F, 0x5F, 0x4F, 0x3F, 0x2F, 0x1F, 0x0F, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0}))
	span0.SetSpanID(pdata.NewSpanID([8]byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0}))
	span0.SetParentSpanID(pdata.NewSpanID([8]byte{0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0}))
	span0.SetName("ProtoSpan1")
	span0.SetStartTimestamp(pdata.TimestampFromTime(now))
	span0.SetEndTimestamp(pdata.TimestampFromTime(now.Add(12 * time.Second)))
	span0.Attributes().UpsertString(conventions.AttributeNetHostIP, "192.168.0.1")
	span0.Attributes().UpsertInt(conventions.AttributeNetHostPort, 8009)
	span0.Attributes().UpsertString(conventions.AttributeNetPeerName, "memcached")
	span0.Attributes().UpsertString(conventions.AttributeNetPeerIP, "fe80::1453:a77c:da4d:d21b")
	span0.Attributes().UpsertInt(conventions.AttributeNetPeerPort, 11211)
	span0.Attributes().UpsertString(tracetranslator.TagSpanKind, string(tracetranslator.OpenTracingSpanKindConsumer))

	// Second span/resource
	want.ResourceSpans().At(1).Resource().Attributes().UpsertString(conventions.AttributeServiceName, "search")
	span1 := want.ResourceSpans().At(1).InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	span1.SetTraceID(pdata.NewTraceID([16]byte{0x7A, 0x6A, 0x5A, 0x4A, 0x3A, 0x2A, 0x1A, 0x0A, 0xC7, 0xC6, 0xC5, 0xC4, 0xC3, 0xC2, 0xC1, 0xC0}))
	span1.SetSpanID(pdata.NewSpanID([8]byte{0x67, 0x66, 0x65, 0x64, 0x63, 0x62, 0x61, 0x60}))
	span1.SetParentSpanID(pdata.NewSpanID([8]byte{0x17, 0x16, 0x15, 0x14, 0x13, 0x12, 0x11, 0x10}))
	span1.SetName("CacheWarmUp")
	span1.SetStartTimestamp(pdata.TimestampFromTime(now.Add(-10 * time.Hour)))
	span1.SetEndTimestamp(pdata.TimestampFromTime(now.Add(-10 * time.Hour).Add(7 * time.Second)))
	span1.Attributes().UpsertString(conventions.AttributeNetHostIP, "10.0.0.13")
	span1.Attributes().UpsertInt(conventions.AttributeNetHostPort, 8009)
	span1.Attributes().UpsertString(conventions.AttributeNetPeerName, "redis")
	span1.Attributes().UpsertString(conventions.AttributeNetPeerIP, "fe80::1453:a77c:da4d:d21b")
	span1.Attributes().UpsertInt(conventions.AttributeNetPeerPort, 6379)
	span1.Attributes().UpsertString(tracetranslator.TagSpanKind, string(tracetranslator.OpenTracingSpanKindProducer))
	span1.Events().Resize(2)
	span1.Events().At(0).SetName("DB reset")
	span1.Events().At(0).SetTimestamp(pdata.TimestampFromTime(now.Add(-10 * time.Hour)))
	span1.Events().At(1).SetName("GC Cycle 39")
	span1.Events().At(1).SetTimestamp(pdata.TimestampFromTime(now.Add(-10 * time.Hour)))

	assert.Equal(t, want.SpanCount(), reqs.SpanCount())
	assert.Equal(t, want.ResourceSpans().Len(), reqs.ResourceSpans().Len())
	for i := 0; i < want.ResourceSpans().Len(); i++ {
		wantRS := want.ResourceSpans().At(i)
		wSvcName, ok := wantRS.Resource().Attributes().Get(conventions.AttributeServiceName)
		assert.True(t, ok)
		for j := 0; j < reqs.ResourceSpans().Len(); j++ {
			reqsRS := reqs.ResourceSpans().At(j)
			rSvcName, ok := reqsRS.Resource().Attributes().Get(conventions.AttributeServiceName)
			assert.True(t, ok)
			if rSvcName.StringVal() == wSvcName.StringVal() {
				compareResourceSpans(t, wantRS, reqsRS)
			}
		}
	}
}

func newTestZipkinReceiver() *ZipkinReceiver {
	cfg := createDefaultConfig().(*Config)
	return &ZipkinReceiver{
		config:                   cfg,
		jsonUnmarshaler:          zipkinv2.NewJSONTracesUnmarshaler(cfg.ParseStringTags),
		protobufUnmarshaler:      zipkinv2.NewProtobufTracesUnmarshaler(false, cfg.ParseStringTags),
		protobufDebugUnmarshaler: zipkinv2.NewProtobufTracesUnmarshaler(true, cfg.ParseStringTags),
	}
}

func compareResourceSpans(t *testing.T, wantRS pdata.ResourceSpans, reqsRS pdata.ResourceSpans) {
	assert.Equal(t, wantRS.InstrumentationLibrarySpans().Len(), reqsRS.InstrumentationLibrarySpans().Len())
	wantIL := wantRS.InstrumentationLibrarySpans().At(0)
	reqsIL := reqsRS.InstrumentationLibrarySpans().At(0)
	assert.Equal(t, wantIL.Spans().Len(), reqsIL.Spans().Len())
}
