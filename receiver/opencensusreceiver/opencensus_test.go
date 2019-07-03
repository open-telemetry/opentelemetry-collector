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

package opencensusreceiver

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/internal"
	"github.com/open-telemetry/opentelemetry-service/receiver/receivertest"
)

func TestGrpcGateway_endToEnd(t *testing.T) {
	addr := ":35993"

	// Set the buffer count to 1 to make it flush the test span immediately.
	sink := new(exportertest.SinkTraceExporter)
	ocr, err := New(addr, sink, nil)
	if err != nil {
		t.Fatalf("Failed to create trace receiver: %v", err)
	}
	defer ocr.StopTraceReception()

	mh := receivertest.NewMockHost()
	go func() {
		if err := ocr.StartTraceReception(mh); err != nil {
			t.Fatalf("Failed to start trace receiver: %v", err)
		}
	}()

	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/trace", addr)

	// Verify that CORS is not enabled by default, but that it gives an 405
	// method not allowed error.
	verifyCorsResp(t, url, "origin.com", 405, false)

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

	want := []consumerdata.TraceData{
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
			SourceFormat: "oc_trace",
		},
	}

	if !reflect.DeepEqual(got, want) {
		gj, wj := exportertest.ToJSON(got), exportertest.ToJSON(want)
		if !bytes.Equal(gj, wj) {
			t.Errorf("Mismatched responses\nGot:\n\t%v\n\t%s\nWant:\n\t%v\n\t%s", got, gj, want, wj)
		}
	}
}

func TestGrpcGatewayCors_endToEnd(t *testing.T) {
	addr := ":35991"
	corsOrigins := []string{"allowed-*.com"}

	sink := new(exportertest.SinkTraceExporter)
	ocr, err := New(addr, sink, nil, WithCorsOrigins(corsOrigins))
	if err != nil {
		t.Fatalf("Failed to create trace receiver: %v", err)
	}
	defer ocr.Stop()

	mh := receivertest.NewMockHost()
	go func() {
		if err := ocr.StartTraceReception(mh); err != nil {
			t.Fatalf("Failed to start trace receiver: %v", err)
		}
	}()

	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/trace", addr)

	// Verify allowed domain gets responses that allow CORS.
	verifyCorsResp(t, url, "allowed-origin.com", 200, true)

	// Verify disallowed domain gets responses that disallow CORS.
	verifyCorsResp(t, url, "disallowed-origin.com", 200, false)
}

// As per Issue https://github.com/census-instrumentation/opencensus-service/issues/366
// the agent's mux should be able to accept all Proto affiliated content-types and not
// redirect them to the web-grpc-gateway endpoint.
func TestAcceptAllGRPCProtoAffiliatedContentTypes(t *testing.T) {
	t.Skip("Currently a flaky test as we need a way to flush all written traces")

	addr := ":35991"
	cbts := new(exportertest.SinkTraceExporter)
	ocr, err := New(addr, cbts, nil)
	if err != nil {
		t.Fatalf("Failed to create trace receiver: %v", err)
	}

	mh := receivertest.NewMockHost()
	if err := ocr.StartTraceReception(mh); err != nil {
		t.Fatalf("Failed to start the trace receiver: %v", err)
	}
	defer ocr.Stop()

	// Now start the client with the various Proto affiliated gRPC Content-SubTypes as per:
	//      https://godoc.org/google.golang.org/grpc#CallContentSubtype
	protoAffiliatedContentSubTypes := []string{"", "proto"}
	for _, subContentType := range protoAffiliatedContentSubTypes {
		if err := runContentTypeTests(addr, asSubContentType, subContentType); err != nil {
			t.Errorf("%q subContentType failed to send proto: %v", subContentType, err)
		}
	}

	// Now start the client with the various Proto affiliated gRPC Content-Types,
	// as we encountered in https://github.com/census-instrumentation/opencensus-service/issues/366
	protoAffiliatedContentTypes := []string{"application/grpc", "application/grpc+proto"}
	for _, contentType := range protoAffiliatedContentTypes {
		if err := runContentTypeTests(addr, asContentType, contentType); err != nil {
			t.Errorf("%q Content-type failed to send proto: %v", contentType, err)
		}
	}

	// Before we exit we have to verify that we got exactly 4 TraceService requests.
	wantLen := len(protoAffiliatedContentSubTypes) + len(protoAffiliatedContentTypes)
	gotReqs := cbts.AllTraces()
	if len(gotReqs) != wantLen {
		t.Errorf("Receiver ExportTraceServiceRequest length mismatch:: Got %d Want %d", len(gotReqs), wantLen)
	}
}

const (
	asSubContentType = true
	asContentType    = false
)

func runContentTypeTests(addr string, contentTypeDesignation bool, contentType string) error {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDisableRetry(),
	}

	if contentTypeDesignation == asContentType {
		opts = append(opts, grpc.WithDefaultCallOptions(
			grpc.Header(&metadata.MD{"Content-Type": []string{contentType}})))
	} else {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.CallContentSubtype(contentType)))
	}

	cc, err := grpc.Dial(addr, opts...)
	if err != nil {
		return fmt.Errorf("Creating grpc.ClientConn: %v", err)
	}
	defer cc.Close()

	// First step is to send the Node.
	acc := agenttracepb.NewTraceServiceClient(cc)

	stream, err := acc.Export(context.Background())
	if err != nil {
		return fmt.Errorf("Initializing the export stream: %v", err)
	}

	msg := &agenttracepb.ExportTraceServiceRequest{
		Node: &commonpb.Node{
			Attributes: map[string]string{
				"sub-type": contentType,
			},
		},
		Spans: []*tracepb.Span{
			{
				TraceId: []byte{
					0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
					0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
				},
			},
		},
	}
	return stream.Send(msg)
}

func verifyCorsResp(t *testing.T, url string, origin string, wantStatus int, wantAllowed bool) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		t.Fatalf("Error creating trace OPTIONS request: %v", err)
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error sending OPTIONS to grpc-gateway server: %v", err)
	}

	err = resp.Body.Close()
	if err != nil {
		t.Errorf("Error closing OPTIONS response body, %v", err)
	}

	if resp.StatusCode != wantStatus {
		t.Errorf("Unexpected status from OPTIONS: %v", resp.StatusCode)
	}

	gotAllowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	gotAllowMethods := resp.Header.Get("Access-Control-Allow-Methods")

	wantAllowOrigin := ""
	wantAllowMethods := ""
	if wantAllowed {
		wantAllowOrigin = origin
		wantAllowMethods = "POST"
	}

	if gotAllowOrigin != wantAllowOrigin {
		t.Errorf("Unexpected Access-Control-Allow-Origin: %v", gotAllowOrigin)
	}
	if gotAllowMethods != wantAllowMethods {
		t.Errorf("Unexpected Access-Control-Allow-Methods: %v", gotAllowMethods)
	}
}

// Issue #379: Invoking Stop on an unstarted OpenCensus receiver should never crash.
func TestStopWithoutStartNeverCrashes(t *testing.T) {
	ocr, err := New(":55444", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create an OpenCensus receiver: %v", err)
	}
	// Stop it before ever invoking Start*.
	ocr.Stop()
}
