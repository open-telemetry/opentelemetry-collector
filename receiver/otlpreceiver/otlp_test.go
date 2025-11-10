// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
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
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const otlpReceiverName = "receiver_test"

var otlpReceiverID = component.MustNewIDWithName("otlp", otlpReceiverName)

func TestJsonHttp(t *testing.T) {
	tests := []struct {
		name               string
		encoding           string
		contentType        string
		err                error
		expectedStatus     *spb.Status
		expectedStatusCode int
	}{
		{
			name:        "JSONUncompressed",
			encoding:    "",
			contentType: "application/json",
		},
		{
			name:        "JSONUncompressedUTF8",
			encoding:    "",
			contentType: "application/json; charset=utf-8",
		},
		{
			name:        "JSONUncompressedUppercase",
			encoding:    "",
			contentType: "APPLICATION/JSON",
		},
		{
			name:        "JSONGzipCompressed",
			encoding:    "gzip",
			contentType: "application/json",
		},
		{
			name:        "JSONZstdCompressed",
			encoding:    "zstd",
			contentType: "application/json",
		},
		{
			name:               "Permanent NotGRPCError",
			encoding:           "",
			contentType:        "application/json",
			err:                consumererror.NewPermanent(errors.New("my error")),
			expectedStatus:     &spb.Status{Code: int32(codes.Internal), Message: "Permanent error: my error"},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "Retryable NotGRPCError",
			encoding:           "",
			contentType:        "application/json",
			err:                errors.New("my error"),
			expectedStatus:     &spb.Status{Code: int32(codes.Unavailable), Message: "my error"},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			name:               "Permanent GRPCError",
			encoding:           "",
			contentType:        "application/json",
			err:                status.New(codes.Internal, "").Err(),
			expectedStatus:     &spb.Status{Code: int32(codes.Internal), Message: ""},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "Retryable GRPCError",
			encoding:           "",
			contentType:        "application/json",
			err:                status.New(codes.Unavailable, "Service Unavailable").Err(),
			expectedStatus:     &spb.Status{Code: int32(codes.Unavailable), Message: "Service Unavailable"},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
	}
	addr := testutil.GetAvailableLocalAddress(t)
	sink := newErrOrSinkConsumer()
	recv := newHTTPReceiver(t, componenttest.NewNopTelemetrySettings(), addr, sink)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink.Reset()
			sink.SetConsumeError(tt.err)

			for _, dr := range generateDataRequests(t) {
				url := "http://" + addr + dr.path
				respBytes := doHTTPRequest(t, url, tt.encoding, tt.contentType, dr.jsonBytes, tt.expectedStatusCode)
				if tt.err == nil {
					tr := ptraceotlp.NewExportResponse()
					require.NoError(t, tr.UnmarshalJSON(respBytes), "Unable to unmarshal response to Response")
					sink.checkData(t, dr.data, 1)
				} else {
					errStatus := &spb.Status{}
					require.NoError(t, json.Unmarshal(respBytes, errStatus))
					if s, ok := status.FromError(tt.err); ok {
						assert.Equal(t, s.Proto().Code, errStatus.Code)
						assert.Equal(t, s.Proto().Message, errStatus.Message)
					} else {
						fmt.Println(errStatus)
						assert.True(t, proto.Equal(errStatus, tt.expectedStatus))
					}
					sink.checkData(t, dr.data, 0)
				}
			}
		})
	}
}

