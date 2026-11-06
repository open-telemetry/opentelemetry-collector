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
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

const (
	tracesTelemetryType   = "traces"
	metricsTelemetryType  = "metrics"
	logsTelemetryType     = "logs"
	profilesTelemetryType = "profiles"
)

type responseSerializer interface {
	MarshalJSON() ([]byte, error)
	MarshalProto() ([]byte, error)
}

type responseSerializerProvider = func() responseSerializer

func provideTracesResponseSerializer() responseSerializer {
	response := ptraceotlp.NewExportResponse()
	partial := response.PartialSuccess()
	partial.SetErrorMessage("hello")
	partial.SetRejectedSpans(1)
	return response
}

func provideMetricsResponseSerializer() responseSerializer {
	response := pmetricotlp.NewExportResponse()
	partial := response.PartialSuccess()
	partial.SetErrorMessage("hello")
	partial.SetRejectedDataPoints(1)
	return response
}

func provideLogsResponseSerializer() responseSerializer {
	response := plogotlp.NewExportResponse()
	partial := response.PartialSuccess()
	partial.SetErrorMessage("hello")
	partial.SetRejectedLogRecords(1)
	return response
}

func provideProfilesResponseSerializer() responseSerializer {
	response := pprofileotlp.NewExportResponse()
	partial := response.PartialSuccess()
	partial.SetErrorMessage("hello")
	partial.SetRejectedProfiles(1)
	return response
}

func TestErrorResponses(t *testing.T) {
	errMsgPrefix := func(srv *httptest.Server) string {
		return fmt.Sprintf("error exporting items, request to %s responded with HTTP Status Code ", srv.URL)
	}

	tests := []struct {
		name           string
		responseStatus int
		responseBody   *status.Status
		checkErr       func(t *testing.T, err error, srv *httptest.Server)
		headers        map[string]string
	}{
		{
			name:           "400",
			responseStatus: http.StatusBadRequest,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "402",
			responseStatus: http.StatusPaymentRequired,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "404",
			responseStatus: http.StatusNotFound,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "405",
			responseStatus: http.StatusMethodNotAllowed,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "413",
			responseStatus: http.StatusRequestEntityTooLarge,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "414",
			responseStatus: http.StatusRequestURITooLong,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "431",
			responseStatus: http.StatusRequestHeaderFieldsTooLarge,
			responseBody:   status.New(codes.InvalidArgument, "Bad field"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "429",
			responseStatus: http.StatusTooManyRequests,
			responseBody:   status.New(codes.ResourceExhausted, "Quota exceeded"),
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				require.EqualError(t, err, status.New(codes.ResourceExhausted, errMsgPrefix(srv)+"429, Message=Quota exceeded, Details=[]").String())
			},
		},
		{
			name:           "429-Retry-After",
			responseStatus: http.StatusTooManyRequests,
			responseBody:   status.New(codes.InvalidArgument, "Quota exceeded"),
			headers:        map[string]string{"Retry-After": "Mon, 09 Feb 2025 15:04:05 GMT"},
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				// Cannot test for the delay part since it depends on now. Check first part (which has a negative duration) and last part:
				require.ErrorContains(t, err, "Throttle (-")
				require.ErrorContains(t, err, "), error: "+status.New(codes.ResourceExhausted, errMsgPrefix(srv)+"429, Message=Quota exceeded, Details=[]").String())
			},
		},
		{
			name:           "429-Retry-After-Malformed",
			responseStatus: http.StatusTooManyRequests,
			responseBody:   status.New(codes.InvalidArgument, "Quota exceeded"),
			headers:        map[string]string{"Retry-After": "Malformed"},
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				// Cannot test for the delay part since it depends on now. Check first part (which has a negative duration) and last part:
				require.EqualError(t, err, status.New(codes.ResourceExhausted, errMsgPrefix(srv)+"429, Message=Quota exceeded, Details=[]").String())
			},
		},
		{
			name:           "500",
			responseStatus: http.StatusInternalServerError,
			responseBody:   status.New(codes.InvalidArgument, "Internal server error"),
			checkErr: func(t *testing.T, err error, _ *httptest.Server) {
				assert.True(t, consumererror.IsPermanent(err))
			},
		},
		{
			name:           "502",
			responseStatus: http.StatusBadGateway,
			responseBody:   status.New(codes.InvalidArgument, "Bad gateway"),
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				require.EqualError(t, err, status.New(codes.Unavailable, errMsgPrefix(srv)+"502, Message=Bad gateway, Details=[]").String())
			},
		},
		{
			name:           "503",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   status.New(codes.InvalidArgument, "Server overloaded"),
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				require.EqualError(t, err, status.New(codes.Unavailable, errMsgPrefix(srv)+"503, Message=Server overloaded, Details=[]").String())
			},
		},
		{
			name:           "503-Retry-After",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   status.New(codes.InvalidArgument, "Server overloaded"),
			headers:        map[string]string{"Retry-After": "30"},
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				require.EqualError(t, err, exporterhelper.NewThrottleRetry(
					status.New(codes.Unavailable, errMsgPrefix(srv)+"503, Message=Server overloaded, Details=[]").Err(),
					time.Duration(30)*time.Second,
				).Error())
			},
		},
		{
			name:           "504",
			responseStatus: http.StatusGatewayTimeout,
			responseBody:   status.New(codes.InvalidArgument, "Gateway timeout"),
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				require.EqualError(t, err, status.New(codes.Unavailable, errMsgPrefix(srv)+"504, Message=Gateway timeout, Details=[]").String())
			},
		},
		{
			name:           "Bad response payload",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   status.New(codes.InvalidArgument, strings.Repeat("a", maxHTTPResponseReadBytes+1)),
			checkErr: func(t *testing.T, err error, srv *httptest.Server) {
				require.EqualError(t, err, status.New(codes.Unavailable, errMsgPrefix(srv)+"503").String())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := createBackend("/v1/traces", func(writer http.ResponseWriter, _ *http.Request) {
				for k, v := range test.headers {
					writer.Header().Add(k, v)
				}
				writer.WriteHeader(test.responseStatus)
				if test.responseBody != nil {
					msg, err := proto.Marshal(test.responseBody.Proto())
					assert.NoError(t, err)
					_, err = writer.Write(msg)
					assert.NoError(t, err)
				}
			})
			defer srv.Close()

			cfg := &Config{
				Encoding:       EncodingProto,
				TracesEndpoint: srv.URL + "/v1/traces",
				// Create without QueueConfig and RetryConfig so that ConsumeTraces
				// returns the errors that we want to check immediately.
			}
			exp, err := createTraces(context.Background(), exportertest.NewNopSettings(metadata.Type), cfg)
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
			require.Error(t, err)
			test.checkErr(t, err, srv)
		})
	}
}

