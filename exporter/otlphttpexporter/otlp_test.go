// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

func TestErrorResponses(t *testing.T) {
	errMsgPrefix := func(srv *httptest.Server) string {
		return fmt.Sprintf("error exporting items, request to %s/v1/traces responded with HTTP Status Code ", srv.URL)
	}

	tests := []struct {
		name           string
		responseStatus int
		responseBody   *status.Status
		err            func(srv *httptest.Server) error
		isPermErr      bool
		headers        map[string]string
	}{
		{
			name:           "400",
			responseStatus: http.StatusBadRequest,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			isPermErr:      true,
		},
		{
			name:           "402",
			responseStatus: http.StatusPaymentRequired,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			isPermErr:      true,
		},
		{
			name:           "404",
			responseStatus: http.StatusNotFound,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			isPermErr:      true,
		},
		{
			name:           "405",
			responseStatus: http.StatusMethodNotAllowed,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			isPermErr:      true,
		},
		{
			name:           "413",
			responseStatus: http.StatusRequestEntityTooLarge,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			isPermErr:      true,
		},
		{
			name:           "414",
			responseStatus: http.StatusRequestURITooLong,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			isPermErr:      true,
		},
		{
			name:           "431",
			responseStatus: http.StatusRequestHeaderFieldsTooLarge,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			isPermErr:      true,
		},
		{
			name:           "419",
			responseStatus: http.StatusTooManyRequests,
			responseBody:   status.New(codes.InvalidArgument, "Quota exceeded"),
			err: func(srv *httptest.Server) error {
				return exporterhelper.NewThrottleRetry(
					errors.New(errMsgPrefix(srv)+"429, Message=Quota exceeded, Details=[]"),
					time.Duration(0)*time.Second)
			},
		},
		{
			name:           "500",
			responseStatus: http.StatusInternalServerError,
			responseBody:   status.New(codes.InvalidArgument, "Internal server error"),
			isPermErr:      true,
		},
		{
			name:           "502",
			responseStatus: http.StatusBadGateway,
			responseBody:   status.New(codes.InvalidArgument, "Bad gateway"),
			err: func(srv *httptest.Server) error {
				return exporterhelper.NewThrottleRetry(
					errors.New(errMsgPrefix(srv)+"502, Message=Bad gateway, Details=[]"),
					time.Duration(0)*time.Second)
			},
		},
		{
			name:           "503",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   status.New(codes.InvalidArgument, "Server overloaded"),
			err: func(srv *httptest.Server) error {
				return exporterhelper.NewThrottleRetry(
					errors.New(errMsgPrefix(srv)+"503, Message=Server overloaded, Details=[]"),
					time.Duration(0)*time.Second)
			},
		},
		{
			name:           "503-Retry-After",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   status.New(codes.InvalidArgument, "Server overloaded"),
			headers:        map[string]string{"Retry-After": "30"},
			err: func(srv *httptest.Server) error {
				return exporterhelper.NewThrottleRetry(
					errors.New(errMsgPrefix(srv)+"503, Message=Server overloaded, Details=[]"),
					time.Duration(30)*time.Second)
			},
		},
		{
			name:           "504",
			responseStatus: http.StatusGatewayTimeout,
			responseBody:   status.New(codes.InvalidArgument, "Gateway timeout"),
			err: func(srv *httptest.Server) error {
				return exporterhelper.NewThrottleRetry(
					errors.New(errMsgPrefix(srv)+"504, Message=Gateway timeout, Details=[]"),
					time.Duration(0)*time.Second)
			},
		},
		{
			name:           "Bad response payload",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   status.New(codes.InvalidArgument, strings.Repeat("a", maxHTTPResponseReadBytes+1)),
			err: func(srv *httptest.Server) error {
				return exporterhelper.NewThrottleRetry(
					errors.New(errMsgPrefix(srv)+"503"),
					time.Duration(0)*time.Second)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := createBackend("/v1/traces", func(writer http.ResponseWriter, request *http.Request) {
				for k, v := range test.headers {
					writer.Header().Add(k, v)
				}
				writer.WriteHeader(test.responseStatus)
				if test.responseBody != nil {
					msg, err := proto.Marshal(test.responseBody.Proto())
					require.NoError(t, err)
					_, err = writer.Write(msg)
					require.NoError(t, err)
				}
			})
			defer srv.Close()

			cfg := &Config{
				TracesEndpoint: fmt.Sprintf("%s/v1/traces", srv.URL),
				// Create without QueueSettings and RetryConfig so that ConsumeTraces
				// returns the errors that we want to check immediately.
			}
			exp, err := createTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
			require.NoError(t, err)

			// start the exporter
			err = exp.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, exp.Shutdown(context.Background()))
			})

			// generate traces
			traces := ptrace.NewTraces()
			err = exp.ConsumeTraces(context.Background(), traces)
			assert.Error(t, err)

			if test.isPermErr {
				assert.True(t, consumererror.IsPermanent(err))
			} else {
				assert.EqualValues(t, test.err(srv), err)
			}
		})
	}
}

