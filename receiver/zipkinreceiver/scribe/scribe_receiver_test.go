// Copyright 2019, OpenCensus Authors
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

package scribe

import (
	"context"
	"log"
	"net"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/apache/thrift/lib/go/thrift"
	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/omnition/scribe-go/if/scribe/gen-go/scribe"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/processor"
)

func TestNonEqualCategoryIsIgnored(t *testing.T) {
	sink := &mockTraceSink{}
	traceReceiver, err := NewReceiver("", 0, "not-zipkin")
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}

	messages := []*scribe.LogEntry{
		{
			Category: "zipkin",
			Message:  "Shouldn't be parsed, should be just ignored",
		},
	}

	scribeReceiver := traceReceiver.(*scribeReceiver)
	tstatus, err := scribeReceiver.collector.Log(messages)
	if tstatus != scribe.ResultCode_OK {
		t.Errorf("got %v, want scribe.ResultCode_OK", tstatus)
	}
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}
	if len(sink.receivedData) != 0 {
		t.Fatalf("no items should have been captured")
	}
}

func TestScribeReceiverPortAlreadyInUse(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to open a port: %v", err)
	}
	defer l.Close()
	_, portStr, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to convert %s to an int: %v", portStr, err)
	}
	traceReceiver, err := NewReceiver("", uint16(port), "zipkin")
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	err = traceReceiver.StartTraceReception(context.Background(), &mockTraceSink{})
	if err == nil {
		traceReceiver.StopTraceReception(context.Background())
		t.Fatal("conflict on port was expected")
	}
	log.Printf("%v", err)
}

func TestScribeReceiverServer(t *testing.T) {
	const host = ""
	const port = 9410

	traceReceiver, err := NewReceiver(host, port, "zipkin")
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}

	messages := []*scribe.LogEntry{
		{
			Category: "zipkin",
			// Encoded span obtained at https://github.com/openzipkin/zipkin/blob/a40ebcb47986ed1efb85a0543a17611d78404eb2/zipkin-collector/scribe/src/test/java/zipkin2/collector/scribe/ScribeSpanConsumerTest.java#L197
			Message: "CgABq/sBMnzE048LAAMAAAAOZ2V0VHJhY2VzQnlJZHMKAATN0p+4EGfTdAoABav7ATJ8xNOPDwAGDAAAAAQKAAEABR/wq+2DeAsAAgAAAAJzcgwAAwgAAX8AAAEGAAIkwwsAAwAAAAx6aXBraW4tcXVlcnkAAAoAAQAFH/Cr7zj4CwACAAAIAGFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFh\nYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhDAADCAABfwAAAQYAAiTDCwADAAAADHppcGtpbi1xdWVyeQAACgABAAUf8KwLPyILAAIAAABOR2MoOSwwLlBTU2NhdmVuZ2UsMjAxNS0wOS0xNyAxMjozNzowMiArMDAwMCwzMDQubWlsbGlzZWNvbmRzKzc2Mi5taWNyb3NlY29uZHMpDAADCAABfwAAAQYAAiTDCwADAAAADHppcGtpbi1xdWVyeQAIAAQABKZ6AAoAAQAFH/CsDLfACwACAAAAAnNzDAADCAABfwAAAQYAAiTDCwADAAAADHppcGtpbi1xdWVyeQAADwAIDAAAAAULAAEAAAATc3J2L2ZpbmFnbGUudmVyc2lvbgsAAgAAAAY2LjI4LjAIAAMAAAAGDAAECAABfwAAAQYAAgAACwADAAAADHppcGtpbi1xdWVyeQAACwABAAAAD3Nydi9tdXgvZW5hYmxlZAsAAgAAAAEBCAADAAAAAAwABAgAAX8AAAEGAAIAAAsAAwAAAAx6aXBraW4tcXVlcnkAAAsAAQAAAAJzYQsAAgAAAAEBCAADAAAAAAwABAgAAX8AAAEGAAIkwwsAAwAAAAx6aXBraW4tcXVlcnkAAAsAAQAAAAJjYQsAAgAAAAEBCAADAAAAAAwABAgAAX8AAAEGAAL5YAsAAwAAAAx6aXBraW4tcXVlcnkAAAsAAQAAAAZudW1JZHMLAAIAAAAEAAAAAQgAAwAAAAMMAAQIAAF/AAABBgACJMMLAAMAAAAMemlwa2luLXF1ZXJ5AAACAAkAAA==\n",
		},
	}
	sink := newMockTraceSink(len(messages))
	traceReceiver.StartTraceReception(context.Background(), sink)
	if err != nil {
		t.Fatalf("Failed to start trace reception: %v", err)
	}
	defer func() {
		err := traceReceiver.StopTraceReception(context.Background())
		if err != nil {
			t.Fatalf("Error stopping trace reception: %v", err)
		}
	}()

	var trans thrift.TTransport
	trans, err = thrift.NewTSocket(net.JoinHostPort("localhost", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("error creating thrift socket: %v", err)
	}
	trans = thrift.NewTFramedTransport(trans)
	defer trans.Close()

	client := scribe.NewScribeClientFactory(trans, thrift.NewTBinaryProtocolFactoryDefault())
	if err = trans.Open(); err != nil {
		t.Fatalf("error opening thrift socket: %v", err)
	}

	client.Log(messages)
	sink.Wait()

	if len(sink.receivedData) != 1 {
		t.Fatalf("got %d items, want 1 item", len(sink.receivedData))
	}

	if !reflect.DeepEqual(sink.receivedData[0], wantTraceData) {
		t.Errorf("got:\n%+v\nwant:\n%+v\n", sink.receivedData[0], wantTraceData)
	}
}