func TestErrorResponseInvalidResponseBody(t *testing.T) {
	resp := &http.Response{
		StatusCode:    http.StatusBadRequest,
		Body:          io.NopCloser(badReader{}),
		ContentLength: 100,
	}
	assert.Nil(t, readResponseStatus(resp))
}

func TestUserAgent(t *testing.T) {
	set := exportertest.NewNopSettings(metadata.Type)
	set.BuildInfo.Description = "Collector"
	set.BuildInfo.Version = "1.2.3test"

	tests := []struct {
		name       string
		headers    configopaque.MapList
		expectedUA string
	}{
		{
			name:       "default_user_agent",
			expectedUA: "Collector/1.2.3test",
		},
		{
			name:       "custom_user_agent",
			headers:    configopaque.MapList{{Name: "User-Agent", Value: "My Custom Agent"}},
			expectedUA: "My Custom Agent",
		},
		{
			name:       "custom_user_agent_lowercase",
			headers:    configopaque.MapList{{Name: "user-agent", Value: "My Custom Agent"}},
			expectedUA: "My Custom Agent",
		},
	}

	t.Run("traces", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srv := createBackend("/v1/traces", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("user-agent"), tt.expectedUA)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					Encoding:       EncodingProto,
					TracesEndpoint: srv.URL + "/v1/traces",
					ClientConfig: confighttp.ClientConfig{
						Headers: tt.headers,
					},
				}
				exp, err := createTraces(context.Background(), set, cfg)
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
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srv := createBackend("/v1/metrics", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("user-agent"), tt.expectedUA)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					Encoding:        EncodingProto,
					MetricsEndpoint: srv.URL + "/v1/metrics",
					ClientConfig: confighttp.ClientConfig{
						Headers: tt.headers,
					},
				}
				exp, err := createMetrics(context.Background(), set, cfg)
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
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srv := createBackend("/v1/logs", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("user-agent"), tt.expectedUA)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					Encoding:     EncodingProto,
					LogsEndpoint: srv.URL + "/v1/logs",
					ClientConfig: confighttp.ClientConfig{
						Headers: tt.headers,
					},
				}
				exp, err := createLogs(context.Background(), set, cfg)
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

	t.Run("profiles", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				srv := createBackend("/v1development/profiles", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("user-agent"), test.expectedUA)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					Encoding: EncodingProto,
					ClientConfig: confighttp.ClientConfig{
						Endpoint: srv.URL,
						Headers:  test.headers,
					},
				}
				exp, err := createProfiles(context.Background(), set, cfg)
				require.NoError(t, err)

				// start the exporter
				err = exp.Start(context.Background(), componenttest.NewNopHost())
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, exp.Shutdown(context.Background()))
				})

				// generate data
				profiles := pprofile.NewProfiles()
				err = exp.ConsumeProfiles(context.Background(), profiles)
				require.NoError(t, err)
			})
		}
	})
}

