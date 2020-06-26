// Copyright The OpenTelemetry Authors
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

//lint:file-ignore U1000 t.Skip() flaky test causes unused function warning.

package otlpreceiver

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
	collectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/observability/observabilitytest"
	"go.opentelemetry.io/collector/testutil"
	"go.opentelemetry.io/collector/translator/conventions"
)

const otlpReceiver = "otlp_receiver_test"

func TestGrpcGateway_endToEnd(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	sink := new(exportertest.SinkTraceExporter)
	ocr, err := New(otlpReceiver, "tcp", addr, sink, nil)
	require.NoError(t, err, "Failed to create trace receiver: %v", err)

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver: %v", err)
	defer ocr.Shutdown(context.Background())

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/trace", addr)

	// Verify that CORS is not enabled by default, but that it gives an 405
	// method not allowed error.
	verifyCorsResp(t, url, "origin.com", 405, false)

	traceJSON := []byte(`
	{
	  "resource_spans": [
		{
		  "resource": {
			"attributes": [
			  {
				"key": "host.hostname",
				"value": { "stringValue": "testHost" }
			  }
			]
		  },
		  "instrumentation_library_spans": [
			{
			  "spans": [
				{
				  "trace_id": "W47/95gDgQPSabYzgT/GDA==",
				  "span_id": "7uGbfsPBsXM=",
				  "name": "testSpan",
				  "start_time_unix_nano": 1544712660000000000,
				  "end_time_unix_nano": 1544712661000000000,
				  "attributes": [
					{
					  "key": "attr1",
					  "value": { "intValue": 55 }
					}
				  ]
				}
			  ]
			}
		  ]
		}
	  ]
	}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(traceJSON))
	require.NoError(t, err, "Error creating trace POST request: %v", err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "Error posting trace to grpc-gateway server: %v", err)

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

	if respStr != "{}" {
		t.Errorf("Got unexpected response from trace grpc-gateway: %v", respStr)
	}

	got := sink.AllTraces()[0]

	want := pdata.TracesFromOtlp([]*otlptrace.ResourceSpans{
		{
			Resource: &otlpresource.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   conventions.AttributeHostHostname,
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "testHost"}},
					},
				},
			},
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{
						{
							TraceId:           []byte{0x5B, 0x8E, 0xFF, 0xF7, 0x98, 0x3, 0x81, 0x3, 0xD2, 0x69, 0xB6, 0x33, 0x81, 0x3F, 0xC6, 0xC},
							SpanId:            []byte{0xEE, 0xE1, 0x9B, 0x7E, 0xC3, 0xC1, 0xB1, 0x73},
							Name:              "testSpan",
							StartTimeUnixNano: 1544712660000000000,
							EndTimeUnixNano:   1544712661000000000,
							Attributes: []*otlpcommon.KeyValue{
								{
									Key:   "attr1",
									Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: 55}},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.EqualValues(t, got, want)
}

func TestProtoHttp(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	sink := new(exportertest.SinkTraceExporter)
	ocr, err := New(otlpReceiver, "tcp", addr, sink, nil)
	require.NoError(t, err, "Failed to create trace receiver: %v", err)

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver: %v", err)
	defer ocr.Shutdown(context.Background())

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/trace", addr)

	wantOtlp := pdata.TracesToOtlp(testdata.GenerateTraceDataOneSpan())

	traceProto := collectortrace.ExportTraceServiceRequest{
		ResourceSpans: wantOtlp,
	}
	traceBytes, err := proto.Marshal(&traceProto)
	if err != nil {
		t.Errorf("Error marshaling protobuf: %v", err)
	}

	buf := bytes.NewBuffer(traceBytes)

	req, err := http.NewRequest("POST", url, buf)
	require.NoError(t, err, "Error creating trace POST request: %v", err)
	req.Header.Set("Content-Type", "application/x-protobuf")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "Error posting trace to grpc-gateway server: %v", err)

	respBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "Error reading response from trace grpc-gateway")

	err = resp.Body.Close()
	require.NoError(t, err, "Error closing response body")

	if resp.StatusCode != 200 {
		t.Errorf("Unexpected status from trace grpc-gateway: %v", resp.StatusCode)
	}

	if resType := resp.Header.Get("Content-Type"); resType != "application/x-protobuf" {
		t.Errorf("response Content-Type got: %s, want: %s", resType, "application/x-protobuf")
	}

	tmp := collectortrace.ExportTraceServiceResponse{}
	err = proto.Unmarshal(respBytes, &tmp)
	require.NoError(t, err, "Unable to unmarshal response to ExportTraceServiceResponse proto")

	gotOtlp := pdata.TracesToOtlp(sink.AllTraces()[0])

	if len(gotOtlp) != len(wantOtlp) {
		t.Fatalf("len(traces):\nGot: %d\nWant: %d\n", len(gotOtlp), len(wantOtlp))
	}

	got := gotOtlp[0]
	want := wantOtlp[0]

	// assert.Equal doesn't work on protos, see:
	// https://github.com/stretchr/testify/issues/758
	if !proto.Equal(got, want) {
		t.Errorf("Sending trace proto over http failed\nGot:\n%v\nWant:\n%v\n",
			proto.MarshalTextString(got),
			proto.MarshalTextString(want))
	}

}

func TestTraceGrpcGatewayCors_endToEnd(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	corsOrigins := []string{"allowed-*.com"}

	sink := new(exportertest.SinkTraceExporter)
	ocr, err := New(otlpReceiver, "tcp", addr, sink, nil, WithCorsOrigins(corsOrigins))
	require.NoError(t, err, "Failed to create trace receiver: %v", err)
	defer ocr.Shutdown(context.Background())

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver: %v", err)

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/trace", addr)

	// Verify allowed domain gets responses that allow CORS.
	verifyCorsResp(t, url, "allowed-origin.com", 200, true)

	// Verify disallowed domain gets responses that disallow CORS.
	verifyCorsResp(t, url, "disallowed-origin.com", 200, false)
}

func TestMetricsGrpcGatewayCors_endToEnd(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	corsOrigins := []string{"allowed-*.com"}

	sink := new(exportertest.SinkMetricsExporter)
	ocr, err := New(otlpReceiver, "tcp", addr, nil, sink, WithCorsOrigins(corsOrigins))
	require.NoError(t, err, "Failed to create metrics receiver: %v", err)
	defer ocr.Shutdown(context.Background())

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start metrics receiver: %v", err)

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/metrics", addr)

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

	addr := testutil.GetAvailableLocalAddress(t)
	cbts := new(exportertest.SinkTraceExporter)
	ocr, err := New(otlpReceiver, "tcp", addr, cbts, nil)
	require.NoError(t, err, "Failed to create trace receiver: %v", err)

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start the trace receiver: %v", err)
	defer ocr.Shutdown(context.Background())

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

	acc := collectortrace.NewTraceServiceClient(cc)

	req := &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.KeyValue{
						{
							Key:   "sub-type",
							Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: contentType}},
						},
					},
				},
				InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
					{
						Spans: []*otlptrace.Span{
							{
								TraceId: []byte{
									0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
									0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = acc.Export(context.Background(), req)
	return err
}

func verifyCorsResp(t *testing.T, url string, origin string, wantStatus int, wantAllowed bool) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	require.NoError(t, err, "Error creating trace OPTIONS request: %v", err)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Error sending OPTIONS to http server: %v", err)

	err = resp.Body.Close()
	if err != nil {
		t.Errorf("Error closing OPTIONS response body, %v", err)
	}

	assert.Equal(t, wantStatus, resp.StatusCode)

	gotAllowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	gotAllowMethods := resp.Header.Get("Access-Control-Allow-Methods")

	wantAllowOrigin := ""
	wantAllowMethods := ""
	if wantAllowed {
		wantAllowOrigin = origin
		wantAllowMethods = "POST"
	}
	assert.Equal(t, wantAllowOrigin, gotAllowOrigin)
	assert.Equal(t, wantAllowMethods, gotAllowMethods)
}

func TestStopWithoutStartNeverCrashes(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ocr, err := New(otlpReceiver, "tcp", addr, nil, nil)
	require.NoError(t, err, "Failed to create an OpenCensus receiver: %v", err)
	// Stop it before ever invoking Start*.
	ocr.stop()
}

func TestNewPortAlreadyUsed(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to listen on %q: %v", addr, err)
	defer ln.Close()

	r, err := New(otlpReceiver, "tcp", addr, nil, nil)
	require.Error(t, err)
	require.Nil(t, r)
}

func TestMultipleStopReceptionShouldNotError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	r, err := New(otlpReceiver, "tcp", addr, new(exportertest.SinkTraceExporter), new(exportertest.SinkMetricsExporter))
	require.NoError(t, err)
	require.NotNil(t, r)

	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, r.Shutdown(context.Background()))
}

func TestStartWithoutConsumersShouldFail(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	r, err := New(otlpReceiver, "tcp", addr, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

func tempSocketName(t *testing.T) string {
	tmpfile, err := ioutil.TempFile("", "sock")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())
	socket := tmpfile.Name()
	require.NoError(t, os.Remove(socket))
	return socket
}

func TestReceiveOnUnixDomainSocket_endToEnd(t *testing.T) {
	socketName := tempSocketName(t)
	cbts := new(exportertest.SinkTraceExporter)
	r, err := New(otlpReceiver, "unix", socketName, cbts, nil)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
	defer r.Shutdown(context.Background())

	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	span := `
	{
	  "resource_spans": [
		{
		  "instrumentation_library_spans": [
			{
			  "spans": [
				{
				  "trace_id": "YpsR8/le4OgjwSSxhjlrEg==",
				  "span_id": "2CogcbJh7Ko=",
				  "name": "testSpan",
				  "start_time_unix_nano": 1544712660000000000,
				  "end_time_unix_nano": 1544712661000000000
				}
			  ]
			}
		  ]
		}
	  ]
	}`

	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
				return net.Dial("unix", socketName)
			},
		},
	}

	response, err := c.Post("http://unix/v1/trace", "application/json", strings.NewReader(span))
	require.NoError(t, err)
	defer response.Body.Close()

	bodyBytes, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	bodyString := string(bodyBytes)
	fmt.Println(bodyString)

	require.Equal(t, 200, response.StatusCode)
}

// TestOTLPReceiverTrace_HandleNextConsumerResponse checks if the trace receiver
// is returning the proper response (return and metrics) when the next consumer
// in the pipeline reports error. The test changes the responses returned by the
// next trace consumer, checks if data was passed down the pipeline and if
// proper metrics were recorded. It also uses all endpoints supported by the
// trace receiver.
func TestOTLPReceiverTrace_HandleNextConsumerResponse(t *testing.T) {
	type ingestionStateTest struct {
		okToIngest   bool
		expectedCode codes.Code
	}
	tests := []struct {
		name                         string
		expectedReceivedBatches      int
		expectedIngestionBlockedRPCs int
		ingestionStates              []ingestionStateTest
	}{
		{
			name:                         "IngestTest",
			expectedReceivedBatches:      2,
			expectedIngestionBlockedRPCs: 1,
			ingestionStates: []ingestionStateTest{
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
				{
					okToIngest:   false,
					expectedCode: codes.Unknown,
				},
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
			},
		},
	}

	addr := testutil.GetAvailableLocalAddress(t)
	req := &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.KeyValue{
						{
							Key:   conventions.AttributeServiceName,
							Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test-svc"}},
						},
					},
				},
				InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
					{
						Spans: []*otlptrace.Span{
							{
								TraceId: []byte{
									0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
									0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
								},
							},
						},
					},
				},
			},
		},
	}

	exportBidiFn := func(
		t *testing.T,
		cc *grpc.ClientConn,
		msg *collectortrace.ExportTraceServiceRequest) error {

		acc := collectortrace.NewTraceServiceClient(cc)
		_, err := acc.Export(context.Background(), req)

		return err
	}

	exporters := []struct {
		receiverTag string
		exportFn    func(
			t *testing.T,
			cc *grpc.ClientConn,
			msg *collectortrace.ExportTraceServiceRequest) error
	}{
		{
			receiverTag: "otlp_trace",
			exportFn:    exportBidiFn,
		},
	}
	for _, exporter := range exporters {
		for _, tt := range tests {
			t.Run(tt.name+"/"+exporter.receiverTag, func(t *testing.T) {
				doneFn := observabilitytest.SetupRecordedMetricsTest()
				defer doneFn()

				sink := new(exportertest.SinkTraceExporter)

				var opts []Option
				ocr, err := New(otlpReceiver, "tcp", addr, nil, nil, opts...)
				require.Nil(t, err)
				require.NotNil(t, ocr)

				ocr.traceConsumer = sink
				require.Nil(t, ocr.Start(context.Background(), componenttest.NewNopHost()))
				defer ocr.Shutdown(context.Background())

				cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
				if err != nil {
					t.Errorf("grpc.Dial: %v", err)
				}
				defer cc.Close()

				for _, ingestionState := range tt.ingestionStates {
					if ingestionState.okToIngest {
						sink.SetConsumeTraceError(nil)
					} else {
						sink.SetConsumeTraceError(fmt.Errorf("%q: consumer error", tt.name))
					}

					err = exporter.exportFn(t, cc, req)

					status, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, ingestionState.expectedCode, status.Code())
				}

				require.Equal(t, tt.expectedReceivedBatches, len(sink.AllTraces()))
				require.Nil(
					t,
					observabilitytest.CheckValueViewReceiverReceivedSpans(
						exporter.receiverTag,
						tt.expectedReceivedBatches),
				)
			})
		}
	}
}