func TestHandleInvalidRequests(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	sink := newErrOrSinkConsumer()
	recv := newHTTPReceiver(t, componenttest.NewNopTelemetrySettings(), addr, sink)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	tests := []struct {
		name        string
		uri         string
		method      string
		contentType string

		expectedStatus       int
		expectedResponseBody string
	}{
		{
			name:        "no content type",
			uri:         defaultTracesURLPath,
			method:      http.MethodPost,
			contentType: "",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "invalid content type",
			uri:         defaultTracesURLPath,
			method:      http.MethodPost,
			contentType: "invalid",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "invalid request",
			uri:         defaultTracesURLPath,
			method:      http.MethodPost,
			contentType: "application/json",

			expectedStatus: http.StatusBadRequest,
		},
		{
			uri:         defaultTracesURLPath,
			method:      http.MethodPatch,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			uri:         defaultTracesURLPath,
			method:      http.MethodGet,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			name:        "no content type",
			uri:         defaultMetricsURLPath,
			method:      http.MethodPost,
			contentType: "",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "invalid content type",
			uri:         defaultMetricsURLPath,
			method:      http.MethodPost,
			contentType: "invalid",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "invalid request",
			uri:         defaultMetricsURLPath,
			method:      http.MethodPost,
			contentType: "application/json",

			expectedStatus: http.StatusBadRequest,
		},
		{
			uri:         defaultMetricsURLPath,
			method:      http.MethodPatch,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			uri:         defaultMetricsURLPath,
			method:      http.MethodGet,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			name:        "no content type",
			uri:         defaultLogsURLPath,
			method:      http.MethodPost,
			contentType: "",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "invalid content type",
			uri:         defaultLogsURLPath,
			method:      http.MethodPost,
			contentType: "invalid",

			expectedStatus:       http.StatusUnsupportedMediaType,
			expectedResponseBody: "415 unsupported media type, supported: [application/json, application/x-protobuf]",
		},
		{
			name:        "invalid request",
			uri:         defaultLogsURLPath,
			method:      http.MethodPost,
			contentType: "application/json",

			expectedStatus: http.StatusBadRequest,
		},
		{
			uri:         defaultLogsURLPath,
			method:      http.MethodPatch,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
		{
			uri:         defaultLogsURLPath,
			method:      http.MethodGet,
			contentType: "application/json",

			expectedStatus:       http.StatusMethodNotAllowed,
			expectedResponseBody: "405 method not allowed, supported: [POST]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.uri+" "+tt.name, func(t *testing.T) {
			url := "http://" + addr + tt.uri
			req, err := http.NewRequest(tt.method, url, bytes.NewReader([]byte(`1234`)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", tt.contentType)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.name == "invalid request" {
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
				return
			}
			assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedResponseBody, string(body))
		})
	}

	require.NoError(t, recv.Shutdown(context.Background()))
}

func TestProtoHttp(t *testing.T) {
	tests := []struct {
		name               string
		encoding           string
		err                error
		expectedStatus     *spb.Status
		expectedStatusCode int
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
			name:     "ProtoZstdCompressed",
			encoding: "zstd",
		},
		{
			name:               "Permanent NotGRPCError",
			encoding:           "",
			err:                consumererror.NewPermanent(errors.New("my error")),
			expectedStatus:     &spb.Status{Code: int32(codes.Internal), Message: "Permanent error: my error"},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "Retryable NotGRPCError",
			encoding:           "",
			err:                errors.New("my error"),
			expectedStatus:     &spb.Status{Code: int32(codes.Unavailable), Message: "my error"},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			name:               "Permanent GRPCError",
			encoding:           "",
			err:                status.New(codes.InvalidArgument, "Bad Request").Err(),
			expectedStatus:     &spb.Status{Code: int32(codes.InvalidArgument), Message: "Bad Request"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Retryable GRPCError",
			encoding:           "",
			err:                status.New(codes.Unavailable, "Service Unavailable").Err(),
			expectedStatus:     &spb.Status{Code: int32(codes.Unavailable), Message: "Service Unavailable"},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
	}
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	sink := newErrOrSinkConsumer()
	recv := newHTTPReceiver(t, componenttest.NewNopTelemetrySettings(), addr, sink)

	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink.Reset()
			sink.SetConsumeError(tt.err)

			for _, dr := range generateDataRequests(t) {
				url := "http://" + addr + dr.path
				respBytes := doHTTPRequest(t, url, tt.encoding, "application/x-protobuf", dr.protoBytes, tt.expectedStatusCode)
				if tt.err == nil {
					tr := ptraceotlp.NewExportResponse()
					require.NoError(t, tr.UnmarshalProto(respBytes))
					sink.checkData(t, dr.data, 1)
				} else {
					errStatus := &spb.Status{}
					require.NoError(t, proto.Unmarshal(respBytes, errStatus))
					if s, ok := status.FromError(tt.err); ok {
						assert.True(t, proto.Equal(errStatus, s.Proto()))
					} else {
						assert.True(t, proto.Equal(errStatus, tt.expectedStatus))
					}
					sink.checkData(t, dr.data, 0)
				}
			}
		})
	}
}

