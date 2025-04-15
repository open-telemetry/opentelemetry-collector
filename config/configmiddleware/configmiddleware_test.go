// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configmiddleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
)

var testID = component.MustNewID("test")

// mockComponent is a base mock that implements component.Component
type mockComponent struct{}

func (m mockComponent) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (m mockComponent) Shutdown(_ context.Context) error {
	return nil
}

// mockHTTPServerMiddleware implements extensionmiddleware.HTTPServer
type mockHTTPServerMiddleware struct {
	mockComponent
}

func (m mockHTTPServerMiddleware) GetHTTPHandler(next http.Handler) (http.Handler, error) {
	return next, nil
}

// mockHTTPClientMiddleware implements extensionmiddleware.HTTPClient
type mockHTTPClientMiddleware struct {
	mockComponent
}

func (m mockHTTPClientMiddleware) GetHTTPRoundTripper(next http.RoundTripper) (http.RoundTripper, error) {
	return next, nil
}

// mockGRPCServerMiddleware implements extensionmiddleware.GRPCServer
type mockGRPCServerMiddleware struct {
	mockComponent
}

func (m mockGRPCServerMiddleware) GetGRPCServerOptions() ([]grpc.ServerOption, error) {
	return []grpc.ServerOption{}, nil
}

// mockGRPCClientMiddleware implements extensionmiddleware.GRPCClient
type mockGRPCClientMiddleware struct {
	mockComponent
}

func (m mockGRPCClientMiddleware) GetGRPCClientOptions() ([]grpc.DialOption, error) {
	return []grpc.DialOption{}, nil
}

func TestMiddleware_GetHTTPServerHandler(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Middleware
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockHTTPServerMiddleware{},
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockComponent{},
			},
			wantErr: errNotHTTPServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.middleware.GetHTTPServerHandler(ctx, tt.extensions)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, value)
			}
		})
	}
}

func TestMiddleware_GetHTTPClientRoundTripper(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Middleware
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockHTTPClientMiddleware{},
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockComponent{},
			},
			wantErr: errNotHTTPClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.middleware.GetHTTPClientRoundTripper(ctx, tt.extensions)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, value)
			}
		})
	}
}

func TestMiddleware_GetGRPCServerOptions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Middleware
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockGRPCServerMiddleware{},
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockComponent{},
			},
			wantErr: errNotGRPCServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.middleware.GetGRPCServerOptions(ctx, tt.extensions)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, value)
			}
		})
	}
}

func TestMiddleware_GetGRPCClientOptions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Middleware
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockGRPCClientMiddleware{},
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Middleware{
				MiddlewareID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockComponent{},
			},
			wantErr: errNotGRPCClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.middleware.GetGRPCClientOptions(ctx, tt.extensions)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, value)
			}
		})
	}
}
