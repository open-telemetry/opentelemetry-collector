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

package jaeger_test

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/internal"
	jaegerreceiver "github.com/census-instrumentation/opencensus-service/receiver/jaeger"
	"github.com/census-instrumentation/opencensus-service/receiver/testhelper"
)

func TestJaegerAgentUDP_ThriftCompact_6831(t *testing.T) {
	port := 6831
	addrForClient := fmt.Sprintf(":%d", port)
	testJaegerAgent(t, addrForClient, &jaegerreceiver.Configuration{
		AgentCompactThriftPort: port,
	})
}

func TestJaegerAgentUDP_ThriftBinary_6832(t *testing.T) {
	t.Skipf("Unfortunately due to Jaeger internal versioning, OpenCensus-Go's Thrift seems to conflict with ours")

	port := 6832
	addrForClient := fmt.Sprintf(":%d", port)
	testJaegerAgent(t, addrForClient, &jaegerreceiver.Configuration{
		AgentBinaryThriftPort: port,
	})
}

func testJaegerAgent(t *testing.T, agentEndpoint string, receiverConfig *jaegerreceiver.Configuration) {
	// 1. Create the Jaeger receiver aka "server"
	jr, err := jaegerreceiver.New(context.Background(), receiverConfig)
	if err != nil {
		t.Fatalf("Failed to create new Jaeger Receiver: %v", err)
	}
	defer jr.StopTraceReception(context.Background())

	sink := new(testhelper.ConcurrentSpanSink)
	if err := jr.StartTraceReception(context.Background(), sink); err != nil {
		t.Fatalf("StartTraceReception failed: %v", err)
	}

	now := time.Unix(1542158650, 536343000).UTC()
	nowPlus10min := now.Add(10 * time.Minute)
	nowPlus10min2sec := now.Add(10 * time.Minute).Add(2 * time.Second)

	// 2. Then with a "live application", send spans to the Jaeger exporter.
	jexp, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: agentEndpoint,
		ServiceName:   "TestingAgentUDP",
		Process: jaeger.Process{
			ServiceName: "issaTest",
			Tags: []jaeger.Tag{
				jaeger.BoolTag("bool", true),
				jaeger.StringTag("string", "yes"),
				jaeger.Int64Tag("int64", 1e7),
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create the Jaeger OpenCensus exporter for the live application: %v", err)
	}

	// 3. Now finally send some spans
	spandata := []*trace.SpanData{
		{
			SpanContext: trace.SpanContext{
				TraceID: trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x80},
				SpanID:  trace.SpanID{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8},
			},
			ParentSpanID: trace.SpanID{0x1F, 0x1E, 0x1D, 0x1C, 0x1B, 0x1A, 0x19, 0x18},
			Name:         "DBSearch",
			StartTime:    now,
			EndTime:      nowPlus10min,
			Status: trace.Status{
				Code:    trace.StatusCodeNotFound,
				Message: "Stale indices",
			},
			Links: []trace.Link{
				{
					TraceID: trace.TraceID{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80},
					SpanID:  trace.SpanID{0xCF, 0xCE, 0xCD, 0xCC, 0xCB, 0xCA, 0xC9, 0xC8},
					Type:    trace.LinkTypeParent,
				},
			},
		},
		{
			SpanContext: trace.SpanContext{
				TraceID: trace.TraceID{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80},
				SpanID:  trace.SpanID{0xCF, 0xCE, 0xCD, 0xCC, 0xCB, 0xCA, 0xC9, 0xC8},
			},
			Name:      "ProxyFetch",
			StartTime: nowPlus10min,
			EndTime:   nowPlus10min2sec,
			Status: trace.Status{
				Code:    trace.StatusCodeInternal,
				Message: "Frontend crash",
			},
			Links: []trace.Link{
				{
					TraceID: trace.TraceID{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80},
					SpanID:  trace.SpanID{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8},
					Type:    trace.LinkTypeChild,
				},
			},
		},
	}

	for _, sd := range spandata {
		jexp.ExportSpan(sd)
	}
	jexp.Flush()

	// Simulate and account for network latency but also the reception process on the server.
	<-time.After(500 * time.Millisecond)

	for i := 0; i < 10; i++ {
		jexp.Flush()
		<-time.After(60 * time.Millisecond)
	}

	got := sink.AllTraces()

	want := []*agenttracepb.ExportTraceServiceRequest{
		{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: "issaTest"},
				LibraryInfo: &commonpb.LibraryInfo{},
				Identifier:  &commonpb.ProcessIdentifier{},
				Attributes: map[string]string{
					"bool":   "true",
					"string": "yes",
					"int64":  "10000000",
				},
			},

			Spans: []*tracepb.Span{
				{
					TraceId:      []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x80},
					SpanId:       []byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8},
					ParentSpanId: []byte{0x1F, 0x1E, 0x1D, 0x1C, 0x1B, 0x1A, 0x19, 0x18},
					Name:         &tracepb.TruncatableString{Value: "DBSearch"},
					StartTime:    internal.TimeToTimestamp(now),
					EndTime:      internal.TimeToTimestamp(nowPlus10min),
					Status: &tracepb.Status{
						Code:    trace.StatusCodeNotFound,
						Message: "Stale indices",
					},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"status.code": {
								Value: &tracepb.AttributeValue_IntValue{IntValue: trace.StatusCodeNotFound},
							},
							"status.message": {
								Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "Stale indices"}},
							},
						},
					},
					Links: &tracepb.Span_Links{
						Link: []*tracepb.Span_Link{
							{
								TraceId: []byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80},
								SpanId:  []byte{0xCF, 0xCE, 0xCD, 0xCC, 0xCB, 0xCA, 0xC9, 0xC8},
								Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
							},
						},
					},
				},
				{
					TraceId:   []byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80},
					SpanId:    []byte{0xCF, 0xCE, 0xCD, 0xCC, 0xCB, 0xCA, 0xC9, 0xC8},
					Name:      &tracepb.TruncatableString{Value: "ProxyFetch"},
					StartTime: internal.TimeToTimestamp(nowPlus10min),
					EndTime:   internal.TimeToTimestamp(nowPlus10min2sec),
					Status: &tracepb.Status{
						Code:    trace.StatusCodeInternal,
						Message: "Frontend crash",
					},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"status.code": {
								Value: &tracepb.AttributeValue_IntValue{IntValue: trace.StatusCodeInternal},
							},
							"status.message": {
								Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "Frontend crash"}},
							},
						},
					},
					Links: &tracepb.Span_Links{
						Link: []*tracepb.Span_Link{
							{
								TraceId: []byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80},
								SpanId:  []byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8},
								// TODO: (@pjanotti, @odeke-em) contact the Jaeger maintains to inquire about
								// Parent_Linked_Spans as currently they've only got:
								// * Child_of
								// * Follows_from
								// yet OpenCensus has Parent too but Jaeger uses a zero-value for LinkCHILD.
								Type: tracepb.Span_Link_PARENT_LINKED_SPAN,
							},
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		gj, wj := testhelper.ToJSON(got), testhelper.ToJSON(want)
		if !bytes.Equal(gj, wj) {
			t.Errorf("Mismatched responses\nGot:\n\t%v\n\t%s\nWant:\n\t%v\n\t%s", got, gj, want, wj)
		}
	}
}
