// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmiddleware"
)

// testServerMiddleware is a test implementation of configmiddleware.Middleware
type testServerMiddleware struct {
	name string
}

func (*testServerMiddleware) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (*testServerMiddleware) Shutdown(_ context.Context) error {
	return nil
}

func (tm *testServerMiddleware) GetHTTPHandler(handler http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Append middleware name to the URL path
		r.URL.Path += tm.name + "/"

		// Call the next handler in the chain
		handler.ServeHTTP(w, r)

		// Add middleware name to the response
		_, _ = w.Write([]byte("\r\nserved by " + tm.name))
	}), nil
}

func newTestServerMiddleware(name string) component.Component {
	return &testServerMiddleware{
		name: name,
	}
}

func newTestServerConfig(name string) configmiddleware.Middleware {
	return configmiddleware.Middleware{
		MiddlewareID: component.MustNewID(name),
	}
}

func TestServerMiddleware(t *testing.T) {
	// Register two test extensions
	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.MustNewID("test1"): newTestServerMiddleware("test1"),
			component.MustNewID("test2"): newTestServerMiddleware("test2"),
		},
	}

	// Test with different middleware configurations
	testCases := []struct {
		name           string
		middlewares    []configmiddleware.Middleware
		expectedOutput string
	}{
		{
			name:           "no_middlewares",
			middlewares:    nil,
			expectedOutput: "OK{/}",
		},
		{
			name: "single_middleware",
			middlewares: []configmiddleware.Middleware{
				newTestServerConfig("test1"),
			},
			expectedOutput: "OK{/test1/}\r\nserved by test1",
		},
		{
			name: "multiple_middlewares",
			middlewares: []configmiddleware.Middleware{
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
				host,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.NoError(t, err)

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/", nil)

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