func TestPartialSuccessInvalidBody(t *testing.T) {
	cfg := createDefaultConfig()
	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := newExporter(cfg, set)
	require.NoError(t, err)
	invalidBodyCases := []struct {
		telemetryType string
		handler       partialSuccessHandler
	}{
		{
			telemetryType: "traces",
			handler:       exp.tracesPartialSuccessHandler,
		},
		{
			telemetryType: "metrics",
			handler:       exp.metricsPartialSuccessHandler,
		},
		{
			telemetryType: "logs",
			handler:       exp.logsPartialSuccessHandler,
		},
		{
			telemetryType: "profiles",
			handler:       exp.profilesPartialSuccessHandler,
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
	cfg := createDefaultConfig()
	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := newExporter(cfg, set)
	require.NoError(t, err)
	unsupportedContentTypeCases := []struct {
		contentType string
	}{
		{
			contentType: "text/plain",
		},
		{
			contentType: "application/octet-stream",
		},
	}
	for _, telemetryType := range []string{"logs", "metrics", "traces", "profiles"} {
		for _, tt := range unsupportedContentTypeCases {
			t.Run("Unsupported content type "+tt.contentType+" "+telemetryType, func(t *testing.T) {
				var handler func(b []byte, contentType string) error
				switch telemetryType {
				case "logs":
					handler = exp.logsPartialSuccessHandler
				case "metrics":
					handler = exp.metricsPartialSuccessHandler
				case "traces":
					handler = exp.tracesPartialSuccessHandler
				case "profiles":
					handler = exp.profilesPartialSuccessHandler
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
	srv := createBackend("/v1/logs", func(writer http.ResponseWriter, _ *http.Request) {
		response := plogotlp.NewExportResponse()
		partial := response.PartialSuccess()
		partial.SetErrorMessage("hello")
		partial.SetRejectedLogRecords(1)
		b, err := response.MarshalProto()
		assert.NoError(t, err)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, err = writer.Write(b)
		assert.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		Encoding:     EncodingProto,
		LogsEndpoint: srv.URL + "/v1/logs",
		ClientConfig: confighttp.ClientConfig{},
	}
	set := exportertest.NewNopSettings(metadata.Type)

	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)

	exp, err := createLogs(context.Background(), set, cfg)
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
	require.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	require.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
}

func TestPartialResponse_missingHeaderButHasBody(t *testing.T) {
	cfg := createDefaultConfig()
	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := newExporter(cfg, set)
	require.NoError(t, err)

	contentTypes := []struct {
		contentType string
	}{
		{contentType: protobufContentType},
		{contentType: jsonContentType},
	}

	telemetryTypes := []struct {
		telemetryType string
		handler       partialSuccessHandler
		serializer    responseSerializerProvider
	}{
		{
			telemetryType: tracesTelemetryType,
			handler:       exp.tracesPartialSuccessHandler,
			serializer:    provideTracesResponseSerializer,
		},
		{
			telemetryType: metricsTelemetryType,
			handler:       exp.metricsPartialSuccessHandler,
			serializer:    provideMetricsResponseSerializer,
		},
		{
			telemetryType: logsTelemetryType,
			handler:       exp.logsPartialSuccessHandler,
			serializer:    provideLogsResponseSerializer,
		},
		{
			telemetryType: profilesTelemetryType,
			handler:       exp.profilesPartialSuccessHandler,
			serializer:    provideProfilesResponseSerializer,
		},
	}

	for _, ct := range contentTypes {
		for _, tt := range telemetryTypes {
			t.Run(tt.telemetryType+" "+ct.contentType, func(t *testing.T) {
				serializer := tt.serializer()

				var data []byte
				var err error

				switch ct.contentType {
				case jsonContentType:
					data, err = serializer.MarshalJSON()
				case protobufContentType:
					data, err = serializer.MarshalProto()
				default:
					require.Failf(t, "unsupported content type: %s", ct.contentType)
				}
				require.NoError(t, err)

				resp := &http.Response{
					// `-1` indicates a missing Content-Length header in the Go http standard library
					ContentLength: -1,
					Body:          io.NopCloser(bytes.NewReader(data)),
					Header: map[string][]string{
						"Content-Type": {ct.contentType},
					},
				}
				err = handlePartialSuccessResponse(resp, tt.handler)
				assert.NoError(t, err)
			})
		}
	}
}

func TestPartialResponse_missingHeaderAndBody(t *testing.T) {
	cfg := createDefaultConfig()
	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := newExporter(cfg, set)
	require.NoError(t, err)

	contentTypes := []struct {
		contentType string
	}{
		{contentType: protobufContentType},
		{contentType: jsonContentType},
	}

	telemetryTypes := []struct {
		telemetryType string
		handler       partialSuccessHandler
	}{
		{
			telemetryType: tracesTelemetryType,
			handler:       exp.tracesPartialSuccessHandler,
		},
		{
			telemetryType: metricsTelemetryType,
			handler:       exp.metricsPartialSuccessHandler,
		},
		{
			telemetryType: logsTelemetryType,
			handler:       exp.logsPartialSuccessHandler,
		},
		{
			telemetryType: profilesTelemetryType,
			handler:       exp.profilesPartialSuccessHandler,
		},
	}

	for _, ct := range contentTypes {
		for _, tt := range telemetryTypes {
			t.Run(tt.telemetryType+" "+ct.contentType, func(t *testing.T) {
				resp := &http.Response{
					// `-1` indicates a missing Content-Length header in the Go http standard library
					ContentLength: -1,
					Body:          io.NopCloser(bytes.NewReader([]byte{})),
					Header: map[string][]string{
						"Content-Type": {ct.contentType},
					},
				}
				err = handlePartialSuccessResponse(resp, tt.handler)
				assert.NoError(t, err)
			})
		}
	}
}

func TestPartialResponse_nonErrUnexpectedEOFError(t *testing.T) {
	cfg := createDefaultConfig()
	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := newExporter(cfg, set)
	require.NoError(t, err)

	resp := &http.Response{
		// `-1` indicates a missing Content-Length header in the Go http standard library
		ContentLength: -1,
		Body:          io.NopCloser(badReader{}),
	}
	err = handlePartialSuccessResponse(resp, exp.tracesPartialSuccessHandler)
	assert.Error(t, err)
}

func TestPartialSuccess_shortContentLengthHeader(t *testing.T) {
	cfg := createDefaultConfig()
	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := newExporter(cfg, set)
	require.NoError(t, err)

	contentTypes := []struct {
		contentType string
	}{
		{contentType: protobufContentType},
		{contentType: jsonContentType},
	}

	telemetryTypes := []struct {
		telemetryType string
		handler       partialSuccessHandler
		serializer    responseSerializerProvider
	}{
		{
			telemetryType: tracesTelemetryType,
			handler:       exp.tracesPartialSuccessHandler,
			serializer:    provideTracesResponseSerializer,
		},
		{
			telemetryType: metricsTelemetryType,
			handler:       exp.metricsPartialSuccessHandler,
			serializer:    provideMetricsResponseSerializer,
		},
		{
			telemetryType: logsTelemetryType,
			handler:       exp.logsPartialSuccessHandler,
			serializer:    provideLogsResponseSerializer,
		},
		{
			telemetryType: profilesTelemetryType,
			handler:       exp.profilesPartialSuccessHandler,
			serializer:    provideProfilesResponseSerializer,
		},
	}

	for _, ct := range contentTypes {
		for _, tt := range telemetryTypes {
			t.Run(tt.telemetryType+" "+ct.contentType, func(t *testing.T) {
				serializer := tt.serializer()

				var data []byte
				var err error

				switch ct.contentType {
				case jsonContentType:
					data, err = serializer.MarshalJSON()
				case protobufContentType:
					data, err = serializer.MarshalProto()
				default:
					require.Failf(t, "unsupported content type: %s", ct.contentType)
				}
				require.NoError(t, err)

				resp := &http.Response{
					ContentLength: 3,
					Body:          io.NopCloser(bytes.NewReader(data)),
					Header: map[string][]string{
						"Content-Type": {ct.contentType},
					},
				}
				// For short content-length, a real error happens.
				err = handlePartialSuccessResponse(resp, tt.handler)
				assert.Error(t, err)
			})
		}
	}
}

func TestPartialSuccess_longContentLengthHeader(t *testing.T) {
	contentTypes := []struct {
		contentType string
	}{
		{contentType: protobufContentType},
		{contentType: jsonContentType},
	}

	telemetryTypes := []struct {
		telemetryType string
		serializer    responseSerializerProvider
	}{
		{
			telemetryType: tracesTelemetryType,
			serializer:    provideTracesResponseSerializer,
		},
		{
			telemetryType: metricsTelemetryType,
			serializer:    provideMetricsResponseSerializer,
		},
		{
			telemetryType: logsTelemetryType,
			serializer:    provideLogsResponseSerializer,
		},
		{
			telemetryType: profilesTelemetryType,
			serializer:    provideProfilesResponseSerializer,
		},
	}

	for _, ct := range contentTypes {
		for _, tt := range telemetryTypes {
			t.Run(tt.telemetryType+" "+ct.contentType, func(t *testing.T) {
				cfg := createDefaultConfig()
				set := exportertest.NewNopSettings(metadata.Type)
				logger, observed := observer.New(zap.DebugLevel)
				set.Logger = zap.New(logger)
				exp, err := newExporter(cfg, set)
				require.NoError(t, err)

				serializer := tt.serializer()

				var handler partialSuccessHandler

				switch tt.telemetryType {
				case tracesTelemetryType:
					handler = exp.tracesPartialSuccessHandler
				case metricsTelemetryType:
					handler = exp.metricsPartialSuccessHandler
				case logsTelemetryType:
					handler = exp.logsPartialSuccessHandler
				case profilesTelemetryType:
					handler = exp.profilesPartialSuccessHandler
				default:
					require.Failf(t, "unsupported telemetry type: %s", ct.contentType)
				}

				var data []byte

				switch ct.contentType {
				case jsonContentType:
					data, err = serializer.MarshalJSON()
				case protobufContentType:
					data, err = serializer.MarshalProto()
				default:
					require.Failf(t, "unsupported content type: %s", ct.contentType)
				}
				require.NoError(t, err)

				resp := &http.Response{
					ContentLength: 4096,
					Body:          io.NopCloser(bytes.NewReader(data)),
					Header: map[string][]string{
						"Content-Type": {ct.contentType},
					},
				}
				// No real error happens for long content length, so the partial
				// success is handled as success with a warning.
				err = handlePartialSuccessResponse(resp, handler)
				require.NoError(t, err)
				assert.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
				assert.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
			})
		}
	}
}

func TestPartialSuccessInvalidResponseBody(t *testing.T) {
	cfg := createDefaultConfig()
	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := newExporter(cfg, set)
	require.NoError(t, err)

	resp := &http.Response{
		Body:          io.NopCloser(badReader{}),
		ContentLength: 100,
		Header: map[string][]string{
			"Content-Type": {protobufContentType},
		},
	}
	err = handlePartialSuccessResponse(resp, exp.tracesPartialSuccessHandler)
	assert.Error(t, err)
}

func TestPartialSuccess_traces(t *testing.T) {
	srv := createBackend("/v1/traces", func(writer http.ResponseWriter, _ *http.Request) {
		response := ptraceotlp.NewExportResponse()
		partial := response.PartialSuccess()
		partial.SetErrorMessage("hello")
		partial.SetRejectedSpans(1)
		bytes, err := response.MarshalProto()
		assert.NoError(t, err)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, err = writer.Write(bytes)
		assert.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		Encoding:       EncodingProto,
		TracesEndpoint: srv.URL + "/v1/traces",
		ClientConfig:   confighttp.ClientConfig{},
	}
	set := exportertest.NewNopSettings(metadata.Type)
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	exp, err := createTraces(context.Background(), set, cfg)
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
	require.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	require.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
}

func TestPartialSuccess_metrics(t *testing.T) {
	srv := createBackend("/v1/metrics", func(writer http.ResponseWriter, _ *http.Request) {
		response := pmetricotlp.NewExportResponse()
		partial := response.PartialSuccess()
		partial.SetErrorMessage("hello")
		partial.SetRejectedDataPoints(1)
		bytes, err := response.MarshalProto()
		assert.NoError(t, err)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, err = writer.Write(bytes)
		assert.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		Encoding:        EncodingProto,
		MetricsEndpoint: srv.URL + "/v1/metrics",
		ClientConfig:    confighttp.ClientConfig{},
	}
	set := exportertest.NewNopSettings(metadata.Type)
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	exp, err := createMetrics(context.Background(), set, cfg)
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
	require.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	require.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
}

func TestPartialSuccess_profiles(t *testing.T) {
	srv := createBackend("/v1development/profiles", func(writer http.ResponseWriter, _ *http.Request) {
		response := pprofileotlp.NewExportResponse()
		partial := response.PartialSuccess()
		partial.SetErrorMessage("hello")
		partial.SetRejectedProfiles(1)
		bytes, err := response.MarshalProto()
		assert.NoError(t, err)
		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, err = writer.Write(bytes)
		assert.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		Encoding: EncodingProto,
		ClientConfig: confighttp.ClientConfig{
			Endpoint: srv.URL,
		},
	}
	set := exportertest.NewNopSettings(metadata.Type)
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	exp, err := createProfiles(context.Background(), set, cfg)
	require.NoError(t, err)

	// start the exporter
	err = exp.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, exp.Shutdown(context.Background()))
	})

	// generate data
	profiles := pprofile.NewProfiles()
	err = exp.ConsumeProfiles(context.Background(), profiles)
	require.NoError(t, err)
	require.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), 1)
	require.Contains(t, observed.FilterLevelExact(zap.WarnLevel).All()[0].Message, "Partial success")
}

func TestEncoding(t *testing.T) {
	set := exportertest.NewNopSettings(metadata.Type)
	set.BuildInfo.Description = "Collector"
	set.BuildInfo.Version = "1.2.3test"

	tests := []struct {
		name             string
		encoding         EncodingType
		expectedEncoding EncodingType
	}{
		{
			name:             "proto_encoding",
			encoding:         EncodingProto,
			expectedEncoding: "application/x-protobuf",
		},
		{
			name:             "json_encoding",
			encoding:         EncodingJSON,
			expectedEncoding: "application/json",
		},
	}

	t.Run("traces", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srv := createBackend("/v1/traces", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("content-type"), tt.expectedEncoding)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					TracesEndpoint: srv.URL + "/v1/traces",
					Encoding:       tt.encoding,
				}
				exp, err := createTraces(context.Background(), set, cfg)
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
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srv := createBackend("/v1/metrics", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("content-type"), tt.expectedEncoding)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					MetricsEndpoint: srv.URL + "/v1/metrics",
					Encoding:        tt.encoding,
				}
				exp, err := createMetrics(context.Background(), set, cfg)
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
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srv := createBackend("/v1/logs", func(writer http.ResponseWriter, request *http.Request) {
					assert.Contains(t, request.Header.Get("content-type"), tt.expectedEncoding)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					LogsEndpoint: srv.URL + "/v1/logs",
					Encoding:     tt.encoding,
				}
				exp, err := createLogs(context.Background(), set, cfg)
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

	t.Run("profiles", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				srv := createBackend("/v1development/profiles", func(writer http.ResponseWriter, request *http.Request) {
					assert.Equal(t, []string{pprofileotlp.DevelopmentVersion}, request.Header.Values(pprofileotlp.DevelopmentVersionHeader))
					assert.Contains(t, request.Header.Get("content-type"), test.expectedEncoding)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				version, err := strconv.Atoi(pprofileotlp.DevelopmentVersion)
				require.NoError(t, err)
				configuredVersion := configopaque.String(strconv.Itoa(version + 1))
				otherConfiguredVersion := configopaque.String(strconv.Itoa(version + 2))
				cfg := &Config{
					ClientConfig: confighttp.ClientConfig{
						Endpoint: srv.URL,
						Headers: configopaque.MapList{
							{Name: pprofileotlp.DevelopmentVersionHeader, Value: configuredVersion},
							{Name: "OTLP-Profiles-Development-Version", Value: otherConfiguredVersion},
						},
					},
					Encoding: test.encoding,
				}
				exp, err := createProfiles(context.Background(), set, cfg)
				require.NoError(t, err)
				assert.Equal(t, configuredVersion, cfg.ClientConfig.Headers[0].Value)
				assert.Equal(t, otherConfiguredVersion, cfg.ClientConfig.Headers[1].Value)

				// start the exporter
				err = exp.Start(context.Background(), componenttest.NewNopHost())
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, exp.Shutdown(context.Background()))
				})

				// generate data
				profiles := pprofile.NewProfiles()
				err = exp.ConsumeProfiles(context.Background(), profiles)
				require.NoError(t, err)
			})
		}
	})
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

type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTripFunc(req)
}

