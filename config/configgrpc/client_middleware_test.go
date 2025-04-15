// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

// TestClientMiddlewareOrdering verifies that client middleware
// interceptors are called in the right order.
func TestClientMiddlewareOrdering(t *testing.T) {
	// Create a middleware tracking header that will be modified by our middleware interceptors
	const middlewareTrackingHeader = "middleware-sequence"

	// Create middleware extensions that will modify the metadata to track their execution order
	mockMiddleware1 := &mockClientMiddleware{id: "middleware-1"}
	mockMiddleware2 := &mockClientMiddleware{id: "middleware-2"}

	mockExt := map[component.ID]component.Component{
		component.MustNewID("middleware1"): mockMiddleware1,
		component.MustNewID("middleware2"): mockMiddleware2,
	}

	// Start a gRPC server that will record the incoming metadata
	server := &grpcTraceServer{}
	srv, addr := server.startTestServer(t, ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:0",
			Transport: confignet.TransportTypeTCP,
		},
	})
	defer srv.Stop()

	// Create client config with middleware extensions
	clientConfig := ClientConfig{
		Endpoint: addr,
		TLSSetting: configtls.ClientConfig{
			Insecure: true,
		},
		Middlewares: []configmiddleware.Middleware{
			{
				MiddlewareID: component.MustNewID("middleware1"),
			},
			{
				MiddlewareID: component.MustNewID("middleware2"),
			},
		},
	}

	// Create a test host with our mock extensions
	host := &mockHost{ext: mockExt}

	// Send a request using the client with middleware
	resp, err := sendTestRequestWithHost(t, clientConfig, host)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify that the middleware order was respected as recorded in the metadata
	ictx, ok := metadata.FromIncomingContext(server.recordedContext)
	require.True(t, ok, "middleware tracking header not found in metadata")
	md := ictx[middlewareTrackingHeader]
	require.Len(t, md, 1, "expected exactly one middleware tracking header value")

	// The sequence should be "middleware-1,middleware-2" as that's the order they were registered
	expectedSequence := "middleware-1,middleware-2"
	assert.Equal(t, expectedSequence, md[0])
}

// mockClientMiddleware is a mock implementation of a middleware extension
type mockClientMiddleware struct {
	id string
}

var (
	_ component.Component            = &mockClientMiddleware{}
	_ extensionmiddleware.GRPCClient = &mockClientMiddleware{}
)

// Start implements component.Component
func (m *mockClientMiddleware) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown implements component.Component
func (m *mockClientMiddleware) Shutdown(context.Context) error {
	return nil
}

// UnaryClientInterceptor intercepts unary calls and adds middleware ID to the tracking header
func (m *mockClientMiddleware) GetGRPCClientOptions() ([]grpc.DialOption, error) {
	return []grpc.DialOption{grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Get existing metadata or create new metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			// Clone the metadata to avoid modifying the real metadata map
			md = md.Copy()
		}

		// Check if there's already a middleware sequence header
		sequence := ""
		if values := md.Get("middleware-sequence"); len(values) > 0 {
			sequence = values[0]
		}

		// Append this middleware's ID to the sequence
		if sequence == "" {
			sequence = m.id
		} else {
			sequence = fmt.Sprintf("%s,%s", sequence, m.id)
		}

		// Set the updated sequence
		md.Set("middleware-sequence", sequence)

		// Create a new context with the updated metadata
		newCtx := metadata.NewOutgoingContext(ctx, md)

		// Continue the call with our updated context
		return invoker(newCtx, method, req, reply, cc, opts...)
	})}, nil
}
