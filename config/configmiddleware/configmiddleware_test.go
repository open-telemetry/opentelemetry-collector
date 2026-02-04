// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configmiddleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

var testID = component.MustNewID("test")

type mockWrongType struct {
	component.StartFunc
	component.ShutdownFunc
}

func TestConfig_GetHTTPServerHandler(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Config
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: extensionmiddlewaretest.NewNop(),
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockWrongType{},
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
				require.NotNil(t, value)
			}
		})
	}
}

func TestConfig_GetHTTPClientRoundTripper(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Config
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: extensionmiddlewaretest.NewNop(),
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockWrongType{},
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
				require.NotNil(t, value)
			}
		})
	}
}

func TestConfig_GetGRPCServerOptions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Config
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: struct {
					extension.Extension
					extensionmiddleware.GetGRPCServerOptionsFunc
				}{
					Extension: extensionmiddlewaretest.NewNop(),
					GetGRPCServerOptionsFunc: func() ([]grpc.ServerOption, error) {
						return []grpc.ServerOption{
							grpc.EmptyServerOption{},
						}, nil
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "found_and_valid_context",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: struct {
					extension.Extension
					extensionmiddleware.GetGRPCServerOptionsContextFunc
				}{
					Extension: extensionmiddlewaretest.NewNop(),
					GetGRPCServerOptionsContextFunc: func(context.Context) ([]grpc.ServerOption, error) {
						return []grpc.ServerOption{
							grpc.EmptyServerOption{},
						}, nil
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockWrongType{},
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
				require.NotNil(t, value)
			}
		})
	}
}

func TestConfig_GetGRPCClientOptions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		middleware Config
		extensions map[component.ID]component.Component
		wantErr    error
	}{
		{
			name: "found_and_valid",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: struct {
					extension.Extension
					extensionmiddleware.GetGRPCClientOptionsFunc
				}{
					Extension: extensionmiddlewaretest.NewNop(),
					GetGRPCClientOptionsFunc: func() ([]grpc.DialOption, error) {
						return []grpc.DialOption{
							grpc.EmptyDialOption{},
						}, nil
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "found_and_valid_context",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: struct {
					extension.Extension
					extensionmiddleware.GetGRPCClientOptionsContextFunc
				}{
					Extension: extensionmiddlewaretest.NewNop(),
					GetGRPCClientOptionsContextFunc: func(context.Context) ([]grpc.DialOption, error) {
						return []grpc.DialOption{
							grpc.EmptyDialOption{},
						}, nil
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "middleware_not_found",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{},
			wantErr:    errMiddlewareNotFound,
		},
		{
			name: "middleware_wrong_type",
			middleware: Config{
				ID: testID,
			},
			extensions: map[component.ID]component.Component{
				testID: mockWrongType{},
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
				require.NotNil(t, value)
			}
		})
	}
}

func TestConfig_GetGRPCClientOptions_ContextPassed(t *testing.T) {
	type ctxKey struct{}
	testCtx := context.WithValue(context.Background(), ctxKey{}, "test-value")
	var receivedCtx context.Context

	middleware := Config{ID: testID}
	extensions := map[component.ID]component.Component{
		testID: struct {
			extension.Extension
			extensionmiddleware.GetGRPCClientOptionsContextFunc
		}{
			Extension: extensionmiddlewaretest.NewNop(),
			GetGRPCClientOptionsContextFunc: func(ctx context.Context) ([]grpc.DialOption, error) {
				receivedCtx = ctx
				return []grpc.DialOption{grpc.EmptyDialOption{}}, nil
			},
		},
	}

	_, err := middleware.GetGRPCClientOptions(testCtx, extensions)
	require.NoError(t, err)
	require.Equal(t, "test-value", receivedCtx.Value(ctxKey{}))
}

func TestConfig_GetGRPCServerOptions_ContextPassed(t *testing.T) {
	type ctxKey struct{}
	testCtx := context.WithValue(context.Background(), ctxKey{}, "server-test-value")
	var receivedCtx context.Context

	middleware := Config{ID: testID}
	extensions := map[component.ID]component.Component{
		testID: struct {
			extension.Extension
			extensionmiddleware.GetGRPCServerOptionsContextFunc
		}{
			Extension: extensionmiddlewaretest.NewNop(),
			GetGRPCServerOptionsContextFunc: func(ctx context.Context) ([]grpc.ServerOption, error) {
				receivedCtx = ctx
				return []grpc.ServerOption{grpc.EmptyServerOption{}}, nil
			},
		},
	}

	_, err := middleware.GetGRPCServerOptions(testCtx, extensions)
	require.NoError(t, err)
	require.Equal(t, "server-test-value", receivedCtx.Value(ctxKey{}))
}
