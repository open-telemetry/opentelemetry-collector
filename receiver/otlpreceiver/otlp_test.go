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

package otlpreceiver

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	collectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/collector/internal/data/protogen/common/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/protogen/resource/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/internal/internalconsumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/testutil"
	"go.opentelemetry.io/collector/translator/conventions"
)

const otlpReceiverName = "otlp_receiver_test"

var traceJSON = []byte(`
	{
	  "resource_spans": [
		{
		  "resource": {
			"attributes": [
			  {
				"key": "host.name",
				"value": { "stringValue": "testHost" }
			  }
			]
		  },
		  "instrumentation_library_spans": [
			{
			  "spans": [
				{
				  "trace_id": "5B8EFFF798038103D269B633813FC60C",
				  "span_id": "EEE19B7EC3C1B173",
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

var resourceSpansOtlp = otlptrace.ResourceSpans{

	Resource: otlpresource.Resource{
		Attributes: []otlpcommon.KeyValue{
			{
				Key:   conventions.AttributeHostName,
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "testHost"}},
			},
		},
	},
	InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
		{
			Spans: []*otlptrace.Span{
				{
					TraceId:           data.NewTraceID([16]byte{0x5B, 0x8E, 0xFF, 0xF7, 0x98, 0x3, 0x81, 0x3, 0xD2, 0x69, 0xB6, 0x33, 0x81, 0x3F, 0xC6, 0xC}),
					SpanId:            data.NewSpanID([8]byte{0xEE, 0xE1, 0x9B, 0x7E, 0xC3, 0xC1, 0xB1, 0x73}),
					Name:              "testSpan",
					StartTimeUnixNano: 1544712660000000000,
					EndTimeUnixNano:   1544712661000000000,
					Attributes: []otlpcommon.KeyValue{
						{
							Key:   "attr1",
							Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: 55}},
						},
					},
				},
			},
		},
	},
}

var traceOtlp = pdata.TracesFromOtlp([]*otlptrace.ResourceSpans{&resourceSpansOtlp})

func TestJsonHttp(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		err      error
	}{
		{
			name:     "JSONUncompressed",
			encoding: "",
		},
		{
			name:     "JSONGzipCompressed",
			encoding: "gzip",
		},
		{
			name:     "NotGRPCError",
			encoding: "",
			err:      errors.New("my error"),
		},
		{
			name:     "GRPCError",
			encoding: "",
			err:      status.New(codes.Internal, "").Err(),
		},
	}
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	sink := &internalconsumertest.ErrOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}
	ocr := newHTTPReceiver(t, addr, sink, nil)

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	defer ocr.Shutdown(context.Background())

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	// Previously we used /v1/trace as the path. The correct path according to OTLP spec
	// is /v1/traces. We currently support both on the receiving side to give graceful
	// period for senders to roll out a fix, so we test for both paths to make sure
	// the receiver works correctly.
	targetURLPaths := []string{"/v1/trace", "/v1/traces"}

	for _, test := range tests {
		for _, targetURLPath := range targetURLPaths {
			t.Run(test.name+targetURLPath, func(t *testing.T) {
				url := fmt.Sprintf("http://%s%s", addr, targetURLPath)
				sink.Reset()
				testHTTPJSONRequest(t, url, sink, test.encoding, test.err)
			})
		}
	}
}

func testHTTPJSONRequest(t *testing.T, url string, sink *internalconsumertest.ErrOrSinkConsumer, encoding string, expectedErr error) {
	var buf *bytes.Buffer
	var err error
	switch encoding {
	case "gzip":
		buf, err = compressGzip(traceJSON)
		require.NoError(t, err, "Error while gzip compressing trace: %v", err)
	default:
		buf = bytes.NewBuffer(traceJSON)
	}
	sink.SetConsumeError(expectedErr)
	req, err := http.NewRequest("POST", url, buf)
	require.NoError(t, err, "Error creating trace POST request: %v", err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", encoding)

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

	allTraces := sink.AllTraces()
	if expectedErr == nil {
		assert.Equal(t, 200, resp.StatusCode)
		var respJSON map[string]interface{}
		assert.NoError(t, json.Unmarshal([]byte(respStr), &respJSON))
		assert.Len(t, respJSON, 0, "Got unexpected response from trace grpc-gateway")

		require.Len(t, allTraces, 1)

		got := allTraces[0]
		assert.EqualValues(t, got, traceOtlp)
	} else {
		errStatus := &spb.Status{}
		assert.NoError(t, json.Unmarshal([]byte(respStr), errStatus))
		if s, ok := status.FromError(expectedErr); ok {
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.True(t, proto.Equal(errStatus, s.Proto()))
		} else {
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.True(t, proto.Equal(errStatus, &spb.Status{Code: int32(codes.Unknown), Message: "my error"}))
		}
		require.Len(t, allTraces, 0)
	}

}

func TestJsonMarshaling(t *testing.T) {
	m := jsonpb.Marshaler{}
	json, err := m.MarshalToString(&resourceSpansOtlp)
	assert.NoError(t, err)

	var resourceSpansOtlp2 otlptrace.ResourceSpans
	err = jsonpb.UnmarshalString(json, &resourceSpansOtlp2)
	assert.NoError(t, err)

	assert.EqualValues(t, resourceSpansOtlp, resourceSpansOtlp2)
}

func TestJsonUnmarshaling(t *testing.T) {
	var resourceSpansOtlp2 otlptrace.ResourceSpans
	err := jsonpb.UnmarshalString(`
		{
		  "instrumentation_library_spans": [
			{
			  "spans": [
				{
				}
			  ]
			}
		  ]
		}`, &resourceSpansOtlp2)
	assert.NoError(t, err)
	assert.EqualValues(t, data.TraceID{}, resourceSpansOtlp2.InstrumentationLibrarySpans[0].Spans[0].TraceId)

	tests := []struct {
		name  string
		json  string
		bytes [16]byte
	}{
		{
			name:  "empty string trace id",
			json:  `""`,
			bytes: [16]byte{},
		},
		{
			name:  "zero bytes trace id",
			json:  `"00000000000000000000000000000000"`,
			bytes: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var resourceSpansOtlp2 otlptrace.ResourceSpans
			jsonStr := fmt.Sprintf(`
			{
			  "instrumentation_library_spans": [
				{
				  "spans": [
					{
					  "trace_id": %v
					}
				  ]
				}
			  ]
			}`, test.json)
			err := jsonpb.UnmarshalString(jsonStr, &resourceSpansOtlp2)
			assert.NoError(t, err)
			assert.EqualValues(t, data.NewTraceID(test.bytes), resourceSpansOtlp2.InstrumentationLibrarySpans[0].Spans[0].TraceId)
		})
	}
}

func TestProtoHttp(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		err      error
	}{
		{
			name:     "ProtoUncompressed",
			encoding: "",
		},
		{
			name:     "ProtoGzipCompressed",
			encoding: "gzip",
		},
		{
			name:     "NotGRPCError",
			encoding: "",
			err:      errors.New("my error"),
		},
		{
			name:     "GRPCError",
			encoding: "",
			err:      status.New(codes.Internal, "").Err(),
		},
	}
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	tSink := &internalconsumertest.ErrOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}
	ocr := newHTTPReceiver(t, addr, tSink, consumertest.NewMetricsNop())

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	defer ocr.Shutdown(context.Background())

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	wantOtlp := pdata.TracesToOtlp(testdata.GenerateTraceDataOneSpan())
	traceProto := collectortrace.ExportTraceServiceRequest{
		ResourceSpans: wantOtlp,
	}
	traceBytes, err := traceProto.Marshal()
	if err != nil {
		t.Errorf("Error marshaling protobuf: %v", err)
	}

	// Previously we used /v1/trace as the path. The correct path according to OTLP spec
	// is /v1/traces. We currently support both on the receiving side to give graceful
	// period for senders to roll out a fix, so we test for both paths to make sure
	// the receiver works correctly.
	targetURLPaths := []string{"/v1/trace", "/v1/traces"}

	for _, test := range tests {
		for _, targetURLPath := range targetURLPaths {
			t.Run(test.name+targetURLPath, func(t *testing.T) {
				url := fmt.Sprintf("http://%s%s", addr, targetURLPath)
				tSink.Reset()
				testHTTPProtobufRequest(t, url, tSink, test.encoding, traceBytes, test.err, wantOtlp)
			})
		}
	}
}

func createHTTPProtobufRequest(
	t *testing.T,
	url string,
	encoding string,
	traceBytes []byte,
) *http.Request {
	var buf *bytes.Buffer
	var err error
	switch encoding {
	case "gzip":
		buf, err = compressGzip(traceBytes)
		require.NoError(t, err, "Error while gzip compressing trace: %v", err)
	default:
		buf = bytes.NewBuffer(traceBytes)
	}
	req, err := http.NewRequest("POST", url, buf)
	require.NoError(t, err, "Error creating trace POST request: %v", err)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", encoding)
	return req
}

func testHTTPProtobufRequest(
	t *testing.T,
	url string,
	tSink *internalconsumertest.ErrOrSinkConsumer,
	encoding string,
	traceBytes []byte,
	expectedErr error,
	wantOtlp []*otlptrace.ResourceSpans,
) {
	tSink.SetConsumeError(expectedErr)

	req := createHTTPProtobufRequest(t, url, encoding, traceBytes)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "Error posting trace to grpc-gateway server: %v", err)

	respBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "Error reading response from trace grpc-gateway")
	require.NoError(t, resp.Body.Close(), "Error closing response body")

	require.Equal(t, "application/x-protobuf", resp.Header.Get("Content-Type"), "Unexpected response Content-Type")

	allTraces := tSink.AllTraces()

	if expectedErr == nil {
		require.Equal(t, 200, resp.StatusCode, "Unexpected return status")
		tmp := &collectortrace.ExportTraceServiceResponse{}
		err := tmp.Unmarshal(respBytes)
		require.NoError(t, err, "Unable to unmarshal response to ExportTraceServiceResponse proto")

		require.Len(t, allTraces, 1)

		gotOtlp := pdata.TracesToOtlp(allTraces[0])

		if len(gotOtlp) != len(wantOtlp) {
			t.Fatalf("len(traces):\nGot: %d\nWant: %d\n", len(gotOtlp), len(wantOtlp))
		}

		got := gotOtlp[0]
		want := wantOtlp[0]

		if !assert.EqualValues(t, got, want) {
			t.Errorf("Sending trace proto over http failed\nGot:\n%v\nWant:\n%v\n",
				got.String(),
				want.String())
		}
	} else {
		errStatus := &spb.Status{}
		assert.NoError(t, proto.Unmarshal(respBytes, errStatus))
		if s, ok := status.FromError(expectedErr); ok {
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.True(t, proto.Equal(errStatus, s.Proto()))
		} else {
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.True(t, proto.Equal(errStatus, &spb.Status{Code: int32(codes.Unknown), Message: "my error"}))
		}
		require.Len(t, allTraces, 0)
	}
}

func TestOTLPReceiverInvalidContentEncoding(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		encoding    string
		reqBodyFunc func() (*bytes.Buffer, error)
		resBodyFunc func() ([]byte, error)
		status      int
	}{
		{
			name:     "JsonGzipUncompressed",
			content:  "application/json",
			encoding: "gzip",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return bytes.NewBuffer([]byte(`{"key": "value"}`)), nil
			},
			resBodyFunc: func() ([]byte, error) {
				return json.Marshal(status.New(codes.InvalidArgument, "gzip: invalid header").Proto())
			},
			status: 400,
		},
		{
			name:     "ProtoGzipUncompressed",
			content:  "application/x-protobuf",
			encoding: "gzip",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return bytes.NewBuffer([]byte(`{"key": "value"}`)), nil
			},
			resBodyFunc: func() ([]byte, error) {
				return proto.Marshal(status.New(codes.InvalidArgument, "gzip: invalid header").Proto())
			},
			status: 400,
		},
	}
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	tSink := new(consumertest.TracesSink)
	mSink := new(consumertest.MetricsSink)
	ocr := newHTTPReceiver(t, addr, tSink, mSink)

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	defer ocr.Shutdown(context.Background())

	url := fmt.Sprintf("http://%s/v1/traces", addr)

	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := test.reqBodyFunc()
			require.NoError(t, err, "Error creating request body: %v", err)

			req, err := http.NewRequest("POST", url, body)
			require.NoError(t, err, "Error creating trace POST request: %v", err)
			req.Header.Set("Content-Type", test.content)
			req.Header.Set("Content-Encoding", test.encoding)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err, "Error posting trace to grpc-gateway server: %v", err)

			respBytes, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err, "Error reading response from trace grpc-gateway")
			exRespBytes, err := test.resBodyFunc()
			require.NoError(t, err, "Error creating expecting response body")
			require.NoError(t, resp.Body.Close(), "Error closing response body")

			require.Equal(t, test.status, resp.StatusCode, "Unexpected return status")
			require.Equal(t, test.content, resp.Header.Get("Content-Type"), "Unexpected response Content-Type")
			require.Equal(t, exRespBytes, respBytes, "Unexpected response content")
		})
	}
}

func TestGRPCNewPortAlreadyUsed(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to listen on %q: %v", addr, err)
	defer ln.Close()

	r := newGRPCReceiver(t, otlpReceiverName, addr, new(consumertest.TracesSink), new(consumertest.MetricsSink))
	require.NotNil(t, r)

	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

func TestHTTPNewPortAlreadyUsed(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to listen on %q: %v", addr, err)
	defer ln.Close()

	r := newHTTPReceiver(t, addr, new(consumertest.TracesSink), new(consumertest.MetricsSink))
	require.NotNil(t, r)

	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

func TestGRPCStartWithoutConsumers(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	r := newGRPCReceiver(t, otlpReceiverName, addr, nil, nil)
	require.NotNil(t, r)
	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

func TestHTTPStartWithoutConsumers(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	r := newHTTPReceiver(t, addr, nil, nil)
	require.NotNil(t, r)
	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

func createSingleSpanTrace() *collectortrace.ExportTraceServiceRequest {
	return &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(testdata.GenerateTraceDataOneSpan()),
	}
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
	req := createSingleSpanTrace()

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
				doneFn, err := obsreporttest.SetupRecordedMetricsTest()
				require.NoError(t, err)
				defer doneFn()

				sink := &internalconsumertest.ErrOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}

				ocr := newGRPCReceiver(t, exporter.receiverTag, addr, sink, nil)
				require.NotNil(t, ocr)
				require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()))
				defer ocr.Shutdown(context.Background())

				cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
				require.NoError(t, err)
				defer cc.Close()

				for _, ingestionState := range tt.ingestionStates {
					if ingestionState.okToIngest {
						sink.SetConsumeError(nil)
					} else {
						sink.SetConsumeError(fmt.Errorf("%q: consumer error", tt.name))
					}

					err = exporter.exportFn(t, cc, req)

					status, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, ingestionState.expectedCode, status.Code())
				}

				require.Equal(t, tt.expectedReceivedBatches, len(sink.AllTraces()))

				obsreporttest.CheckReceiverTracesViews(t, exporter.receiverTag, "grpc", int64(tt.expectedReceivedBatches), int64(tt.expectedIngestionBlockedRPCs))
			})
		}
	}
}

func TestGRPCInvalidTLSCredentials(t *testing.T) {
	cfg := &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: "IncorrectTLS",
		},
		Protocols: Protocols{
			GRPC: &configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  testutil.GetAvailableLocalAddress(t),
					Transport: "tcp",
				},
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "willfail",
					},
				},
			},
		},
	}

	// TLS is resolved during Creation of the receiver for GRPC.
	_, err := createReceiver(cfg, zap.NewNop())
	assert.EqualError(t, err,
		`failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither`)
}

func TestHTTPInvalidTLSCredentials(t *testing.T) {
	cfg := &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: "IncorrectTLS",
		},
		Protocols: Protocols{
			HTTP: &confighttp.HTTPServerSettings{
				Endpoint: testutil.GetAvailableLocalAddress(t),
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "willfail",
					},
				},
			},
		},
	}

	// TLS is resolved during Start for HTTP.
	r := newReceiver(t, NewFactory(), cfg, new(consumertest.TracesSink), new(consumertest.MetricsSink))
	assert.EqualError(t, r.Start(context.Background(), componenttest.NewNopHost()),
		`failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither`)
}

func newGRPCReceiver(t *testing.T, name string, endpoint string, tc consumer.TracesConsumer, mc consumer.MetricsConsumer) *otlpReceiver {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SetName(name)
	cfg.GRPC.NetAddr.Endpoint = endpoint
	cfg.HTTP = nil
	return newReceiver(t, factory, cfg, tc, mc)
}

func newHTTPReceiver(t *testing.T, endpoint string, tc consumer.TracesConsumer, mc consumer.MetricsConsumer) *otlpReceiver {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SetName(otlpReceiverName)
	cfg.HTTP.Endpoint = endpoint
	cfg.GRPC = nil
	return newReceiver(t, factory, cfg, tc, mc)
}

func newReceiver(t *testing.T, factory component.ReceiverFactory, cfg *Config, tc consumer.TracesConsumer, mc consumer.MetricsConsumer) *otlpReceiver {
	r, err := createReceiver(cfg, zap.NewNop())
	require.NoError(t, err)
	if tc != nil {
		params := component.ReceiverCreateParams{}
		_, err := factory.CreateTracesReceiver(context.Background(), params, cfg, tc)
		require.NoError(t, err)
	}
	if mc != nil {
		params := component.ReceiverCreateParams{}
		_, err := factory.CreateMetricsReceiver(context.Background(), params, cfg, mc)
		require.NoError(t, err)
	}
	return r
}

func compressGzip(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	defer gw.Close()

	_, err := gw.Write(body)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

type senderFunc func(msg *collectortrace.ExportTraceServiceRequest)

func TestShutdown(t *testing.T) {
	endpointGrpc := testutil.GetAvailableLocalAddress(t)
	endpointHTTP := testutil.GetAvailableLocalAddress(t)

	nextSink := new(consumertest.TracesSink)

	// Create OTLP receiver with gRPC and HTTP protocols.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SetName(otlpReceiverName)
	cfg.GRPC.NetAddr.Endpoint = endpointGrpc
	cfg.HTTP.Endpoint = endpointHTTP
	ocr := newReceiver(t, factory, cfg, nextSink, nil)
	require.NotNil(t, ocr)
	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()))

	conn, err := grpc.Dial(endpointGrpc, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()

	doneSignalGrpc := make(chan bool)
	doneSignalHTTP := make(chan bool)

	senderGrpc := func(msg *collectortrace.ExportTraceServiceRequest) {
		// Send request via OTLP/gRPC.
		client := collectortrace.NewTraceServiceClient(conn)
		client.Export(context.Background(), msg)
	}
	senderHTTP := func(msg *collectortrace.ExportTraceServiceRequest) {
		// Send request via OTLP/HTTP.
		traceBytes, err2 := msg.Marshal()
		if err2 != nil {
			t.Errorf("Error marshaling protobuf: %v", err2)
		}
		url := fmt.Sprintf("http://%s/v1/traces", endpointHTTP)
		req := createHTTPProtobufRequest(t, url, "", traceBytes)
		client := &http.Client{}
		resp, err2 := client.Do(req)
		if err2 == nil {
			ioutil.ReadAll(resp.Body)
		}
	}

	// Send traces to the receiver until we signal via done channel, and then
	// send one more trace after that.
	go generateTraces(senderGrpc, doneSignalGrpc)
	go generateTraces(senderHTTP, doneSignalHTTP)

	// Wait until the receiver outputs anything to the sink.
	assert.Eventually(t, func() bool {
		return nextSink.SpansCount() > 0
	}, time.Second, 10*time.Millisecond)

	// Now shutdown the receiver, while continuing sending traces to it.
	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	err = ocr.Shutdown(ctx)
	assert.NoError(t, err)

	// Remember how many spans the sink received. This number should not change after this
	// point because after Shutdown() returns the component is not allowed to produce
	// any more data.
	sinkSpanCountAfterShutdown := nextSink.SpansCount()

	// Now signal to generateTraces to exit the main generation loop, then send
	// one more trace and stop.
	doneSignalGrpc <- true
	doneSignalHTTP <- true

	// Wait until all follow up traces are sent.
	<-doneSignalGrpc
	<-doneSignalHTTP

	// The last, additional trace should not be received by sink, so the number of spans in
	// the sink should not change.
	assert.EqualValues(t, sinkSpanCountAfterShutdown, nextSink.SpansCount())
}

func generateTraces(senderFn senderFunc, doneSignal chan bool) {
	// Continuously generate spans until signaled to stop.
loop:
	for {
		select {
		case <-doneSignal:
			break loop
		default:
		}
		senderFn(createSingleSpanTrace())
	}

	// After getting the signal to stop, send one more span and then
	// finally stop. We should never receive this last span.
	senderFn(createSingleSpanTrace())

	// Indicate that we are done.
	close(doneSignal)
}
