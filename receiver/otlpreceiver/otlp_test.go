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
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
)

const otlpReceiverName = "receiver_test"

var otlpReceiverID = component.NewIDWithName(typeStr, otlpReceiverName)

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
		  "scope_spans": [
			{
			  "spans": [
				{
				  "trace_id": "5B8EFFF798038103D269B633813FC60C",
				  "span_id": "EEE19B7EC3C1B174",
				  "parent_span_id": "EEE19B7EC3C1B173",
				  "name": "testSpan",
				  "start_time_unix_nano": 1544712660300000000,
				  "end_time_unix_nano": 1544712660600000000,
				  "kind": 2,
				  "attributes": [
					{
					  "key": "attr1",
					  "value": { "intValue": 55 }
					}
				  ]
				},
				{
				  "trace_id": "5B8EFFF798038103D269B633813FC60C",
				  "span_id": "EEE19B7EC3C1B173",
				  "name": "testSpan",
				  "start_time_unix_nano": 1544712660000000000,
				  "end_time_unix_nano": 1544712661000000000,
				  "kind": "SPAN_KIND_CLIENT",
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

var traceOtlp = func() ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr(semconv.AttributeHostName, "testHost")
	spans := rs.ScopeSpans().AppendEmpty().Spans()
	span1 := spans.AppendEmpty()
	span1.SetTraceID([16]byte{0x5B, 0x8E, 0xFF, 0xF7, 0x98, 0x3, 0x81, 0x3, 0xD2, 0x69, 0xB6, 0x33, 0x81, 0x3F, 0xC6, 0xC})
	span1.SetSpanID([8]byte{0xEE, 0xE1, 0x9B, 0x7E, 0xC3, 0xC1, 0xB1, 0x74})
	span1.SetParentSpanID([8]byte{0xEE, 0xE1, 0x9B, 0x7E, 0xC3, 0xC1, 0xB1, 0x73})
	span1.SetName("testSpan")
	span1.SetStartTimestamp(1544712660300000000)
	span1.SetEndTimestamp(1544712660600000000)
	span1.SetKind(ptrace.SpanKindServer)
	span1.Attributes().PutInt("attr1", 55)
	span2 := spans.AppendEmpty()
	span2.SetTraceID([16]byte{0x5B, 0x8E, 0xFF, 0xF7, 0x98, 0x3, 0x81, 0x3, 0xD2, 0x69, 0xB6, 0x33, 0x81, 0x3F, 0xC6, 0xC})
	span2.SetSpanID([8]byte{0xEE, 0xE1, 0x9B, 0x7E, 0xC3, 0xC1, 0xB1, 0x73})
	span2.SetName("testSpan")
	span2.SetStartTimestamp(1544712660000000000)
	span2.SetEndTimestamp(1544712661000000000)
	span2.SetKind(ptrace.SpanKindClient)
	span2.Attributes().PutInt("attr1", 55)
	return td
}()

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
	sink := &errOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}
	ocr := newHTTPReceiver(t, addr, sink, nil)

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, ocr.Shutdown(context.Background())) })

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s/v1/traces", addr)
			sink.Reset()
			testHTTPJSONRequest(t, url, sink, test.encoding, test.err)
		})
	}
}

