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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/internal"
	"github.com/open-telemetry/opentelemetry-service/internal/testutils"
	"github.com/open-telemetry/opentelemetry-service/observability/observabilitytest"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver/ocmetrics"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver/octrace"
	"github.com/open-telemetry/opentelemetry-service/receiver/receivertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGrpcGateway_endToEnd(t *testing.T) {
	addr := ":35993"

	// Set the buffer count to 1 to make it flush the test span immediately.
	sink := new(exportertest.SinkTraceExporter)
	ocr, err := New(addr, sink, nil)
	if err != nil {
		t.Fatalf("Failed to create trace receiver: %v", err)
	}

	mh := receivertest.NewMockHost()
	if err := ocr.StartTraceReception(mh); err != nil {
		t.Fatalf("Failed to start trace receiver: %v", err)
	}
	defer ocr.StopTraceReception()

	// TODO(songy23): make starting server deteminisitc
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

func TestTraceGrpcGatewayCors_endToEnd(t *testing.T) {
	addr := ":35991"
	corsOrigins := []string{"allowed-*.com"}

	sink := new(exportertest.SinkTraceExporter)
	ocr, err := New(addr, sink, nil, WithCorsOrigins(corsOrigins))
	if err != nil {
		t.Fatalf("Failed to create trace receiver: %v", err)
	}
	defer ocr.StopTraceReception()

	mh := receivertest.NewMockHost()
	if err := ocr.StartTraceReception(mh); err != nil {
		t.Fatalf("Failed to start trace receiver: %v", err)
	}

	// TODO(songy23): make starting server deteminisitc
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	url := fmt.Sprintf("http://%s/v1/trace", addr)

	// Verify allowed domain gets responses that allow CORS.
	verifyCorsResp(t, url, "allowed-origin.com", 200, true)

	// Verify disallowed domain gets responses that disallow CORS.
	verifyCorsResp(t, url, "disallowed-origin.com", 200, false)
}

func TestMetricsGrpcGatewayCors_endToEnd(t *testing.T) {
	addr := ":35991"
	corsOrigins := []string{"allowed-*.com"}

	sink := new(exportertest.SinkMetricsExporter)
	ocr, err := New(addr, nil, sink, WithCorsOrigins(corsOrigins))
	if err != nil {
		t.Fatalf("Failed to create metrics receiver: %v", err)
	}
	defer ocr.StopMetricsReception()

	mh := receivertest.NewMockHost()
	if err := ocr.StartMetricsReception(mh); err != nil {
		t.Fatalf("Failed to start metrics receiver: %v", err)
	}

	// TODO(songy23): make starting server deteminisitc
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
	defer ocr.StopTraceReception()

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
	ocr.stop()
}

func TestNewPortAlreadyUsed(t *testing.T) {
	addr := testutils.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("failed to listen on %q: %v", addr, err)
	}
	defer ln.Close()

	r, err := New(addr, nil, nil)
	if err == nil {
		t.Fatalf("want err got nil")
	}
	if r != nil {
		t.Fatalf("want nil got %v", r)
	}
}

func TestMultipleStopReceptionShouldNotError(t *testing.T) {
	addr := testutils.GetAvailableLocalAddress(t)
	r, err := New(addr, new(exportertest.SinkTraceExporter), new(exportertest.SinkMetricsExporter))
	require.NoError(t, err)
	require.NotNil(t, r)

	mh := receivertest.NewMockHost()
	require.NoError(t, r.StartTraceReception(mh))
	require.NoError(t, r.StartMetricsReception(mh))

	require.NoError(t, r.StopMetricsReception())
	require.NoError(t, r.StopTraceReception())
}

