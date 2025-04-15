// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
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
	name string
}

func (*testServerMiddleware) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (*testServerMiddleware) Shutdown(_ context.Context) error {
	return nil
}

func (tm *testServerMiddleware) call(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	ctx = context.WithValue(ctx, middlewareCallsKey, append(getMiddlewareCalls(ctx), tm.name))
	return handler(ctx, req)
}

func (tm *testServerMiddleware) GetGRPCServerOptions() ([]grpc.ServerOption, error) {
	return []grpc.ServerOption{grpc.ChainUnaryInterceptor(tm.call)}, nil
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

func TestGrpcServerUnaryInterceptor(t *testing.T) {
	// Register two test extensions
	host := &mockHost{
		ext: map[component.ID]component.Component{
			component.MustNewID("test1"): newTestServerMiddleware("test1"),
			component.MustNewID("test2"): newTestServerMiddleware("test2"),
		},
	}

	// Setup the server with both middleware options
	server := &grpcTraceServer{}
	var addr string

	// Create the server with middleware interceptors
	{
		var srv *grpc.Server
		srv, addr = server.startTestServerWithHost(t, ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  "localhost:0",
				Transport: confignet.TransportTypeTCP,
			},
			Middlewares: []configmiddleware.Middleware{
				newTestServerConfig("test1"),
				newTestServerConfig("test2"),
			},
		}, host)
		defer srv.Stop()
	}

	// Send a request to trigger the interceptors
	resp, errResp := sendTestRequest(t, ClientConfig{
		Endpoint: addr,
		TLSSetting: configtls.ClientConfig{
			Insecure: true,
		},
	})
	require.NoError(t, errResp)
	require.NotNil(t, resp)

	// Verify interceptors were called in the correct order
	assert.Equal(t, []string{"test1", "test2"}, getMiddlewareCalls(server.recordedContext))
}