func TestExport_ErrorShowsModifiedURL(t *testing.T) {
	t.Parallel()
	originalURL := "http://localhost:4318/placeholder-token/v1/logs"
	modifiedURL := "https://user:secret@actual-destination.example.com:443/rewritten-token/v1/logs?api_key=key#fragment" // #nosec G101 -- synthetic credentials test redaction.
	transportErr := errors.New("connection failed")
	var sentRequest *http.Request

	transport := &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, originalURL, req.URL.String())
			parsedModified, err := url.Parse(modifiedURL)
			require.NoError(t, err)
			req.URL = parsedModified
			sentRequest = req

			return nil, transportErr
		},
	}

	client := &http.Client{Transport: transport}

	logger, _ := observer.New(zap.DebugLevel)
	exp := &baseExporter{
		client: client,
		logger: zap.New(logger),
		config: &Config{
			Encoding: EncodingProto,
		},
	}

	err := exp.export(context.Background(), originalURL, []byte("test data"), nil)

	require.Error(t, err)
	require.ErrorIs(t, err, transportErr)
	assert.Contains(t, err.Error(), "https://actual-destination.example.com:443")
	assert.NotContains(t, err.Error(), "localhost")
	for _, secret := range []string{"user", "secret", "rewritten-token", "api_key", "fragment"} {
		assert.NotContains(t, err.Error(), secret)
	}
	assert.Equal(t, modifiedURL, sentRequest.URL.String())
	assert.False(t, consumererror.IsPermanent(err))
}