func TestStartWithoutConsumersShouldFail(t *testing.T) {
	addr := testutils.GetAvailableLocalAddress(t)
	r, err := New(addr, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	mh := receivertest.NewMockHost()
	require.Error(t, r.StartTraceReception(mh))
	require.Error(t, r.StartMetricsReception(mh))

}

func TestOCReceiverTrace_HostIngestionStatusChanges(t *testing.T) {
	type ingestionStateTest struct {
		okToIngest   bool
		expectedCode codes.Code
	}
	tests := []struct {
		name                                       string
		backPressureSetting                        configmodels.BackPressureSetting
		expectedReceivedBatches                    int
		expectedIngestionBlockedRPCs               int
		expectedIngestionBlockedRPCsNoBackPressure int
		ingestionStates                            []ingestionStateTest
	}{
		{
			name:                         "EnableBackPressure",
			backPressureSetting:          configmodels.EnableBackPressure,
			expectedReceivedBatches:      2,
			expectedIngestionBlockedRPCs: 1,
			expectedIngestionBlockedRPCsNoBackPressure: 0,
			ingestionStates: []ingestionStateTest{
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
				{
					okToIngest:   false,
					expectedCode: codes.Unavailable,
				},
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
			},
		},
		{
			name:                         "DisableBackPressure",
			backPressureSetting:          configmodels.DisableBackPressure,
			expectedReceivedBatches:      2,
			expectedIngestionBlockedRPCs: 1,
			expectedIngestionBlockedRPCsNoBackPressure: 1,
			ingestionStates: []ingestionStateTest{
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
				{
					okToIngest:   false,
					expectedCode: codes.OK,
				},
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
			},
		},
	}

	addr := testutils.GetAvailableLocalAddress(t)
	msg := &agenttracepb.ExportTraceServiceRequest{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "test-svc"},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doneFn := observabilitytest.SetupRecordedMetricsTest()
			defer doneFn()

			sink := new(exportertest.SinkTraceExporter)
			host := receivertest.NewMockHost().(*receivertest.MockHost)
			opts := WithTraceReceiverOptions(octrace.WithBackPressureSetting(
				tt.backPressureSetting))
			ocr, err := New(addr, nil, nil, opts)
			require.Nil(t, err)
			require.NotNil(t, ocr)

			ocr.traceConsumer = sink
			err = ocr.StartTraceReception(host)
			require.Nil(t, err)
			defer ocr.StopTraceReception()

			cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				t.Errorf("grpc.Dial: %v", err)
			}
			defer cc.Close()

			for _, ingestionState := range tt.ingestionStates {
				host.SetOkToIngest(ingestionState.okToIngest)

				acc := agenttracepb.NewTraceServiceClient(cc)
				stream, err := acc.Export(context.Background())
				require.NoError(t, err)
				require.NotNil(t, stream)

				err = stream.Send(msg)
				stream.CloseSend()
				if err == nil {
					for {
						if _, err = stream.Recv(); err != nil {
							if err == io.EOF {
								err = nil
							}
							break
						}
					}
				}
				status, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, ingestionState.expectedCode, status.Code())
			}

			require.Equal(t, tt.expectedReceivedBatches, len(sink.AllTraces()))
			require.Nil(
				t,
				observabilitytest.CheckValueViewReceiverReceivedSpans(
					"oc_trace",
					tt.expectedReceivedBatches),
			)
			require.Nil(
				t,
				observabilitytest.CheckValueViewReceiverIngestionBlockedRPCs(
					"oc_trace",
					tt.expectedIngestionBlockedRPCs),
			)

			// This view should only have data if ingestion was blocked and there was no back pressure.
			err = observabilitytest.CheckValueViewReceiverIngestionBlockedRPCsWithDataLoss(
				"oc_trace",
				tt.expectedIngestionBlockedRPCsNoBackPressure)
			if tt.expectedIngestionBlockedRPCsNoBackPressure == 0 {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestOCReceiverMetrics_HostIngestionStatusChanges(t *testing.T) {
	type ingestionStateTest struct {
		okToIngest   bool
		expectedCode codes.Code
	}
	tests := []struct {
		name                                       string
		backPressureSetting                        configmodels.BackPressureSetting
		expectedReceivedBatches                    int
		expectedIngestionBlockedRPCs               int
		expectedIngestionBlockedRPCsNoBackPressure int
		ingestionStates                            []ingestionStateTest
	}{
		{
			name:                         "EnableBackPressure",
			backPressureSetting:          configmodels.EnableBackPressure,
			expectedReceivedBatches:      2,
			expectedIngestionBlockedRPCs: 1,
			expectedIngestionBlockedRPCsNoBackPressure: 0,
			ingestionStates: []ingestionStateTest{
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
				{
					okToIngest:   false,
					expectedCode: codes.Unavailable,
				},
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
			},
		},
		{
			name:                         "DisableBackPressure",
			backPressureSetting:          configmodels.DisableBackPressure,
			expectedReceivedBatches:      2,
			expectedIngestionBlockedRPCs: 1,
			expectedIngestionBlockedRPCsNoBackPressure: 1,
			ingestionStates: []ingestionStateTest{
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
				{
					okToIngest:   false,
					expectedCode: codes.OK,
				},
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
			},
		},
	}

	addr := testutils.GetAvailableLocalAddress(t)
	msg := &agentmetricspb.ExportMetricsServiceRequest{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "test-svc"},
		},
		Metrics: []*metricspb.Metric{
			makeMetric(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doneFn := observabilitytest.SetupRecordedMetricsTest()
			defer doneFn()

			sink := new(exportertest.SinkMetricsExporter)
			host := receivertest.NewMockHost().(*receivertest.MockHost)
			opts := WithMetricsReceiverOptions(ocmetrics.WithBackPressureSetting(
				tt.backPressureSetting))
			ocr, err := New(addr, nil, nil, opts)
			require.Nil(t, err)
			require.NotNil(t, ocr)

			ocr.metricsConsumer = sink
			err = ocr.StartMetricsReception(host)
			require.Nil(t, err)
			defer ocr.StopMetricsReception()

			cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				t.Errorf("grpc.Dial: %v", err)
			}
			defer cc.Close()

			for _, ingestionState := range tt.ingestionStates {
				host.SetOkToIngest(ingestionState.okToIngest)

				acc := agentmetricspb.NewMetricsServiceClient(cc)
				stream, err := acc.Export(context.Background())
				require.NoError(t, err)
				require.NotNil(t, stream)

				err = stream.Send(msg)
				stream.CloseSend()
				if err == nil {
					for {
						if _, err = stream.Recv(); err != nil {
							if err == io.EOF {
								err = nil
							}
							break
						}
					}
				}
				status, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, ingestionState.expectedCode, status.Code())
			}

			// TODO: Add common observability for metric receiver.
			//require.Equal(t, tt.expectedReceivedBatches, len(sink.AllMetrics()))
			//require.Nil(
			//	t,
			//	observabilitytest.CheckValueViewReceiverReceivedMetrics(
			//		"oc_metrics",
			//		tt.expectedReceivedBatches),
			//)
			require.Nil(
				t,
				observabilitytest.CheckValueViewReceiverIngestionBlockedRPCs(
					"oc_metrics",
					tt.expectedIngestionBlockedRPCs),
			)

			// This view should only have data if ingestion was blocked and there was no back pressure.
			err = observabilitytest.CheckValueViewReceiverIngestionBlockedRPCsWithDataLoss(
				"oc_metrics",
				tt.expectedIngestionBlockedRPCsNoBackPressure)
			if tt.expectedIngestionBlockedRPCsNoBackPressure == 0 {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func makeMetric(val int) *metricspb.Metric {
	key := &metricspb.LabelKey{
		Key:         fmt.Sprintf("%s%d", "key", val),
		Description: "label key",
	}
	value := &metricspb.LabelValue{
		Value:    fmt.Sprintf("%s%d", "value", val),
		HasValue: true,
	}

	descriptor := &metricspb.MetricDescriptor{
		Name:        fmt.Sprintf("%s%d", "metric_descriptort_", val),
		Description: "metric descriptor",
		Unit:        "1",
		Type:        metricspb.MetricDescriptor_GAUGE_INT64,
		LabelKeys:   []*metricspb.LabelKey{key},
	}

	now := time.Now().UTC()
	point := &metricspb.Point{
		Timestamp: internal.TimeToTimestamp(now.Add(20 * time.Second)),
		Value: &metricspb.Point_Int64Value{
			Int64Value: int64(val),
		},
	}

	ts := &metricspb.TimeSeries{
		StartTimestamp: internal.TimeToTimestamp(now.Add(-10 * time.Second)),
		LabelValues:    []*metricspb.LabelValue{value},
		Points:         []*metricspb.Point{point},
	}

	return &metricspb.Metric{
		MetricDescriptor: descriptor,
		Timeseries:       []*metricspb.TimeSeries{ts},
	}
}

func TestReceiver_start(t *testing.T) {
	type fields struct {
		host receiver.Host
	}
	type args struct {
		traceConsumer   consumer.TraceConsumer
		metricsConsumer consumer.MetricsConsumer
		host            receiver.Host
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name:    "nil_host",
			wantErr: true,
		},
		{
			name: "different_host",
			args: args{
				host: receivertest.NewMockHost(),
			},
			fields: fields{
				host: receivertest.NewMockHost(),
			},
			wantErr: true,
		},
		{
			name: "no_consumers",
			args: args{
				host: receivertest.NewMockHost(),
			},
			wantErr: true,
		},
		{
			name: "only_trace_consumer",
			args: args{
				traceConsumer: new(exportertest.SinkTraceExporter),
				host:          receivertest.NewMockHost(),
			},
		},
		{
			name: "only_metrics_consumer",
			args: args{
				metricsConsumer: new(exportertest.SinkMetricsExporter),
				host:            receivertest.NewMockHost(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := testutils.GetAvailableLocalAddress(t)
			ocr, _ := New(addr, tt.args.traceConsumer, tt.args.metricsConsumer)
			defer ocr.stop()
			ocr.host = tt.fields.host
			if err := ocr.start(tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("Receiver.start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
