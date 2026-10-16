// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

func TestExportEndpointDiagnostics(t *testing.T) {
	const requestURL = "https://diagnostic-user:diagnostic-password@[2001:db8::1]:4318/receiver/path-secret/v1/traces?token=query-secret#fragment-secret"
	const origin = "https://[2001:db8::1]:4318"
	for _, logFullURL := range []bool{true, false} {
		for _, test := range []struct {
			name       string
			statusCode int
			statusBody bool
			permanent  bool
			throttled  bool
		}{
			{name: "success", statusCode: http.StatusOK},
			{name: "permanent", statusCode: http.StatusBadRequest, permanent: true},
			{name: "retryable", statusCode: http.StatusBadGateway},
			{name: "status_body", statusCode: http.StatusServiceUnavailable, statusBody: true},
			{name: "throttled", statusCode: http.StatusTooManyRequests, statusBody: true, throttled: true},
		} {
			t.Run(fmt.Sprintf("%s/log_full_url=%t", test.name, logFullURL), func(t *testing.T) {
				var responseBody string
				if test.statusCode != http.StatusOK {
					responseBody = "not an OTLP status"
				}
				if test.statusBody {
					body, err := proto.Marshal(status.New(codes.Unavailable, "backend unavailable").Proto())
					require.NoError(t, err)
					responseBody = string(body)
				}
				core, logs := observer.New(zap.DebugLevel)
				exp := &baseExporter{
					config: &Config{Encoding: EncodingProto, LogFullURL: logFullURL},
					logger: zap.New(core),
					client: &http.Client{Transport: &mockTransport{roundTripFunc: func(req *http.Request) (*http.Response, error) {
						// The diagnostic option must not change the destination or request body.
						assert.Equal(t, requestURL, req.URL.String())
						body, err := io.ReadAll(req.Body)
						require.NoError(t, err)
						assert.Equal(t, "telemetry", string(body))
						header := http.Header{}
						if test.throttled {
							header.Set(headerRetryAfter, "30")
						}
						return &http.Response{
							StatusCode:    test.statusCode,
							Header:        header,
							Body:          io.NopCloser(strings.NewReader(responseBody)),
							ContentLength: int64(len(responseBody)),
						}, nil
					}}},
				}
				err := exp.export(t.Context(), requestURL, []byte("telemetry"), exp.tracesPartialSuccessHandler)
				if test.statusCode == http.StatusOK {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					assert.Equal(t, test.permanent, consumererror.IsPermanent(err))
					require.ErrorContains(t, err, fmt.Sprintf("HTTP Status Code %d", test.statusCode))
					if test.statusBody {
						require.ErrorContains(t, err, "Message=backend unavailable")
					}
					if test.throttled {
						require.ErrorContains(t, err, "Throttle (30s), error:")
					} else {
						assert.NotContains(t, err.Error(), "Throttle (")
					}
				}
				expectedURL := requestURL
				if !logFullURL {
					expectedURL = origin
				}
				entries := logs.FilterMessage("Preparing to make HTTP request").All()
				require.Len(t, entries, 1)
				assert.Equal(t, expectedURL, entries[0].ContextMap()["url"])
				if err != nil {
					require.ErrorContains(t, err, "request to "+expectedURL+" responded")
					if !logFullURL {
						for _, secret := range []string{"diagnostic-user", "diagnostic-password", "path-secret", "query-secret", "fragment-secret"} {
							assert.NotContains(t, err.Error(), secret)
						}
					}
				}
			})
		}
	}
}

func TestExportEndpointDiagnosticsTransportError(t *testing.T) {
	const originalURL = "https://original.example:4318/path-secret?token=query-secret"
	const modifiedURL = "https://diagnostic-user:diagnostic-password@actual.example:8443/redirect-secret?token=redirect-token" // #nosec G101 -- Synthetic credentials exercise redaction.
	for _, logFullURL := range []bool{true, false} {
		t.Run(fmt.Sprintf("log_full_url=%t", logFullURL), func(t *testing.T) {
			transportErr := errors.New("connection failed")
			var actualRequest *http.Request
			core, logs := observer.New(zap.DebugLevel)
			exp := &baseExporter{
				config: &Config{Encoding: EncodingProto, LogFullURL: logFullURL},
				logger: zap.New(core),
				client: &http.Client{Transport: &mockTransport{roundTripFunc: func(req *http.Request) (*http.Response, error) {
					actualRequest = req
					var err error
					req.URL, err = url.Parse(modifiedURL)
					require.NoError(t, err)
					return nil, transportErr
				}}},
			}
			err := exp.export(t.Context(), originalURL, []byte("telemetry"), nil)
			require.Error(t, err)
			require.ErrorIs(t, err, transportErr)
			assert.False(t, consumererror.IsPermanent(err))
			var urlErr *url.Error
			require.ErrorAs(t, err, &urlErr)
			expectedErrorURL := modifiedURL
			expectedLogURL := originalURL
			if !logFullURL {
				expectedErrorURL = "https://actual.example:8443"
				expectedLogURL = "https://original.example:4318"
			}
			assert.Equal(t, expectedErrorURL, urlErr.URL)
			require.ErrorContains(t, err, expectedErrorURL)
			assert.Equal(t, modifiedURL, actualRequest.URL.String())
			entries := logs.FilterMessage("Preparing to make HTTP request").All()
			require.Len(t, entries, 1)
			assert.Equal(t, expectedLogURL, entries[0].ContextMap()["url"])
			if !logFullURL {
				for _, secret := range []string{"diagnostic-user", "diagnostic-password", "redirect-secret", "redirect-token", "path-secret", "query-secret"} {
					assert.NotContains(t, err.Error(), secret)
				}
			}
		})
	}
}

func TestExportEndpointDiagnosticsInvalidURL(t *testing.T) {
	const requestURL = "https://example.com/path-secret?token=query-secret\n"
	for _, logFullURL := range []bool{true, false} {
		t.Run(fmt.Sprintf("log_full_url=%t", logFullURL), func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			exp := &baseExporter{
				config: &Config{Encoding: EncodingProto, LogFullURL: logFullURL},
				logger: zap.New(core),
			}
			err := exp.export(t.Context(), requestURL, nil, nil)
			require.Error(t, err)
			assert.True(t, consumererror.IsPermanent(err))
			var urlErr *url.Error
			require.ErrorAs(t, err, &urlErr)
			expectedURL := requestURL
			if !logFullURL {
				expectedURL = "[REDACTED]"
				assert.NotContains(t, err.Error(), "path-secret")
				assert.NotContains(t, err.Error(), "query-secret")
			}
			assert.Equal(t, expectedURL, urlErr.URL)
			entries := logs.FilterMessage("Preparing to make HTTP request").All()
			require.Len(t, entries, 1)
			assert.Equal(t, expectedURL, entries[0].ContextMap()["url"])
		})
	}
}
