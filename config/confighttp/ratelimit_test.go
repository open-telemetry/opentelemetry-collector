// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestServerRateLimit(t *testing.T) {
	// prepare
	hss := ServerConfig{
		Endpoint: "localhost:0",
		RateLimit: &RateLimit{
			RateLimiterID: component.NewID(component.MustNewType("mock")),
		},
	}

	limiter := &mockRateLimiter{}

	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.NewID(component.MustNewType("mock")): limiter,
		},
	}

	var handlerCalled bool
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
	})

	srv, err := hss.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings(), handler)
	require.NoError(t, err)

	// test
	srv.Handler.ServeHTTP(&httptest.ResponseRecorder{}, httptest.NewRequest("GET", "/", nil))

	// verify
	assert.True(t, handlerCalled)
	assert.Equal(t, 1, limiter.calls)
}

func TestInvalidServerRateLimit(t *testing.T) {
	hss := ServerConfig{
		RateLimit: &RateLimit{
			RateLimiterID: component.NewID(component.MustNewType("non_existing")),
		},
	}

	srv, err := hss.ToServer(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings(), http.NewServeMux())
	require.Error(t, err)
	require.Nil(t, srv)
}

func TestRejectedServerRateLimit(t *testing.T) {
	// prepare
	hss := ServerConfig{
		Endpoint: "localhost:0",
		RateLimit: &RateLimit{
			RateLimiterID: component.NewID(component.MustNewType("mock")),
		},
	}
	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.NewID(component.MustNewType("mock")): &mockRateLimiter{
				err: errors.New("rate limited"),
			},
		},
	}

	srv, err := hss.ToServer(context.Background(), host, componenttest.NewNopTelemetrySettings(), http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	require.NoError(t, err)

	// test
	response := &httptest.ResponseRecorder{}
	srv.Handler.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))

	// verify
	assert.Equal(t, response.Result().StatusCode, http.StatusTooManyRequests)
	assert.Equal(t, response.Result().Status, fmt.Sprintf("%v %s", http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests)))
}

// Mocks

type mockRateLimiter struct {
	calls int
	err   error
}

func (m *mockRateLimiter) Take(context.Context, string, http.Header) error {
	m.calls++
	return m.err
}

func (m *mockRateLimiter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (m *mockRateLimiter) Shutdown(_ context.Context) error {
	return nil
}