func TestExport_RedactsURL(t *testing.T) {
	t.Parallel()
	throttleBody, err := proto.Marshal(status.New(codes.Unavailable, "unavailable").Proto())
	require.NoError(t, err)
	endpoints := []struct {
		url      string
		expected string
	}{
		{"https://myendpoint.com/TOKEN/v1/logs", "https://myendpoint.com"},
		{"https://ingest.example.com:4318/v1/metrics?api_key=abc", "https://ingest.example.com:4318"},
		{"https://user:secret@ingest.example.com/v1/logs", "https://ingest.example.com"},
		{"http://[::1]:4318/v1/traces", "http://[::1]:4318"},
		{"https://user:secret@[::1]:443/path?api_key=abc#fragment", "https://[::1]:443"},
		{"https://ingest.example.com/path%2Ftoken?api_key=abc#fragment", "https://ingest.example.com"},
	}
	responses := []struct {
		name      string
		code      int
		permanent bool
		body      string
	}{
		{"permanent", http.StatusBadRequest, true, ""},
		{"retryable", http.StatusBadGateway, false, "not-protobuf"},
		{"throttled", http.StatusServiceUnavailable, false, string(throttleBody)},
	}
	for _, endpoint := range endpoints {
		for _, response := range responses {
			t.Run(endpoint.url+"/"+response.name, func(t *testing.T) {
				t.Parallel()
				logger, logs := observer.New(zap.DebugLevel)
				exp := &baseExporter{
					config: &Config{Encoding: EncodingJSON},
					logger: zap.New(logger),
					client: &http.Client{Transport: &mockTransport{
						roundTripFunc: func(req *http.Request) (*http.Response, error) {
							assert.Equal(t, endpoint.url, req.URL.String())
							return &http.Response{
								StatusCode:    response.code,
								Header:        http.Header{"Content-Type": {protobufContentType}, "Retry-After": {"30"}},
								ContentLength: int64(len(response.body)),
								Body:          io.NopCloser(strings.NewReader(response.body)),
							}, nil
						},
					}},
				}
				err := exp.export(t.Context(), endpoint.url, nil, nil)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "request to "+endpoint.expected+" responded with HTTP Status Code "+strconv.Itoa(response.code))
				assert.Equal(t, response.permanent, consumererror.IsPermanent(err))
				if response.name == "throttled" {
					assert.Contains(t, err.Error(), "Throttle (30s)")
					assert.Contains(t, err.Error(), "Message=unavailable")
				}
				for _, secret := range []string{"TOKEN", "api_key", "user", "secret", "/v1", "/path", "fragment"} {
					assert.NotContains(t, err.Error(), secret)
				}
				entries := logs.FilterMessage("Preparing to make HTTP request").All()
				require.Len(t, entries, 1)
				assert.Equal(t, endpoint.expected, entries[0].ContextMap()["url"])
			})
		}
	}
}

