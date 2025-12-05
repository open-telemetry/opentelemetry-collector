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
		return fmt.Sprintf("error exporting items, request to %s/v1/traces responded with HTTP Status Code ", srv.URL)
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
					time.Duration(30)*time.Second).Error())
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
					assert.Contains(t, request.Header.Get("content-type"), test.expectedEncoding)
					writer.WriteHeader(http.StatusOK)
				})
				defer srv.Close()

				cfg := &Config{
					ClientConfig: confighttp.ClientConfig{
						Endpoint: srv.URL,
					},
					Encoding: test.encoding,
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