func TestHandleInvalidRequests(t *testing.T) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	cfg := &Config{
		Protocols: Protocols{HTTP: &confighttp.HTTPServerSettings{Endpoint: endpoint}},
	}

	// Traces
	tr, err := NewFactory().CreateTracesReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tr)
	require.NoError(t, tr.Start(context.Background(), componenttest.NewNopHost()))

	// Metrics
	mr, err := NewFactory().CreateMetricsReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tr)
	require.NoError(t, mr.Start(context.Background(), componenttest.NewNopHost()))

	// Logs
	lr, err := NewFactory().CreateLogsReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tr)
	require.NoError(t, lr.Start(context.Background(), componenttest.NewNopHost()))

	tests := []struct {
		name        string
		uri         string
		method      string
		contentType string

		expectedStatus       int
		expectedResponseBody string
	}{
		{
			name:        "POST /v1/traces, no content type",
			uri:         "/v1/traces",
			method:      http.MethodPost,
			contentType: "",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "PATCH /v1/traces",
			uri:         "/v1/traces",
			method:      http.MethodPatch,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			name:        "GET /v1/traces",
			uri:         "/v1/traces",
			method:      http.MethodGet,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			name:        "POST /v1/metrics, no content type",
			uri:         "/v1/metrics",
			method:      http.MethodPost,
			contentType: "",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "PATCH /v1/metrics",
			uri:         "/v1/metrics",
			method:      http.MethodPatch,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			name:        "GET /v1/metrics",
			uri:         "/v1/metrics",
			method:      http.MethodGet,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			name:        "POST /v1/logs, no content type",
			uri:         "/v1/logs",
			method:      http.MethodPost,
			contentType: "",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "PATCH /v1/logs",
			uri:         "/v1/logs",
			method:      http.MethodPatch,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			name:        "GET /v1/logs",
			uri:         "/v1/logs",
			method:      http.MethodGet,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s%s", endpoint, test.uri)
			req, err2 := http.NewRequest(test.method, url, bytes.NewReader([]byte(`{}`)))
			require.NoError(t, err2)
			req.Header.Set("Content-Type", test.contentType)

			client := &http.Client{}
			resp, err2 := client.Do(req)
			require.NoError(t, err2)

			body, err2 := io.ReadAll(resp.Body)
			require.NoError(t, err2)

			require.Equal(t, resp.Header.Get("Content-Type"), "text/plain")
			require.Equal(t, resp.StatusCode, test.expectedStatus)
			require.EqualValues(t, body, test.expectedResponseBody)
		})
	}

	err = tr.Shutdown(context.Background())
	require.NoError(t, err)
}

func testHTTPJSONRequest(t *testing.T, url string, sink *errOrSinkConsumer, encoding string, expectedErr error) {
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
	req, err := http.NewRequest(http.MethodPost, url, buf)
	require.NoError(t, err, "Error creating trace POST request: %v", err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", encoding)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "Error posting trace to http server: %v", err)

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	allTraces := sink.AllTraces()
	if expectedErr == nil {
		assert.Equal(t, 200, resp.StatusCode)
		tr := ptraceotlp.NewExportResponse()
		assert.NoError(t, tr.UnmarshalJSON(respBytes), "Unable to unmarshal response to Response")

		require.Len(t, allTraces, 1)
		assert.EqualValues(t, allTraces[0], traceOtlp)
	} else {
		errStatus := &spb.Status{}
		assert.NoError(t, json.Unmarshal(respBytes, errStatus))
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
	tSink := &errOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}
	ocr := newHTTPReceiver(t, addr, tSink, consumertest.NewNop())

	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, ocr.Shutdown(context.Background())) })

	// TODO(nilebox): make starting server deterministic
	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	td := testdata.GenerateTraces(1)
	marshaler := &ptrace.ProtoMarshaler{}
	traceBytes, err := marshaler.MarshalTraces(td)
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s/v1/traces", addr)
			tSink.Reset()
			testHTTPProtobufRequest(t, url, tSink, test.encoding, traceBytes, test.err, td)
		})
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
	req, err := http.NewRequest(http.MethodPost, url, buf)
	require.NoError(t, err, "Error creating trace POST request: %v", err)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", encoding)
	return req
}

func testHTTPProtobufRequest(
	t *testing.T,
	url string,
	tSink *errOrSinkConsumer,
	encoding string,
	traceBytes []byte,
	expectedErr error,
	wantData ptrace.Traces,
) {
	tSink.SetConsumeError(expectedErr)

	req := createHTTPProtobufRequest(t, url, encoding, traceBytes)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "Error posting trace to grpc-gateway server: %v", err)

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Error reading response from trace grpc-gateway")
	require.NoError(t, resp.Body.Close(), "Error closing response body")

	assert.Equal(t, "application/x-protobuf", resp.Header.Get("Content-Type"), "Unexpected response Content-Type")

	allTraces := tSink.AllTraces()

	if expectedErr == nil {
		require.Equal(t, 200, resp.StatusCode, "Unexpected return status")

		tr := ptraceotlp.NewExportResponse()
		assert.NoError(t, tr.UnmarshalProto(respBytes), "Unable to unmarshal response to Response")

		require.Len(t, allTraces, 1)
		assert.EqualValues(t, allTraces[0], wantData)
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
	t.Cleanup(func() { require.NoError(t, ocr.Shutdown(context.Background())) })

	url := fmt.Sprintf("http://%s/v1/traces", addr)

	// Wait for the servers to start
	<-time.After(10 * time.Millisecond)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := test.reqBodyFunc()
			require.NoError(t, err, "Error creating request body: %v", err)

			req, err := http.NewRequest(http.MethodPost, url, body)
			require.NoError(t, err, "Error creating trace POST request: %v", err)
			req.Header.Set("Content-Type", test.content)
			req.Header.Set("Content-Encoding", test.encoding)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err, "Error posting trace to grpc-gateway server: %v", err)

			respBytes, err := io.ReadAll(resp.Body)
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
	t.Cleanup(func() {
		assert.NoError(t, ln.Close())
	})

	r := newGRPCReceiver(t, addr, consumertest.NewNop(), consumertest.NewNop())
	require.NotNil(t, r)

	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