func TestExport_RedactsInvalidURL(t *testing.T) {
	t.Parallel()
	for _, endpoint := range []string{
		"https://ingest.example.com/secret%xx",
		"https://ingest.example.com:secret/path",
		"/secret/path?api_key=abc",
		"https:///secret/path?api_key=abc",
		"mailto:secret@example.com",
	} {
		t.Run(endpoint, func(t *testing.T) {
			t.Parallel()
			logger, logs := observer.New(zap.DebugLevel)
			exp := &baseExporter{
				config: &Config{Encoding: EncodingProto},
				logger: zap.New(logger),
				client: &http.Client{},
			}
			err := exp.export(t.Context(), endpoint, nil, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "[REDACTED]")
			assert.NotContains(t, err.Error(), "secret")
			assert.NotContains(t, err.Error(), "api_key")
			entries := logs.FilterMessage("Preparing to make HTTP request").All()
			require.Len(t, entries, 1)
			assert.Equal(t, "[REDACTED]", entries[0].ContextMap()["url"])
			if _, parseErr := url.Parse(endpoint); parseErr != nil {
				assert.True(t, consumererror.IsPermanent(err))
				var urlErr *url.Error
				require.ErrorAs(t, err, &urlErr)
			}
		})
	}
}

func TestExport_TransportErrorPreservesCause(t *testing.T) {
	t.Parallel()
	exp := &baseExporter{
		config: &Config{Encoding: EncodingProto},
		logger: zap.NewNop(),
		client: &http.Client{Transport: &mockTransport{
			roundTripFunc: func(*http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			},
		}},
	}
	err := exp.export(t.Context(), "https://ingest.example.com/secret?api_key=abc", nil, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.NotContains(t, err.Error(), "secret")
	assert.NotContains(t, err.Error(), "api_key")
	assert.False(t, consumererror.IsPermanent(err))
}

func TestPartialSuccess_responseBodyTooLarge(t *testing.T) {
	srv := createBackend("/v1/metrics", func(writer http.ResponseWriter, _ *http.Request) {
		response := pmetricotlp.NewExportResponse()
		response.PartialSuccess().SetErrorMessage(strings.Repeat("x", 70000))
		response.PartialSuccess().SetRejectedDataPoints(1)
		b, _ := response.MarshalProto()

		writer.Header().Set("Content-Type", "application/x-protobuf")
		writer.Header().Set("Content-Length", strconv.Itoa(len(b)))
		_, err := writer.Write(b)
		assert.NoError(t, err)
	})
	defer srv.Close()

	cfg := &Config{
		Encoding:        EncodingProto,
		MetricsEndpoint: srv.URL + "/v1/metrics",
	}
	exp, err := createMetrics(context.Background(), exportertest.NewNopSettings(metadata.Type), cfg)
	require.NoError(t, err)
	err = exp.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, exp.Shutdown(context.Background()))
	}()

	err = exp.ConsumeMetrics(context.Background(), pmetric.NewMetrics())
	require.Error(t, err)
	assert.True(t, consumererror.IsPermanent(err))
	assert.Contains(t, err.Error(), "error parsing protobuf response:")
}

