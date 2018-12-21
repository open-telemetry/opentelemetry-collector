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

package opencensus_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/census-instrumentation/opencensus-service/receiver/opencensus"
	"github.com/census-instrumentation/opencensus-service/receiver/testhelper"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus/octrace"
)

func TestGrpcGateway_endToEnd(t *testing.T) {
	addr := ":35990"

	// Set the buffer count to 1 to make it flush the test span immediately.
	ocr, err := opencensus.New(addr, opencensus.WithTraceReceiverOptions(octrace.WithSpanBufferCount(1)))
	if err != nil {
		t.Fatalf("Failed to create trace receiver: %v", err)
	}
	defer ocr.StopTraceReception(context.Background())

	sink := new(testhelper.ConcurrentSpanSink)

	go func() {
		if err := ocr.StartTraceReception(context.Background(), sink); err != nil {
			t.Fatalf("Failed to start trace receiver: %v", err)
		}
	}()

	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/trace", addr)
	traceJSON := []byte(`
    {
       "node":{"identifier":{"hostName":"testHost"}},
       "spans":[
          {
              "traceId":"W47/95gDgQPSabYzgT/GDA==",
              "spanId":"7uGbfsPBsXM=",
              "name":{"value":"testSpan"},
              "startTime":"2018-12-13T14:51:00Z",
              "endTime":"2018-12-13T14:51:01Z",
              "attributes": {
                "attributeMap": {
                  "attr1": {"intValue": "55"}
                }
              }
          }
       ]
    }`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(traceJSON))
	if err != nil {
		t.Fatalf("Error creating trace POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error posting trace to grpc-gateway server: %v", err)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response from trace grpc-gateway, %v", err)
	}
	respStr := string(respBytes)

	err = resp.Body.Close()
	if err != nil {
		t.Errorf("Error closing response body, %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Unexpected status from trace grpc-gateway: %v", resp.StatusCode)
	}

	if respStr != "" {
		t.Errorf("Got unexpected response from trace grpc-gateway: %v", respStr)
	}

	got := sink.AllTraces()

	want := []*agenttracepb.ExportTraceServiceRequest{
		{
			Node: &commonpb.Node{
				Identifier: &commonpb.ProcessIdentifier{HostName: "testHost"},
			},

			Spans: []*tracepb.Span{
				{
					TraceId:   []byte{0x5B, 0x8E, 0xFF, 0xF7, 0x98, 0x3, 0x81, 0x3, 0xD2, 0x69, 0xB6, 0x33, 0x81, 0x3F, 0xC6, 0xC},
					SpanId:    []byte{0xEE, 0xE1, 0x9B, 0x7E, 0xC3, 0xC1, 0xB1, 0x73},
					Name:      &tracepb.TruncatableString{Value: "testSpan"},
					StartTime: internal.TimeToTimestamp(time.Unix(1544712660, 0).UTC()),
					EndTime:   internal.TimeToTimestamp(time.Unix(1544712661, 0).UTC()),
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							"attr1": {
								Value: &tracepb.AttributeValue_IntValue{IntValue: 55},
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