func TestHTTPNewPortAlreadyUsed(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to listen on %q: %v", addr, err)
	t.Cleanup(func() {
		assert.NoError(t, ln.Close())
	})

	r := newHTTPReceiver(t, addr, consumertest.NewNop(), consumertest.NewNop())
	require.NotNil(t, r)

	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

// TestOTLPReceiverGRPCTracesIngestTest checks that the gRPC trace receiver
// is returning the proper response (return and metrics) when the next consumer
// in the pipeline reports error. The test changes the responses returned by the
// next trace consumer, checks if data was passed down the pipeline and if
// proper metrics were recorded. It also uses all endpoints supported by the
// trace receiver.
func TestOTLPReceiverGRPCTracesIngestTest(t *testing.T) {
	type ingestionStateTest struct {
		okToIngest   bool
		expectedCode codes.Code
	}

	expectedReceivedBatches := 2
	expectedIngestionBlockedRPCs := 1
	ingestionStates := []ingestionStateTest{
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
	}

	addr := testutil.GetAvailableLocalAddress(t)
	td := testdata.GenerateTraces(1)

	tt, err := obsreporttest.SetupTelemetry(otlpReceiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	sink := &errOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}

	ocr := newGRPCReceiver(t, addr, sink, nil)
	require.NotNil(t, ocr)
	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, ocr.Shutdown(context.Background())) })

	cc, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, cc.Close())
	}()

	for _, ingestionState := range ingestionStates {
		if ingestionState.okToIngest {
			sink.SetConsumeError(nil)
		} else {
			sink.SetConsumeError(errors.New("consumer error"))
		}

		_, err = ptraceotlp.NewGRPCClient(cc).Export(context.Background(), ptraceotlp.NewExportRequestFromTraces(td))
		errStatus, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, ingestionState.expectedCode, errStatus.Code())
	}

	require.Equal(t, expectedReceivedBatches, len(sink.AllTraces()))

	require.NoError(t, tt.CheckReceiverTraces("grpc", int64(expectedReceivedBatches), int64(expectedIngestionBlockedRPCs)))
}

// TestOTLPReceiverHTTPTracesNextConsumerResponse checks that the HTTP trace receiver
// is returning the proper response (return and metrics) when the next consumer
// in the pipeline reports error. The test changes the responses returned by the
// next trace consumer, checks if data was passed down the pipeline and if
// proper metrics were recorded. It also uses all endpoints supported by the
// trace receiver.
func TestOTLPReceiverHTTPTracesIngestTest(t *testing.T) {
	type ingestionStateTest struct {
		okToIngest   bool
		expectedCode codes.Code
	}

	expectedReceivedBatches := 2
	expectedIngestionBlockedRPCs := 1
	ingestionStates := []ingestionStateTest{
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
	}

	addr := testutil.GetAvailableLocalAddress(t)
	td := testdata.GenerateTraces(1)

	tt, err := obsreporttest.SetupTelemetry(otlpReceiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	sink := &errOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}

	ocr := newHTTPReceiver(t, addr, sink, nil)
	require.NotNil(t, ocr)
	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, ocr.Shutdown(context.Background())) })

	for _, ingestionState := range ingestionStates {
		if ingestionState.okToIngest {
			sink.SetConsumeError(nil)
		} else {
			sink.SetConsumeError(errors.New("consumer error"))
		}

		pbMarshaler := ptrace.ProtoMarshaler{}
		pbBytes, err := pbMarshaler.MarshalTraces(td)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/traces", bytes.NewReader(pbBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", pbContentType)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		respBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		if ingestionState.expectedCode == codes.OK {
			require.Equal(t, 200, resp.StatusCode)
			tr := ptraceotlp.NewExportResponse()
			assert.NoError(t, tr.UnmarshalProto(respBytes))
		} else {
			errStatus := &spb.Status{}
			assert.NoError(t, proto.Unmarshal(respBytes, errStatus))
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.Equal(t, ingestionState.expectedCode, codes.Code(errStatus.Code))
		}
	}

	require.Equal(t, expectedReceivedBatches, len(sink.AllTraces()))

	require.NoError(t, tt.CheckReceiverTraces("http", int64(expectedReceivedBatches), int64(expectedIngestionBlockedRPCs)))
}