func TestErrorResponseInvalidResponseBody(t *testing.T) {
	resp := &http.Response{
		StatusCode:    400,
		Body:          io.NopCloser(badReader{}),
		ContentLength: 100,
	}
	status := readResponseStatus(resp)
	assert.Nil(t, status)
}

func TestUserAgent(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	set.BuildInfo.Description = "Collector"
	set.BuildInfo.Version = "1.2.3test"

	tests := []struct {
		name       string
		headers    map[string]configopaque.String
		expectedUA string
	}{
		{
			name:       "default_user_agent",
			expectedUA: "Collector/1.2.3test",
		},
		{
			name:       "custom_user_agent",
			headers:    map[string]configopaque.String{"User-Agent": "My Custom Agent"},
			expectedUA: "My Custom Agent",
		},
		{
			name:       "custom_user_agent_lowercase",
			headers:    map[string]configopaque.String{"user-agent": "My Custom Agent"},
			expectedUA: "My Custom Agent",
		},
	}

	t.Run("traces", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				srv := createBackend("/v1/traces", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("user-agent"), test.expectedUA)
					writer.WriteHeader(200)
				})
				defer srv.Close()

				cfg := &Config{
					TracesEndpoint: fmt.Sprintf("%s/v1/traces", srv.URL),
					HTTPClientSettings: confighttp.HTTPClientSettings{
						Headers: test.headers,
					},
				}
				exp, err := createTracesExporter(context.Background(), set, cfg)
				require.NoError(t, err)

				// start the exporter
				err = exp.Start(context.Background(), componenttest.NewNopHost())
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, exp.Shutdown(context.Background()))
				})

				// generate data
				traces := ptrace.NewTraces()
				err = exp.ConsumeTraces(context.Background(), traces)
				require.NoError(t, err)
			})
		}
	})

	t.Run("metrics", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				srv := createBackend("/v1/metrics", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("user-agent"), test.expectedUA)
					writer.WriteHeader(200)
				})
				defer srv.Close()

				cfg := &Config{
					MetricsEndpoint: fmt.Sprintf("%s/v1/metrics", srv.URL),
					HTTPClientSettings: confighttp.HTTPClientSettings{
						Headers: test.headers,
					},
				}
				exp, err := createMetricsExporter(context.Background(), set, cfg)
				require.NoError(t, err)

				// start the exporter
				err = exp.Start(context.Background(), componenttest.NewNopHost())
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, exp.Shutdown(context.Background()))
				})

				// generate data
				metrics := pmetric.NewMetrics()
				err = exp.ConsumeMetrics(context.Background(), metrics)
				require.NoError(t, err)
			})
		}
	})

	t.Run("logs", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				srv := createBackend("/v1/logs", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("user-agent"), test.expectedUA)
					writer.WriteHeader(200)
				})
				defer srv.Close()

				cfg := &Config{
					LogsEndpoint: fmt.Sprintf("%s/v1/logs", srv.URL),
					HTTPClientSettings: confighttp.HTTPClientSettings{
						Headers: test.headers,
					},
				}
				exp, err := createLogsExporter(context.Background(), set, cfg)
				require.NoError(t, err)

				// start the exporter
				err = exp.Start(context.Background(), componenttest.NewNopHost())
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, exp.Shutdown(context.Background()))
				})

				// generate data
				logs := plog.NewLogs()
				err = exp.ConsumeLogs(context.Background(), logs)
				require.NoError(t, err)

				srv.Close()
			})
		}
	})
}