func TestOTLPReceiverInvalidContentEncoding(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		encoding    string
		reqBodyFunc func() (*bytes.Buffer, error)
		checkBody   func(tb testing.TB, got []byte)
		status      int
	}{
		{
			name:     "JsonGzipUncompressed",
			content:  "application/json",
			encoding: "gzip",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return bytes.NewBuffer([]byte(`{"key": "value"}`)), nil
			},
			checkBody: func(tb testing.TB, got []byte) {
				assert.JSONEq(tb, `{"code":3,"message": "gzip: invalid header"}`, string(got))
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
			checkBody: func(tb testing.TB, got []byte) {
				expected, err := proto.Marshal(status.New(codes.InvalidArgument, "gzip: invalid header").Proto())
				require.NoError(tb, err)
				assert.Equal(tb, expected, got)
			},
			status: 400,
		},
		{
			name:     "ProtoZstdUncompressed",
			content:  "application/x-protobuf",
			encoding: "zstd",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return bytes.NewBuffer([]byte(`{"key": "value"}`)), nil
			},
			checkBody: func(tb testing.TB, got []byte) {
				expected, err := proto.Marshal(status.New(codes.InvalidArgument, "invalid input: magic number mismatch").Proto())
				require.NoError(tb, err)
				assert.Equal(tb, expected, got)
			},
			status: 400,
		},
	}
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	recv := newHTTPReceiver(t, componenttest.NewNopTelemetrySettings(), addr, consumertest.NewNop())

	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	url := fmt.Sprintf("http://%s%s", addr, defaultTracesURLPath)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := test.reqBodyFunc()
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, url, body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", test.content)
			req.Header.Set("Content-Encoding", test.encoding)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, test.status, resp.StatusCode, "Unexpected return status")
			assert.Equal(t, test.content, resp.Header.Get("Content-Type"), "Unexpected response Content-Type")

			respBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			test.checkBody(t, respBytes)
		})
	}
}

func TestOTLPReceiverNoContentType(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	// Set the buffer count to 1 to make it flush the test span immediately.
	recv := newHTTPReceiver(t, componenttest.NewNopTelemetrySettings(), addr, consumertest.NewNop())

	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()), "Failed to start trace receiver")
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	url := fmt.Sprintf("http://%s%s", addr, defaultTracesURLPath)

	t.Run("NoContentType", func(t *testing.T) {
		body := bytes.NewBuffer([]byte(`{"key": "value"}`))

		req, err := http.NewRequest(http.MethodPost, url, body)
		require.NoError(t, err, "Error creating trace POST request: %v", err)

		// Set invalid encoding to trigger an error
		req.Header.Set("Content-Encoding", "invalid")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Error posting to server: %v", err)
		// Don't care about the response body, just check the content type
		defer resp.Body.Close()

		require.Equal(t, fallbackContentType, resp.Header.Get("Content-Type"), "Unexpected response Content-Type")
	})
}

func TestGRPCNewPortAlreadyUsed(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to listen on %q: %v", addr, err)
	t.Cleanup(func() {
		assert.NoError(t, ln.Close())
	})

	r := newGRPCReceiver(t, componenttest.NewNopTelemetrySettings(), addr, consumertest.NewNop())
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

	r := newHTTPReceiver(t, componenttest.NewNopTelemetrySettings(), addr, consumertest.NewNop())
	require.NotNil(t, r)

	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

// TestOTLPReceiverGRPCMetricsIngestTest checks that the metrics receiver
// is returning the proper response (return and metrics) when the next consumer
// in the pipeline reports error.
func TestOTLPReceiverGRPCMetricsIngestTest(t *testing.T) {
	// Get a new available port
	addr := testutil.GetAvailableLocalAddress(t)

	// Create a sink
	sink := &errOrSinkConsumer{MetricsSink: new(consumertest.MetricsSink)}

	// Create a telemetry instance
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })
	// Create telemetry settings
	settings := tt.NewTelemetrySettings()

	recv := newGRPCReceiver(t, settings, addr, sink)
	require.NotNil(t, recv)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, cc.Close())
	}()
	// Set up the error case
	sink.SetConsumeError(errors.New("consumer error"))

	md := testdata.GenerateMetrics(1)
	_, err = pmetricotlp.NewGRPCClient(cc).Export(context.Background(), pmetricotlp.NewExportRequestFromMetrics(md))
	errStatus, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, errStatus.Code())

	// Assert receiver metrics including receiver_requests
	assertReceiverMetrics(t, tt, otlpReceiverID, "grpc", 0, 2)
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
		permanent    bool
		expectedCode codes.Code
	}

	expectedReceivedBatches := 2
	expectedIngestionBlockedRPCs := 2
	ingestionStates := []ingestionStateTest{
		{
			okToIngest:   true,
			expectedCode: codes.OK,
		},
		{
			okToIngest:   false,
			expectedCode: codes.Unavailable,
		},
		{
			okToIngest:   false,
			expectedCode: codes.Internal,
			permanent:    true,
		},
		{
			okToIngest:   true,
			expectedCode: codes.OK,
		},
	}

	addr := testutil.GetAvailableLocalAddress(t)
	td := testdata.GenerateTraces(1)

	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	sink := &errOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}

	recv := newGRPCReceiver(t, tt.NewTelemetrySettings(), addr, sink)
	require.NotNil(t, recv)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, cc.Close())
	}()

	for _, ingestionState := range ingestionStates {
		if ingestionState.okToIngest {
			sink.SetConsumeError(nil)
		} else {
			if ingestionState.permanent {
				sink.SetConsumeError(consumererror.NewPermanent(errors.New("consumer error")))
			} else {
				sink.SetConsumeError(errors.New("consumer error"))
			}
		}

		_, err = ptraceotlp.NewGRPCClient(cc).Export(context.Background(), ptraceotlp.NewExportRequestFromTraces(td))
		errStatus, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, ingestionState.expectedCode, errStatus.Code())
	}

	require.Len(t, sink.AllTraces(), expectedReceivedBatches)

	assertReceiverTraces(t, tt, otlpReceiverID, "grpc", int64(expectedReceivedBatches), int64(expectedIngestionBlockedRPCs))
}