func TestGRPCInvalidTLSCredentials(t *testing.T) {
	cfg := &Config{
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

	r, err := NewFactory().CreateTracesReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, r)

	assert.EqualError(t,
		r.Start(context.Background(), componenttest.NewNopHost()),
		`failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither`)
}

func TestGRPCMaxRecvSize(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	sink := new(consumertest.TracesSink)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPC.NetAddr.Endpoint = addr
	cfg.HTTP = nil
	ocr := newReceiver(t, factory, cfg, otlpReceiverID, sink, nil)

	require.NotNil(t, ocr)
	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()))

	cc, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err)

	td := testdata.GenerateTraces(50000)
	require.Error(t, exportTraces(cc, td))
	assert.NoError(t, cc.Close())
	require.NoError(t, ocr.Shutdown(context.Background()))

	cfg.GRPC.MaxRecvMsgSizeMiB = 100
	ocr = newReceiver(t, factory, cfg, otlpReceiverID, sink, nil)

	require.NotNil(t, ocr)
	require.NoError(t, ocr.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, ocr.Shutdown(context.Background())) })

	cc, err = grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, cc.Close())
	}()

	td = testdata.GenerateTraces(50000)
	require.NoError(t, exportTraces(cc, td))
	require.Len(t, sink.AllTraces(), 1)
	assert.Equal(t, td, sink.AllTraces()[0])
}

func TestHTTPInvalidTLSCredentials(t *testing.T) {
	cfg := &Config{
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
	r, err := NewFactory().CreateTracesReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.EqualError(t, r.Start(context.Background(), componenttest.NewNopHost()),
		`failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither`)
}

func testHTTPMaxRequestBodySizeJSON(t *testing.T, payload []byte, size int, expectedStatusCode int) {
	endpoint := testutil.GetAvailableLocalAddress(t)
	url := fmt.Sprintf("http://%s/v1/traces", endpoint)
	cfg := &Config{
		Protocols: Protocols{
			HTTP: &confighttp.HTTPServerSettings{
				Endpoint:           endpoint,
				MaxRequestBodySize: int64(size),
			},
		},
	}

	r, err := NewFactory().CreateTracesReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, r)
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, expectedStatusCode, resp.StatusCode)

	err = r.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestHTTPMaxRequestBodySize_OK(t *testing.T) {
	testHTTPMaxRequestBodySizeJSON(t, traceJSON, len(traceJSON), 200)
}

func TestHTTPMaxRequestBodySize_TooLarge(t *testing.T) {
	testHTTPMaxRequestBodySizeJSON(t, traceJSON, len(traceJSON)-1, 400)
}

func newGRPCReceiver(t *testing.T, endpoint string, tc consumer.Traces, mc consumer.Metrics) component.Component {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPC.NetAddr.Endpoint = endpoint
	cfg.HTTP = nil
	return newReceiver(t, factory, cfg, otlpReceiverID, tc, mc)
}

func newHTTPReceiver(t *testing.T, endpoint string, tc consumer.Traces, mc consumer.Metrics) component.Component {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.HTTP.Endpoint = endpoint
	cfg.GRPC = nil
	return newReceiver(t, factory, cfg, otlpReceiverID, tc, mc)
}

