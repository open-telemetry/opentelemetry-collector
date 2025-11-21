// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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

// testServerMiddleware is a test implementation of configmiddleware.Config
type testServerMiddleware struct {
	extension.Extension
	extensionmiddleware.GetHTTPHandlerFunc
}

func newTestServerMiddleware(name string) component.Component {
	return &testServerMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetHTTPHandlerFunc: func(handler http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Append middleware name to the URL path
				r.URL.Path += name + "/"

				// Call the next handler in the chain
				handler.ServeHTTP(w, r)

				// Add middleware name to the response
				_, _ = w.Write([]byte("\r\nserved by " + name))
			}), nil
		},
	}
}

func newTestServerConfig(name string) configmiddleware.Config {
	return configmiddleware.Config{
		ID: component.MustNewID(name),
	}
}

func TestServerMiddleware(t *testing.T) {
	// Register two test extensions
	extensions := map[component.ID]component.Component{
		component.MustNewID("test1"): newTestServerMiddleware("test1"),
		component.MustNewID("test2"): newTestServerMiddleware("test2"),
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
			expectedOutput: "OK{/}",
		},
		{
			name: "single_middleware",
			middlewares: []configmiddleware.Config{
				newTestServerConfig("test1"),
			},
			expectedOutput: "OK{/test1/}\r\nserved by test1",
		},
		{
			name: "multiple_middlewares",
			middlewares: []configmiddleware.Config{
				newTestServerConfig("test1"),
				newTestServerConfig("test2"),
			},
			expectedOutput: "OK{/test1/test2/}\r\nserved by test2\r\nserved by test1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create server config with the test middlewares
			cfg := ServerConfig{
				Endpoint:    "localhost:0",
				Middlewares: tc.middlewares,
			}

			// Create a test handler that responds with the request path
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("OK{" + r.URL.Path + "}"))
			})

			// Create the server
			srv, err := cfg.ToServer(
				context.Background(),
				extensions,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.NoError(t, err)

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

			// Create a response recorder
			rec := httptest.NewRecorder()

			// Serve the request
			srv.Handler.ServeHTTP(rec, req)

			// Get the response
			resp := rec.Result()
			defer resp.Body.Close()

			// Check the response
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedOutput, string(body))
		})
	}
}

func TestServerMiddlewareErrors(t *testing.T) {
	// Create a basic handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	// Test cases for HTTP server middleware errors
	httpTests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ServerConfig
		errText    string
	}{
		{
			name:       "extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ServerConfig{
				Endpoint: "localhost:0",
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_http_handler_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("http middleware error")),
			},
			config: ServerConfig{
				Endpoint: "localhost:0",
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
			// Trying to create the server should fail
			_, err := tc.config.ToServer(
				context.Background(),
				tc.extensions,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}

	// Test cases for gRPC server middleware errors
	grpcTests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ServerConfig
		errText    string
	}{
		{
			name:       "grpc_extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ServerConfig{
				Endpoint: "localhost:0",
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_grpc_handler_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("grpc middleware error")),
			},
			config: ServerConfig{
				Endpoint: "localhost:0",
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
			// Trying to create the server should fail
			_, err := tc.config.ToServer(
				context.Background(),
				tc.extensions,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}