// TestOTLPReceiverHTTPTracesIngestTest checks that the HTTP trace receiver
// is returning the proper response (return and metrics) when the next consumer
// in the pipeline reports error. The test changes the responses returned by the
// next trace consumer, checks if data was passed down the pipeline and if
// proper metrics were recorded. It also uses all endpoints supported by the
// trace receiver.
func TestOTLPReceiverHTTPTracesIngestTest(t *testing.T) {
	type ingestionStateTest struct {
		okToIngest         bool
		err                error
		expectedCode       codes.Code
		expectedStatusCode int
	}

	expectedReceivedBatches := 2
	expectedIngestionBlockedRPCs := 2
	ingestionStates := []ingestionStateTest{
		{
			okToIngest:   true,
			expectedCode: codes.OK,
		},
		{
			okToIngest:         false,
			err:                consumererror.NewPermanent(errors.New("consumer error")),
			expectedCode:       codes.Internal,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			okToIngest:         false,
			err:                errors.New("consumer error"),
			expectedCode:       codes.Unavailable,
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			okToIngest:   true,
			expectedCode: codes.OK,
		},
	}

	addr := testutil.GetAvailableLocalAddress(t)
	td := testdata.GenerateTraces(1)

	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	sink := &errOrSinkConsumer{TracesSink: new(consumertest.TracesSink)}

	recv := newHTTPReceiver(t, tt.NewTelemetrySettings(), addr, sink)
	require.NotNil(t, recv)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	for _, ingestionState := range ingestionStates {
		if ingestionState.okToIngest {
			sink.SetConsumeError(nil)
		} else {
			sink.SetConsumeError(ingestionState.err)
		}

		pbMarshaler := ptrace.ProtoMarshaler{}
		pbBytes, err := pbMarshaler.MarshalTraces(td)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, "http://"+addr+defaultTracesURLPath, bytes.NewReader(pbBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", pbContentType)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		respBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		if ingestionState.expectedCode == codes.OK {
			require.Equal(t, 200, resp.StatusCode)
			tr := ptraceotlp.NewExportResponse()
			require.NoError(t, tr.UnmarshalProto(respBytes))
		} else {
			errStatus := &spb.Status{}
			require.NoError(t, proto.Unmarshal(respBytes, errStatus))
			assert.Equal(t, ingestionState.expectedStatusCode, resp.StatusCode)
			assert.EqualValues(t, ingestionState.expectedCode, errStatus.Code)
		}
	}

	require.Len(t, sink.AllTraces(), expectedReceivedBatches)

	assertReceiverTraces(t, tt, otlpReceiverID, "http", int64(expectedReceivedBatches), int64(expectedIngestionBlockedRPCs))
}

func TestGRPCInvalidTLSCredentials(t *testing.T) {
	cfg := &Config{
		Protocols: Protocols{
			GRPC: configoptional.Some(configgrpc.ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  testutil.GetAvailableLocalAddress(t),
					Transport: confignet.TransportTypeTCP,
				},
				TLS: configoptional.Some(configtls.ServerConfig{
					Config: configtls.Config{
						CertFile: "willfail",
					},
				}),
			}),
		},
	}

	r, err := NewFactory().CreateTraces(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, r)

	assert.EqualError(t,
		r.Start(context.Background(), componenttest.NewNopHost()),
		`failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither`)
}