func TestExport_StatusErrorShowsResolvedURL(t *testing.T) {
	configuredURL := "http://endpoint.invalid/configured-token/v1/logs?api_key=configured-key#configured-fragment"

	// Routing middleware can clone requests, so only the response carries the
	// rewritten URL. Diagnostics retain that destination without its secrets.
	cloneWithURL := func(rawURL string) func(*http.Request) *http.Request {
		return func(req *http.Request) *http.Request {
			resolved := req.Clone(req.Context())
			var err error
			resolved.URL, err = url.Parse(rawURL)
			require.NoError(t, err)
			return resolved
		}
	}

	withUserinfo, err := url.Parse("https://actual-destination.example.com:443/rewritten-token/v1/logs?api_key=rewritten-key#rewritten-fragment")
	require.NoError(t, err)
	withUserinfo.User = url.UserPassword("user", "secret")

	tests := []struct {
		name         string
		respRequest  func(*http.Request) *http.Request
		wantContains string
		wantAbsent   []string
	}{
		{
			name:         "rewritten URL taken from the response request",
			respRequest:  cloneWithURL("https://actual-destination.example.com/rewritten-token/v1/logs?api_key=rewritten-key#rewritten-fragment"),
			wantContains: "https://actual-destination.example.com",
			wantAbsent:   []string{"endpoint.invalid"},
		},
		{
			name:         "userinfo in the rewritten URL is redacted",
			respRequest:  cloneWithURL(withUserinfo.String()),
			wantContains: "https://actual-destination.example.com:443",
			wantAbsent:   []string{"user", "secret", "endpoint.invalid"},
		},
		{
			name:         "response without request falls back to the configured URL",
			respRequest:  func(*http.Request) *http.Request { return nil },
			wantContains: "http://endpoint.invalid",
		},
		{
			name:         "response request without URL falls back to the configured URL",
			respRequest:  func(*http.Request) *http.Request { return &http.Request{} },
			wantContains: "http://endpoint.invalid",
		},
	}
	statusBody, err := proto.Marshal(status.New(codes.Unavailable, "unavailable").Proto())
	require.NoError(t, err)
	responses := []struct {
		name      string
		code      int
		body      string
		permanent bool
	}{
		{"permanent", http.StatusNotFound, "", true},
		{"retryable with status", http.StatusServiceUnavailable, string(statusBody), false},
	}

	for _, tt := range tests {
		for _, response := range responses {
			t.Run(tt.name+"/"+response.name, func(t *testing.T) {
				transport := &mockTransport{
					roundTripFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode:    response.code,
							Header:        http.Header{"Content-Type": {protobufContentType}},
							Body:          io.NopCloser(strings.NewReader(response.body)),
							ContentLength: int64(len(response.body)),
							Request:       tt.respRequest(req),
						}, nil
					},
				}

				exp := &baseExporter{
					client: &http.Client{Transport: transport},
					logger: zap.NewNop(),
					config: &Config{Encoding: EncodingProto},
				}

				err := exp.export(t.Context(), configuredURL, []byte("test data"), nil)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "request to "+tt.wantContains+" responded with HTTP Status Code "+strconv.Itoa(response.code))
				assert.Equal(t, response.permanent, consumererror.IsPermanent(err))
				if response.body != "" {
					assert.Contains(t, err.Error(), "Message=unavailable")
				}
				for _, absent := range tt.wantAbsent {
					assert.NotContains(t, err.Error(), absent)
				}
				for _, secret := range []string{"configured-token", "configured-key", "configured-fragment", "rewritten-token", "rewritten-key", "rewritten-fragment", "api_key"} {
					assert.NotContains(t, err.Error(), secret)
				}
			})
		}
	}
}
