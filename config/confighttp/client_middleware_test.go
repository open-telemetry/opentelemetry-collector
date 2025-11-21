// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

// testClientMiddleware is a test middleware that appends a string to the response body
type testClientMiddleware struct {
	extension.Extension
	extensionmiddleware.GetHTTPRoundTripperFunc
}

func newTestClientMiddleware(name string) component.Component {
	return &testClientMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetHTTPRoundTripperFunc: func(transport http.RoundTripper) (http.RoundTripper, error) {
			return extensionmiddlewaretest.RoundTripperFunc(
				func(req *http.Request) (*http.Response, error) {
					resp, err := transport.RoundTrip(req)
					if err != nil {
						return resp, err
					}

					// Read the original body
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						return resp, err
					}
					_ = resp.Body.Close()

					// Create a new body with the appended text
					newBody := string(body) + "\r\noutput by " + name

					// Replace the response body
					resp.Body = io.NopCloser(strings.NewReader(newBody))
					resp.ContentLength = int64(len(newBody))

					return resp, nil
				}), nil
		},
	}
}

func newTestClientConfig(name string) configmiddleware.Config {
	return configmiddleware.Config{
		ID: component.MustNewID(name),
	}
}

func TestClientMiddlewares(t *testing.T) {
	// Create a test server that returns "OK"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Register two test extensions
	extensions := map[component.ID]component.Component{
		component.MustNewID("test1"): newTestClientMiddleware("test1"),
		component.MustNewID("test2"): newTestClientMiddleware("test2"),
	}

	// Test with different middleware configurations
	testCases := []struct {
		name           string
		middlewares    []configmiddleware.Config
		expectedOutput string
	}{
		{
			name:           "no_middlewares",
			middlewares:    nil,
			expectedOutput: "OK",
		},
		{
			name: "single_middleware",
			middlewares: []configmiddleware.Config{
				newTestClientConfig("test1"),
			},
			expectedOutput: "OK\r\noutput by test1",
		},
		{
			name: "multiple_middlewares",
			middlewares: []configmiddleware.Config{
				newTestClientConfig("test1"),
				newTestClientConfig("test2"),
			},
			expectedOutput: "OK\r\noutput by test2\r\noutput by test1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP client config with the test middlewares
			clientConfig := ClientConfig{
				Endpoint:    server.URL,
				Middlewares: tc.middlewares,
			}

			// Create the client
			client, err := clientConfig.ToClient(context.Background(), extensions, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			// Create a request to the test server
			req, err := http.NewRequest(http.MethodGet, server.URL, http.NoBody)
			require.NoError(t, err)

			// Send the request
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Check the response
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, string(body))
		})
	}
}

func TestClientMiddlewareErrors(t *testing.T) {
	// Create a test server that returns "OK"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Test cases for HTTP client middleware errors
	httpTests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ClientConfig
		errText    string
	}{
		{
			name:       "extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ClientConfig{
				Endpoint: server.URL,
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_round_tripper_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("http middleware error")),
			},
			config: ClientConfig{
				Endpoint: server.URL,
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "http middleware error",
		},
	}

	for _, tc := range httpTests {
		t.Run(tc.name, func(t *testing.T) {
			// Trying to create the client should fail
			_, err := tc.config.ToClient(context.Background(), tc.extensions, componenttest.NewNopTelemetrySettings())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

// Test failures for gRPC client middlewares by creating a mock implementation
// that can fail in similar ways to HTTP clients
func TestGRPCClientMiddlewareErrors(t *testing.T) {
	// Test cases for gRPC client middleware errors
	grpcTests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ClientConfig
		errText    string
	}{
		{
			name:       "grpc_extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ClientConfig{
				Endpoint: "localhost:1234",
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "grpc_get_client_options_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("grpc middleware error")),
			},
			config: ClientConfig{
				Endpoint: "localhost:1234",
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "grpc middleware error",
		},
	}

	for _, tc := range grpcTests {
		t.Run(tc.name, func(t *testing.T) {
			// For gRPC, we need to use the configgrpc.ClientConfig structure
			// We'll test the middleware failure path here using the HTTP client approach,
			// as the middleware resolution logic is the same
			_, err := tc.config.ToClient(context.Background(), tc.extensions, componenttest.NewNopTelemetrySettings())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}