func TestGRPCMaxRecvSize(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	sink := newErrOrSinkConsumer()

	cfg := createDefaultConfig().(*Config)
	cfg.GRPC.GetOrInsertDefault().NetAddr.Endpoint = addr
	recv := newReceiver(t, componenttest.NewNopTelemetrySettings(), cfg, otlpReceiverID, sink)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))

	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	td := testdata.GenerateTraces(50000)
	err = exportTraces(cc, td)
	require.Error(t, err)
	assert.NoError(t, cc.Close())
	require.NoError(t, recv.Shutdown(context.Background()))

	cfg.GRPC.Get().MaxRecvMsgSizeMiB = 100
	recv = newReceiver(t, componenttest.NewNopTelemetrySettings(), cfg, otlpReceiverID, sink)
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, recv.Shutdown(context.Background())) })

	cc, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
			HTTP: configoptional.Some(HTTPConfig{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: testutil.GetAvailableLocalAddress(t),
					TLS: configoptional.Some(configtls.ServerConfig{
						Config: configtls.Config{
							CertFile: "willfail",
						},
					}),
				},
				TracesURLPath:  defaultTracesURLPath,
				MetricsURLPath: defaultMetricsURLPath,
				LogsURLPath:    defaultLogsURLPath,
			}),
		},
	}

	// TLS is resolved during Start for HTTP.
	r, err := NewFactory().CreateTraces(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.EqualError(t, r.Start(context.Background(), componenttest.NewNopHost()),
		`failed to load TLS config: failed to load TLS cert and key: for auth via TLS, provide both certificate and key, or neither`)
}

func testHTTPMaxRequestBodySize(t *testing.T, path, contentType string, payload []byte, size, expectedStatusCode int) {
	addr := testutil.GetAvailableLocalAddress(t)
	url := "http://" + addr + path
	cfg := &Config{
		Protocols: Protocols{
			HTTP: configoptional.Some(HTTPConfig{
				ServerConfig: confighttp.ServerConfig{
					Endpoint:           addr,
					MaxRequestBodySize: int64(size),
				},
				TracesURLPath:  defaultTracesURLPath,
				MetricsURLPath: defaultMetricsURLPath,
				LogsURLPath:    defaultLogsURLPath,
			}),
		},
	}

	recv := newReceiver(t, componenttest.NewNopTelemetrySettings(), cfg, otlpReceiverID, consumertest.NewNop())
	require.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))

	req := createHTTPRequest(t, url, "", contentType, payload)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, expectedStatusCode, resp.StatusCode)

	require.NoError(t, recv.Shutdown(context.Background()))
}

func TestHTTPMaxRequestBodySize(t *testing.T) {
	dataReqs := generateDataRequests(t)

	for _, dr := range dataReqs {
		testHTTPMaxRequestBodySize(t, dr.path, "application/json", dr.jsonBytes, len(dr.jsonBytes), 200)
		testHTTPMaxRequestBodySize(t, dr.path, "application/json", dr.jsonBytes, len(dr.jsonBytes)-1, 400)

		testHTTPMaxRequestBodySize(t, dr.path, "application/x-protobuf", dr.protoBytes, len(dr.protoBytes), 200)
		testHTTPMaxRequestBodySize(t, dr.path, "application/x-protobuf", dr.protoBytes, len(dr.protoBytes)-1, 400)
	}
}

func newGRPCReceiver(t *testing.T, settings component.TelemetrySettings, endpoint string, c consumertest.Consumer) component.Component {
	cfg := createDefaultConfig().(*Config)
	cfg.GRPC.GetOrInsertDefault().NetAddr.Endpoint = endpoint
	return newReceiver(t, settings, cfg, otlpReceiverID, c)
}

func newHTTPReceiver(t *testing.T, settings component.TelemetrySettings, endpoint string, c consumertest.Consumer) component.Component {
	cfg := createDefaultConfig().(*Config)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.Endpoint = endpoint
	return newReceiver(t, settings, cfg, otlpReceiverID, c)
}

func newReceiver(t *testing.T, settings component.TelemetrySettings, cfg *Config, id component.ID, c consumertest.Consumer) component.Component {
	set := receivertest.NewNopSettings(metadata.Type)
	set.TelemetrySettings = settings
	set.ID = id
	r, err := newOtlpReceiver(cfg, &set)
	require.NoError(t, err)
	r.registerTraceConsumer(c)
	r.registerMetricsConsumer(c)
	r.registerLogsConsumer(c)
	r.registerProfilesConsumer(c)
	return r
}

type dataRequest struct {
	data       any
	path       string
	jsonBytes  []byte
	protoBytes []byte
}

func generateDataRequests(t *testing.T) []dataRequest {
	return []dataRequest{generateTracesRequest(t), generateMetricsRequests(t), generateLogsRequest(t), generateProfilesRequest(t)}
}