func TestPartialSuccessInvalidBody(t *testing.T) {
	invalidBodyCases := []struct {
		telemetryType string
		handler       partialSuccessHandler
	}{
		{
			telemetryType: "traces",
			handler:       tracesPartialSuccessHandler,
		},
		{
			telemetryType: "metrics",
			handler:       metricsPartialSuccessHandler,
		},
		{
			telemetryType: "logs",
			handler:       logsPartialSuccessHandler,
		},
	}
	for _, tt := range invalidBodyCases {
		t.Run("Invalid response body_"+tt.telemetryType, func(t *testing.T) {
			err := tt.handler([]byte{1}, "application/x-protobuf")
			assert.ErrorContains(t, err, "error parsing protobuf response:")
		})
	}
}

func TestPartialSuccessUnsupportedContentType(t *testing.T) {
	unsupportedContentTypeCases := []struct {
		contentType string
	}{
		{
			contentType: "application/json",
		},
		{
			contentType: "text/plain",
		},
		{
			contentType: "application/octet-stream",
		},
	}
	for _, telemetryType := range []string{"logs", "metrics", "traces"} {
		for _, tt := range unsupportedContentTypeCases {
			t.Run("Unsupported content type "+tt.contentType+" "+telemetryType, func(t *testing.T) {
				var handler func(b []byte, contentType string) error
				switch telemetryType {
				case "logs":
					handler = logsPartialSuccessHandler
				case "metrics":
					handler = metricsPartialSuccessHandler
				case "traces":
					handler = tracesPartialSuccessHandler
				default:
					panic(telemetryType)
				}
				exportResponse := ptraceotlp.NewExportResponse()
				exportResponse.PartialSuccess().SetErrorMessage("foo")
				exportResponse.PartialSuccess().SetRejectedSpans(42)
				b, err := exportResponse.MarshalProto()
				require.NoError(t, err)
				err = handler(b, tt.contentType)
				assert.NoError(t, err)
			})
		}
	}
}

func TestPartialSuccess_logs(t *testing.T) {
	srv := createBackend("/v1/logs", func(writer http.ResponseWriter, request *http.Request) {
		response := plogotlp.NewExportResponse()
		partial := response.PartialSuccess()
		partial.SetErrorMessage("hello")
		partial.SetRejectedLogRecords(1)
		b, err := response.MarshalProto()
		require.NoError(t, err)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, err = writer.Write(b)
		require.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		LogsEndpoint:       fmt.Sprintf("%s/v1/logs", srv.URL),
		HTTPClientSettings: confighttp.HTTPClientSettings{},
	}
	exp, err := createLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)

	// start the exporter
	err = exp.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
	})

	// generate data
	logs := plog.NewLogs()
	err = exp.ConsumeLogs(context.Background(), logs)
	require.Error(t, err)
}

func TestPartialResponse_missingHeaderButHasBody(t *testing.T) {
	response := ptraceotlp.NewExportResponse()
	partial := response.PartialSuccess()
	partial.SetErrorMessage("hello")
	partial.SetRejectedSpans(1)
	data, err := response.MarshalProto()
	require.NoError(t, err)
	resp := &http.Response{
		// `-1` indicates a missing Content-Length header in the Go http standard library
		ContentLength: -1,
		Body:          io.NopCloser(bytes.NewReader(data)),
		Header: map[string][]string{
			"Content-Type": {"application/x-protobuf"},
		},
	}
	err = handlePartialSuccessResponse(resp, tracesPartialSuccessHandler)
	assert.True(t, consumererror.IsPermanent(err))
}

func TestPartialResponse_missingHeaderAndBody(t *testing.T) {
	resp := &http.Response{
		// `-1` indicates a missing Content-Length header in the Go http standard library
		ContentLength: -1,
		Body:          io.NopCloser(bytes.NewReader([]byte{})),
		Header: map[string][]string{
			"Content-Type": {"application/x-protobuf"},
		},
	}
	err := handlePartialSuccessResponse(resp, tracesPartialSuccessHandler)
	assert.Nil(t, err)
}

