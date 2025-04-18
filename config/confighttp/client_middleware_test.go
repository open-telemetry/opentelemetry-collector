// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
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
			return extensionmiddlewaretest.HTTPClientFunc(
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
	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.MustNewID("test1"): newTestClientMiddleware("test1"),
			component.MustNewID("test2"): newTestClientMiddleware("test2"),
		},
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
			client, err := clientConfig.ToClient(context.Background(), host, componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)

			// Create a request to the test server
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
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