func generateTracesRequest(t *testing.T) dataRequest {
	protoMarshaler := &ptrace.ProtoMarshaler{}
	jsonMarshaler := &ptrace.JSONMarshaler{}

	td := testdata.GenerateTraces(2)
	traceProto, err := protoMarshaler.MarshalTraces(td)
	require.NoError(t, err)

	traceJSON, err := jsonMarshaler.MarshalTraces(td)
	require.NoError(t, err)

	return dataRequest{data: td, path: defaultTracesURLPath, jsonBytes: traceJSON, protoBytes: traceProto}
}

func generateMetricsRequests(t *testing.T) dataRequest {
	protoMarshaler := &pmetric.ProtoMarshaler{}
	jsonMarshaler := &pmetric.JSONMarshaler{}

	md := testdata.GenerateMetrics(2)
	metricProto, err := protoMarshaler.MarshalMetrics(md)
	require.NoError(t, err)

	metricJSON, err := jsonMarshaler.MarshalMetrics(md)
	require.NoError(t, err)

	return dataRequest{data: md, path: defaultMetricsURLPath, jsonBytes: metricJSON, protoBytes: metricProto}
}

func generateLogsRequest(t *testing.T) dataRequest {
	protoMarshaler := &plog.ProtoMarshaler{}
	jsonMarshaler := &plog.JSONMarshaler{}

	ld := testdata.GenerateLogs(2)
	logProto, err := protoMarshaler.MarshalLogs(ld)
	require.NoError(t, err)

	logJSON, err := jsonMarshaler.MarshalLogs(ld)
	require.NoError(t, err)

	return dataRequest{data: ld, path: defaultLogsURLPath, jsonBytes: logJSON, protoBytes: logProto}
}

func generateProfilesRequest(t *testing.T) dataRequest {
	protoMarshaler := &pprofile.ProtoMarshaler{}
	jsonMarshaler := &pprofile.JSONMarshaler{}

	md := testdata.GenerateProfiles(2)
	profileProto, err := protoMarshaler.MarshalProfiles(md)
	require.NoError(t, err)

	profileJSON, err := jsonMarshaler.MarshalProfiles(md)
	require.NoError(t, err)

	return dataRequest{data: md, path: defaultProfilesURLPath, jsonBytes: profileJSON, protoBytes: profileProto}
}

func doHTTPRequest(
	t *testing.T,
	url string,
	encoding string,
	contentType string,
	data []byte,
	expectStatusCode int,
) []byte {
	req := createHTTPRequest(t, url, encoding, contentType, data)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.NoError(t, resp.Body.Close())
	// For cases like "application/json; charset=utf-8", the response will be only "application/json"
	require.True(t, strings.HasPrefix(strings.ToLower(contentType), resp.Header.Get("Content-Type")))

	if expectStatusCode == 0 {
		require.Equal(t, http.StatusOK, resp.StatusCode)
	} else {
		require.Equal(t, expectStatusCode, resp.StatusCode)
	}

	return respBytes
}

func createHTTPRequest(
	t *testing.T,
	url string,
	encoding string,
	contentType string,
	data []byte,
) *http.Request {
	var buf *bytes.Buffer
	switch encoding {
	case "gzip":
		buf = compressGzip(t, data)
	case "zstd":
		buf = compressZstd(t, data)
	case "":
		buf = bytes.NewBuffer(data)
	default:
		t.Fatalf("Unsupported compression type %v", encoding)
	}

	req, err := http.NewRequest(http.MethodPost, url, buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Encoding", encoding)

	return req
}

func compressGzip(t *testing.T, body []byte) *bytes.Buffer {
	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	defer func() {
		require.NoError(t, gw.Close())
	}()

	_, err := gw.Write(body)
	require.NoError(t, err)

	return &buf
}

func compressZstd(t *testing.T, body []byte) *bytes.Buffer {
	var buf bytes.Buffer

	zw, err := zstd.NewWriter(&buf)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, zw.Close())
	}()

	_, err = zw.Write(body)
	require.NoError(t, err)

	return &buf
}

type senderFunc func(td ptrace.Traces)