func newReceiver(t *testing.T, factory receiver.Factory, cfg *Config, id component.ID, tc consumer.Traces, mc consumer.Metrics) component.Component {
	set := receivertest.NewNopCreateSettings()
	set.TelemetrySettings.MetricsLevel = configtelemetry.LevelNormal
	set.ID = id
	var r component.Component
	var err error
	if tc != nil {
		r, err = factory.CreateTracesReceiver(context.Background(), set, cfg, tc)
		require.NoError(t, err)
	}
	if mc != nil {
		r, err = factory.CreateMetricsReceiver(context.Background(), set, cfg, mc)
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

type senderFunc func(td ptrace.Traces)

func TestShutdown(t *testing.T) {
	endpointGrpc := testutil.GetAvailableLocalAddress(t)
	endpointHTTP := testutil.GetAvailableLocalAddress(t)

	nextSink := new(consumertest.TracesSink)

	// Create OTLP receiver with gRPC and HTTP protocols.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPC.NetAddr.Endpoint = endpointGrpc
	cfg.HTTP.Endpoint = endpointHTTP
	set := receivertest.NewNopCreateSettings()
	set.ID = otlpReceiverID
	r, err := NewFactory().CreateTracesReceiver(
		context.Background(),
		set,
		cfg,
		nextSink)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))

	conn, err := grpc.Dial(endpointGrpc, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()

	doneSignalGrpc := make(chan bool)
	doneSignalHTTP := make(chan bool)

	senderGrpc := func(td ptrace.Traces) {
		// Ignore error, may be executed after the receiver shutdown.
		_ = exportTraces(conn, td)
	}
	senderHTTP := func(td ptrace.Traces) {
		// Send request via OTLP/HTTP.
		marshaler := &ptrace.ProtoMarshaler{}
		traceBytes, err2 := marshaler.MarshalTraces(td)
		require.NoError(t, err2)
		url := fmt.Sprintf("http://%s/v1/traces", endpointHTTP)
		req := createHTTPProtobufRequest(t, url, "", traceBytes)
		client := &http.Client{}
		resp, err2 := client.Do(req)
		if err2 == nil {
			resp.Body.Close()
		}
	}

	// Send traces to the receiver until we signal via done channel, and then
	// send one more trace after that.
	go generateTraces(senderGrpc, doneSignalGrpc)
	go generateTraces(senderHTTP, doneSignalHTTP)

	// Wait until the receiver outputs anything to the sink.
	assert.Eventually(t, func() bool {
		return nextSink.SpanCount() > 0
	}, time.Second, 10*time.Millisecond)

	// Now shutdown the receiver, while continuing sending traces to it.
	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	err = r.Shutdown(ctx)
	assert.NoError(t, err)

	// Remember how many spans the sink received. This number should not change after this
	// point because after Shutdown() returns the component is not allowed to produce
	// any more data.
	sinkSpanCountAfterShutdown := nextSink.SpanCount()

	// Now signal to generateTraces to exit the main generation loop, then send
	// one more trace and stop.
	doneSignalGrpc <- true
	doneSignalHTTP <- true

	// Wait until all follow up traces are sent.
	<-doneSignalGrpc
	<-doneSignalHTTP

	// The last, additional trace should not be received by sink, so the number of spans in
	// the sink should not change.
	assert.EqualValues(t, sinkSpanCountAfterShutdown, nextSink.SpanCount())
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
		senderFn(testdata.GenerateTraces(1))
	}

	// After getting the signal to stop, send one more span and then
	// finally stop. We should never receive this last span.
	senderFn(testdata.GenerateTraces(1))

	// Indicate that we are done.
	close(doneSignal)
}

func exportTraces(cc *grpc.ClientConn, td ptrace.Traces) error {
	acc := ptraceotlp.NewGRPCClient(cc)
	req := ptraceotlp.NewExportRequestFromTraces(td)
	_, err := acc.Export(context.Background(), req)

	return err
}

type errOrSinkConsumer struct {
	*consumertest.TracesSink
	*consumertest.MetricsSink
	mu           sync.Mutex
	consumeError error // to be returned by ConsumeTraces, if set
}

// SetConsumeError sets an error that will be returned by the Consume function.
func (esc *errOrSinkConsumer) SetConsumeError(err error) {
	esc.mu.Lock()
	defer esc.mu.Unlock()
	esc.consumeError = err
}

func (esc *errOrSinkConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces stores traces to this sink.
func (esc *errOrSinkConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	if esc.consumeError != nil {
		return esc.consumeError
	}

	return esc.TracesSink.ConsumeTraces(ctx, td)
}

// ConsumeMetrics stores metrics to this sink.
func (esc *errOrSinkConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	if esc.consumeError != nil {
		return esc.consumeError
	}

	return esc.MetricsSink.ConsumeMetrics(ctx, md)
}

// Reset deletes any stored in the sinks, resets error to nil.
func (esc *errOrSinkConsumer) Reset() {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	esc.consumeError = nil
	if esc.TracesSink != nil {
		esc.TracesSink.Reset()
	}
	if esc.MetricsSink != nil {
		esc.MetricsSink.Reset()
	}
}
