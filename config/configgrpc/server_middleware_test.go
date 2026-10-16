// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/tap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

// contextKey is a private type for keys defined in this test.
type contextKey int

// Key for the slice of middleware names in the context.
const middlewareCallsKey contextKey = 0

// getMiddlewareCalls retrieves the middleware calls from context or returns an empty slice.
func getMiddlewareCalls(ctx context.Context) []string {
	calls, ok := ctx.Value(middlewareCallsKey).([]string)
	if !ok {
		return []string{}
	}
	return calls
}

// testServerMiddleware is a test implementation of configmiddleware.Middleware
type testServerMiddleware struct {
	extension.Extension
	extensionmiddleware.GetGRPCServerOptionsFunc
}

func newTestServerMiddleware(name string) extension.Extension {
	return &testServerMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetGRPCServerOptionsFunc: func(_ context.Context) ([]grpc.ServerOption, error) {
			return []grpc.ServerOption{grpc.ChainUnaryInterceptor(
				func(
					ctx context.Context,
					req any, _ *grpc.UnaryServerInfo,
					handler grpc.UnaryHandler,
				) (any, error) {
					ctx = context.WithValue(ctx, middlewareCallsKey, append(getMiddlewareCalls(ctx), name))
					return handler(ctx, req)
				},
			)}, nil
		},
	}
}

// testServerInTapMiddleware is a test implementation of a middleware providing a tap handle.
type testServerInTapMiddleware struct {
	extension.Extension
	extensionmiddleware.GetGRPCServerOptionsFunc
	extensionmiddleware.GetGRPCServerInTapHandleFunc
}

func newTestServerInTapMiddleware(handle tap.ServerInHandle) extension.Extension {
	return &testServerInTapMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetGRPCServerInTapHandleFunc: func(context.Context) (tap.ServerInHandle, error) {
			return handle, nil
		},
	}
}

// recordingInTapHandle returns a tap handle that records its name in the context.
func recordingInTapHandle(name string) tap.ServerInHandle {
	return func(ctx context.Context, _ *tap.Info) (context.Context, error) {
		return context.WithValue(ctx, middlewareCallsKey, append(getMiddlewareCalls(ctx), name)), nil
	}
}

func TestGrpcServerUnaryInterceptor(t *testing.T) {
	// Register two test extensions
	extensions := map[component.ID]component.Component{
		component.MustNewID("test1"): newTestServerMiddleware("test1"),
		component.MustNewID("test2"): newTestServerMiddleware("test2"),
	}

	// Setup the server with both middleware options
	server := &grpcTraceServer{}
	var addr string

	// Create the server with middleware interceptors
	{
		var srv *grpc.Server
		srv, addr = server.startTestServerWithExtensions(t, configoptional.Some(ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  "localhost:0",
				Transport: confignet.TransportTypeTCP,
			},
			Middlewares: []configmiddleware.Config{
				newTestMiddlewareConfig("test1"),
				newTestMiddlewareConfig("test2"),
			},
		}), extensions)
		defer srv.Stop()
	}

	// Send a request to trigger the interceptors
	resp, errResp := sendTestRequest(t, ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	})
	require.NoError(t, errResp)
	require.NotNil(t, resp)

	// Verify interceptors were called in the correct order
	assert.Equal(t, []string{"test1", "test2"}, getMiddlewareCalls(server.recordedContext))
}

func TestGrpcServerInTapHandle(t *testing.T) {
	extensions := map[component.ID]component.Component{
		component.MustNewID("tap1"):  newTestServerInTapMiddleware(recordingInTapHandle("tap1")),
		component.MustNewID("test1"): newTestServerMiddleware("test1"),
		component.MustNewID("tap2"):  newTestServerInTapMiddleware(recordingInTapHandle("tap2")),
	}

	server := &grpcTraceServer{}
	var addr string
	{
		var srv *grpc.Server
		srv, addr = server.startTestServerWithExtensions(t, configoptional.Some(ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  "localhost:0",
				Transport: confignet.TransportTypeTCP,
			},
			Middlewares: []configmiddleware.Config{
				newTestMiddlewareConfig("tap1"),
				newTestMiddlewareConfig("test1"),
				newTestMiddlewareConfig("tap2"),
			},
		}), extensions)
		defer srv.Stop()
	}

	resp, errResp := sendTestRequest(t, ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	})
	require.NoError(t, errResp)
	require.NotNil(t, resp)

	// Tap handles are called in the configured order, before the interceptors.
	assert.Equal(t, []string{"tap1", "tap2", "test1"}, getMiddlewareCalls(server.recordedContext))
}

func TestGrpcServerInTapHandleRejects(t *testing.T) {
	extensions := map[component.ID]component.Component{
		component.MustNewID("reject"): newTestServerInTapMiddleware(func(ctx context.Context, _ *tap.Info) (context.Context, error) {
			return ctx, status.Error(codes.ResourceExhausted, "rejected")
		}),
	}

	server := &grpcTraceServer{}
	var addr string
	{
		var srv *grpc.Server
		srv, addr = server.startTestServerWithExtensions(t, configoptional.Some(ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  "localhost:0",
				Transport: confignet.TransportTypeTCP,
			},
			Middlewares: []configmiddleware.Config{
				newTestMiddlewareConfig("reject"),
			},
		}), extensions)
		defer srv.Stop()
	}

	_, errResp := sendTestRequest(t, ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	})
	require.Error(t, errResp)
	assert.Equal(t, codes.ResourceExhausted, status.Code(errResp))
	assert.Nil(t, server.recordedContext)
}

// TestServerMiddlewareToServerErrors tests failure cases for the ToServer method
// specifically related to middleware resolution and API calls.
func TestServerMiddlewareToServerErrors(t *testing.T) {
	tests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ServerConfig
		errText    string
	}{
		{
			name:       "extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("nonexistent"),
					},
				},
			},
			errText: "failed to resolve middleware \"nonexistent\": middleware not found",
		},
		{
			name: "get_server_options_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("get server options failed")),
			},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "get server options failed",
		},
		{
			name: "get_server_tap_handle_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): &testServerInTapMiddleware{
					Extension: extensionmiddlewaretest.NewNop(),
					GetGRPCServerInTapHandleFunc: func(context.Context) (tap.ServerInHandle, error) {
						return nil, errors.New("get server tap handle failed")
					},
				},
			},
			config: ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint:  "localhost:0",
					Transport: confignet.TransportTypeTCP,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "get server tap handle failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test creating the server with middleware errors
			server := &grpcTraceServer{}
			srv, err := server.startTestServerWithExtensionsError(t, tc.config, tc.extensions)
			if srv != nil {
				srv.Stop()
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}