func TestShutdown(t *testing.T) {
	endpointGrpc := testutil.GetAvailableLocalAddress(t)
	endpointHTTP := testutil.GetAvailableLocalAddress(t)

	nextSink := new(consumertest.TracesSink)

	// Create OTLP receiver with gRPC and HTTP protocols.
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPC.GetOrInsertDefault().NetAddr.Endpoint = endpointGrpc
	cfg.HTTP.GetOrInsertDefault().ServerConfig.Endpoint = endpointHTTP
	set := receivertest.NewNopSettings(metadata.Type)
	set.ID = otlpReceiverID
	r, err := NewFactory().CreateTraces(
		context.Background(),
		set,
		cfg,
		nextSink)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))

	conn, err := grpc.NewClient(endpointGrpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

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
		url := "http://" + endpointHTTP + defaultTracesURLPath
		req := createHTTPRequest(t, url, "", "application/x-protobuf", traceBytes)
		if resp, errResp := http.DefaultClient.Do(req); errResp == nil {
			require.NoError(t, resp.Body.Close())
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
	require.NoError(t, r.Shutdown(ctx))

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
	assert.Equal(t, sinkSpanCountAfterShutdown, nextSink.SpanCount())
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
	consumertest.Consumer
	*consumertest.TracesSink
	*consumertest.MetricsSink
	*consumertest.LogsSink
	*consumertest.ProfilesSink
	mu           sync.Mutex
	consumeError error // to be returned by ConsumeTraces, if set
}

func newErrOrSinkConsumer() *errOrSinkConsumer {
	return &errOrSinkConsumer{
		TracesSink:   new(consumertest.TracesSink),
		MetricsSink:  new(consumertest.MetricsSink),
		LogsSink:     new(consumertest.LogsSink),
		ProfilesSink: new(consumertest.ProfilesSink),
	}
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

// ConsumeLogs stores metrics to this sink.
func (esc *errOrSinkConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	if esc.consumeError != nil {
		return esc.consumeError
	}

	return esc.LogsSink.ConsumeLogs(ctx, ld)
}

// ConsumeProfiles stores profiles to this sink.
func (esc *errOrSinkConsumer) ConsumeProfiles(ctx context.Context, md pprofile.Profiles) error {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	if esc.consumeError != nil {
		return esc.consumeError
	}

	return esc.ProfilesSink.ConsumeProfiles(ctx, md)
}

// Reset deletes any stored in the sinks, resets error to nil.
func (esc *errOrSinkConsumer) Reset() {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	esc.consumeError = nil
	esc.TracesSink.Reset()
	esc.MetricsSink.Reset()
	esc.LogsSink.Reset()
	esc.ProfilesSink.Reset()
}

// Reset deletes any stored in the sinks, resets error to nil.
func (esc *errOrSinkConsumer) checkData(t *testing.T, data any, dataLen int) {
	switch data.(type) {
	case ptrace.Traces:
		allTraces := esc.AllTraces()
		require.Len(t, allTraces, dataLen)
		if dataLen > 0 {
			require.Equal(t, allTraces[0], data)
		}
	case pmetric.Metrics:
		allMetrics := esc.AllMetrics()
		require.Len(t, allMetrics, dataLen)
		if dataLen > 0 {
			require.Equal(t, allMetrics[0], data)
		}
	case plog.Logs:
		allLogs := esc.AllLogs()
		require.Len(t, allLogs, dataLen)
		if dataLen > 0 {
			require.Equal(t, allLogs[0], data)
		}
	case pprofile.Profiles:
		allProfiles := esc.AllProfiles()
		require.Len(t, allProfiles, dataLen)
		if dataLen > 0 {
			require.Equal(t, allProfiles[0], data)
		}
	}
}

func assertReceiverTraces(t *testing.T, tt *componenttest.Telemetry, id component.ID, transport string, accepted, rejected int64) {
	var refused, failed int64
	var outcome string
	gateEnabled := receiverhelper.NewReceiverMetricsGate.IsEnabled()
	// The errors in the OTLP tests are not downstream, so they should be "failed" when the gate is enabled.
	if gateEnabled {
		failed = rejected
		outcome = "failure"
	} else {
		// When the gate is disabled, all errors are "refused".
		refused = rejected
	}

	got, err := tt.GetMetric("otelcol_receiver_failed_spans")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t,
		metricdata.Metrics{
			Name:        "otelcol_receiver_failed_spans",
			Description: "The number of spans that failed to be processed by the receiver due to internal errors. [Alpha]",
			Unit:        "{spans}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String("receiver", id.String()),
							attribute.String("transport", transport)),
						Value: failed,
					},
				},
			},
		}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	got, err = tt.GetMetric("otelcol_receiver_accepted_spans")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t,
		metricdata.Metrics{
			Name:        "otelcol_receiver_accepted_spans",
			Description: "Number of spans successfully pushed into the pipeline. [Alpha]",
			Unit:        "{spans}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String("receiver", id.String()),
							attribute.String("transport", transport)),
						Value: accepted,
					},
				},
			},
		}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	got, err = tt.GetMetric("otelcol_receiver_refused_spans")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t,
		metricdata.Metrics{
			Name:        "otelcol_receiver_refused_spans",
			Description: "Number of spans that could not be pushed into the pipeline. [Alpha]",
			Unit:        "{spans}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String("receiver", id.String()),
							attribute.String("transport", transport)),
						Value: refused,
					},
				},
			},
		}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	// Assert receiver_requests metric
	if gateEnabled {
		got, err := tt.GetMetric("otelcol_receiver_requests")
		require.NoError(t, err)

		// Calculate expected requests based on accepted and refused counts
		var expectedRequests []metricdata.DataPoint[int64]
		if accepted > 0 {
			expectedRequests = append(expectedRequests, metricdata.DataPoint[int64]{
				Attributes: attribute.NewSet(
					attribute.String("receiver", id.String()),
					attribute.String("transport", transport),
					attribute.String("outcome", "success")),
				Value: accepted,
			})
		}
		if rejected > 0 {
			expectedRequests = append(expectedRequests, metricdata.DataPoint[int64]{
				Attributes: attribute.NewSet(
					attribute.String("receiver", id.String()),
					attribute.String("transport", transport),
					attribute.String("outcome", outcome)),
				Value: rejected,
			})
		}

		metricdatatest.AssertEqual(t,
			metricdata.Metrics{
				Name:        "otelcol_receiver_requests",
				Description: "The number of requests performed.",
				Unit:        "{requests}",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  expectedRequests,
				},
			}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	} else {
		_, err := tt.GetMetric("otelcol_receiver_requests")
		require.Error(t, err)
	}
}