func TestPartialResponse_nonErrUnexpectedEOFError(t *testing.T) {
	resp := &http.Response{
		// `-1` indicates a missing Content-Length header in the Go http standard library
		ContentLength: -1,
		Body:          io.NopCloser(badReader{}),
	}
	err := handlePartialSuccessResponse(resp, tracesPartialSuccessHandler)
	assert.Error(t, err)
}

func TestPartialSuccess_shortContentLengthHeader(t *testing.T) {
	response := ptraceotlp.NewExportResponse()
	partial := response.PartialSuccess()
	partial.SetErrorMessage("hello")
	partial.SetRejectedSpans(1)
	data, err := response.MarshalProto()
	require.NoError(t, err)
	resp := &http.Response{
		ContentLength: 3,
		Body:          io.NopCloser(bytes.NewReader(data)),
		Header: map[string][]string{
			"Content-Type": {"application/x-protobuf"},
		},
	}
	err = handlePartialSuccessResponse(resp, tracesPartialSuccessHandler)
	assert.Error(t, err)
}

func TestPartialSuccess_longContentLengthHeader(t *testing.T) {
	response := ptraceotlp.NewExportResponse()
	partial := response.PartialSuccess()
	partial.SetErrorMessage("hello")
	partial.SetRejectedSpans(1)
	data, err := response.MarshalProto()
	require.NoError(t, err)
	resp := &http.Response{
		ContentLength: 4096,
		Body:          io.NopCloser(bytes.NewReader(data)),
		Header: map[string][]string{
			"Content-Type": {"application/x-protobuf"},
		},
	}
	err = handlePartialSuccessResponse(resp, tracesPartialSuccessHandler)
	assert.Error(t, err)
}

func TestPartialSuccessInvalidResponseBody(t *testing.T) {
	resp := &http.Response{
		Body:          io.NopCloser(badReader{}),
		ContentLength: 100,
		Header: map[string][]string{
			"Content-Type": {protobufContentType},
		},
	}
	err := handlePartialSuccessResponse(resp, tracesPartialSuccessHandler)
	assert.Error(t, err)
}

func TestPartialSuccess_traces(t *testing.T) {
	srv := createBackend("/v1/traces", func(writer http.ResponseWriter, request *http.Request) {
		response := ptraceotlp.NewExportResponse()
		partial := response.PartialSuccess()
		partial.SetErrorMessage("hello")
		partial.SetRejectedSpans(1)
		bytes, err := response.MarshalProto()
		require.NoError(t, err)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, err = writer.Write(bytes)
		require.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		TracesEndpoint:     fmt.Sprintf("%s/v1/traces", srv.URL),
		HTTPClientSettings: confighttp.HTTPClientSettings{},
	}
	exp, err := createTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)

	// start the exporter
	err = exp.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
	})

	// generate data
	traces := ptrace.NewTraces()
	err = exp.ConsumeTraces(context.Background(), traces)
	require.Error(t, err)
}

func TestPartialSuccess_metrics(t *testing.T) {
	srv := createBackend("/v1/metrics", func(writer http.ResponseWriter, request *http.Request) {
		response := pmetricotlp.NewExportResponse()
		partial := response.PartialSuccess()
		partial.SetErrorMessage("hello")
		partial.SetRejectedDataPoints(1)
		bytes, err := response.MarshalProto()
		require.NoError(t, err)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, err = writer.Write(bytes)
		require.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		MetricsEndpoint:    fmt.Sprintf("%s/v1/metrics", srv.URL),
		HTTPClientSettings: confighttp.HTTPClientSettings{},
	}
	exp, err := createMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)

	// start the exporter
	err = exp.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
	})

	// generate data
	metrics := pmetric.NewMetrics()
	err = exp.ConsumeMetrics(context.Background(), metrics)
	require.Error(t, err)
}

func createBackend(endpoint string, handler func(writer http.ResponseWriter, request *http.Request)) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, handler)

	srv := httptest.NewServer(mux)

	return srv
}

type badReader struct{}

func (b badReader) Read([]byte) (int, error) {
	return 0, errors.New("Bad read")
}