// TODO: Move this to processortest.
type mockTraceSink struct {
	wg           *sync.WaitGroup
	receivedData []data.TraceData
}

func newMockTraceSink(numReceiveTraceDataCount int) *mockTraceSink {
	wg := &sync.WaitGroup{}
	wg.Add(numReceiveTraceDataCount)
	return &mockTraceSink{
		wg: wg,
	}
}

var _ processor.TraceDataProcessor = (*mockTraceSink)(nil)

func (m *mockTraceSink) ProcessTraceData(ctx context.Context, tracedata data.TraceData) error {
	m.receivedData = append(m.receivedData, tracedata)
	m.wg.Done()
	return nil
}

func (m *mockTraceSink) Wait() {
	m.wg.Wait()
}

var wantTraceData = data.TraceData{
	Node: &commonpb.Node{
		ServiceInfo: &commonpb.ServiceInfo{Name: "zipkin-query"},
		Attributes: map[string]string{
			"ipv4": "127.0.0.1",
			"port": "9411",
		},
	},
	Spans: []*tracepb.Span{
		{
			TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAB, 0xFB, 0x01, 0x32, 0x7C, 0xC4, 0xD3, 0x8F},
			SpanId:       []byte{0xCD, 0xD2, 0x9F, 0xB8, 0x10, 0x67, 0xD3, 0x74},
			ParentSpanId: []byte{0xAB, 0xFB, 0x01, 0x32, 0x7C, 0xC4, 0xD3, 0x8F},
			Name:         &tracepb.TruncatableString{Value: "getTracesByIds"},
			Kind:         tracepb.Span_SERVER,
			StartTime:    &timestamp.Timestamp{Seconds: 1442493420, Nanos: 635000000},
			EndTime:      &timestamp.Timestamp{Seconds: 1442493422, Nanos: 680000000},
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"ca": {
						Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
					},
					"numIds": {
						Value: &tracepb.AttributeValue_IntValue{IntValue: 1},
					},
					"sa": {
						Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
					},
					"srv/finagle.version": {
						Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "6.28.0"}},
					},
					"srv/mux/enabled": {
						Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
					},
				},
			},
			TimeEvents: &tracepb.Span_TimeEvents{
				TimeEvent: []*tracepb.Span_TimeEvent{
					{
						Time: &timestamp.Timestamp{Seconds: 1442493420, Nanos: 635000000},
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Attributes: &tracepb.Span_Attributes{
									AttributeMap: map[string]*tracepb.AttributeValue{
										"sr": {
											Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "zipkin-query"}},
										},
									},
								},
							},
						},
					},
					{
						Time: &timestamp.Timestamp{Seconds: 1442493420, Nanos: 747000000},
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Attributes: &tracepb.Span_Attributes{
									AttributeMap: map[string]*tracepb.AttributeValue{
										"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": {
											Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "zipkin-query"}},
										},
									},
								},
							},
						},
					},
					{
						Time: &timestamp.Timestamp{Seconds: 1442493422, Nanos: 583586000},
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Attributes: &tracepb.Span_Attributes{
									AttributeMap: map[string]*tracepb.AttributeValue{
										"Gc(9,0.PSScavenge,2015-09-17 12:37:02 +0000,304.milliseconds+762.microseconds)": {
											Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "zipkin-query"}},
										},
									},
								},
							},
						},
					},
					{
						Time: &timestamp.Timestamp{Seconds: 1442493422, Nanos: 680000000},
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Attributes: &tracepb.Span_Attributes{
									AttributeMap: map[string]*tracepb.AttributeValue{
										"ss": {
											Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "zipkin-query"}},
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
}