func assertReceiverMetrics(t *testing.T, tt *componenttest.Telemetry, id component.ID, transport string, accepted, rejected int64) {
	var refused, failed int64
	var outcome string
	gateEnabled := receiverhelper.NewReceiverMetricsGate.IsEnabled()
	// The error used in the metrics test is not downstream.
	if gateEnabled {
		failed = rejected
		outcome = "failure"
	} else {
		// When the gate is disabled, all errors are "refused".
		refused = rejected
	}

	got, err := tt.GetMetric("otelcol_receiver_failed_metric_points")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t,
		metricdata.Metrics{
			Name:        "otelcol_receiver_failed_metric_points",
			Description: "The number of metric points that failed to be processed by the receiver due to internal errors. [Alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String("receiver", id.String()),
							attribute.String("transport", transport)),
						Value: failed,
					},
				},
			},
		}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	got, err = tt.GetMetric("otelcol_receiver_accepted_metric_points")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t,
		metricdata.Metrics{
			Name:        "otelcol_receiver_accepted_metric_points",
			Description: "Number of metric points successfully pushed into the pipeline. [Alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String("receiver", id.String()),
							attribute.String("transport", transport)),
						Value: accepted,
					},
				},
			},
		}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	got, err = tt.GetMetric("otelcol_receiver_refused_metric_points")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t,
		metricdata.Metrics{
			Name:        "otelcol_receiver_refused_metric_points",
			Description: "Number of metric points that could not be pushed into the pipeline. [Alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String("receiver", id.String()),
							attribute.String("transport", transport)),
						Value: refused,
					},
				},
			},
		}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	// Assert receiver_requests metric
	if gateEnabled {
		got, err := tt.GetMetric("otelcol_receiver_requests")
		require.NoError(t, err)

		// Calculate expected requests based on accepted and refused counts
		var expectedRequests []metricdata.DataPoint[int64]
		if accepted > 0 {
			expectedRequests = append(expectedRequests, metricdata.DataPoint[int64]{
				Attributes: attribute.NewSet(
					attribute.String("receiver", id.String()),
					attribute.String("transport", transport),
					attribute.String("outcome", "success")),
				Value: accepted,
			})
		}
		if rejected > 0 {
			expectedRequests = append(expectedRequests, metricdata.DataPoint[int64]{
				Attributes: attribute.NewSet(
					attribute.String("receiver", id.String()),
					attribute.String("transport", transport),
					attribute.String("outcome", outcome)),
				Value: 1, // One request failed
			})
		}

		metricdatatest.AssertEqual(t,
			metricdata.Metrics{
				Name:        "otelcol_receiver_requests",
				Description: "The number of requests performed.",
				Unit:        "{requests}",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  expectedRequests,
				},
			}, got, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	} else {
		_, err := tt.GetMetric("otelcol_receiver_requests")
		require.Error(t, err)
	}
}
