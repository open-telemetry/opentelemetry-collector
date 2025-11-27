// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"
)

// testlientMiddleware is a mock implementation of a middleware extension
type testClientMiddleware struct {
	extension.Extension
	extensionmiddleware.GetGRPCClientOptionsFunc
}

func newTestMiddlewareConfig(name string) configmiddleware.Config {
	return configmiddleware.Config{
		ID: component.MustNewID(name),
	}
}

func newTestClientMiddleware(name string) extension.Extension {
	return &testClientMiddleware{
		Extension: extensionmiddlewaretest.NewNop(),
		GetGRPCClientOptionsFunc: func() ([]grpc.DialOption, error) {
			return []grpc.DialOption{
				grpc.WithChainUnaryInterceptor(
					func(
						ctx context.Context,
						method string,
						req, reply any,
						cc *grpc.ClientConn,
						invoker grpc.UnaryInvoker,
						opts ...grpc.CallOption,
					) error {
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
							sequence = name
						} else {
							sequence = fmt.Sprintf("%s,%s", sequence, name)
						}

						// Set the updated sequence
						md.Set("middleware-sequence", sequence)

						// Create a new context with the updated metadata
						newCtx := metadata.NewOutgoingContext(ctx, md)

						// Continue the call with our updated context
						return invoker(newCtx, method, req, reply, cc, opts...)
					}),
			}, nil
		},
	}
}

// TestClientMiddlewareOrdering verifies that client middleware
// interceptors are called in the right order.
func TestClientMiddlewareOrdering(t *testing.T) {
	// Create a middleware tracking header that will be modified by our middleware interceptors
	const middlewareTrackingHeader = "middleware-sequence"

	// Create middleware extensions that will modify the metadata to track their execution order
	mockMiddleware1 := newTestClientMiddleware("middleware-1")
	mockMiddleware2 := newTestClientMiddleware("middleware-2")

	mockExt := map[component.ID]component.Component{
		component.MustNewID("middleware1"): mockMiddleware1,
		component.MustNewID("middleware2"): mockMiddleware2,
	}

	// Start a gRPC server that will record the incoming metadata
	server := &grpcTraceServer{}
	srv, addr := server.startTestServer(t, configoptional.Some(ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "localhost:0",
			Transport: confignet.TransportTypeTCP,
		},
	}))
	defer srv.Stop()

	// Create client config with middleware extensions
	clientConfig := ClientConfig{
		Endpoint: addr,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
		Middlewares: []configmiddleware.Config{
			newTestMiddlewareConfig("middleware1"),
			newTestMiddlewareConfig("middleware2"),
		},
	}

	// Send a request using the client with middleware
	resp, err := sendTestRequestWithExtensions(t, clientConfig, mockExt)
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

// TestClientMiddlewareToClientErrors tests failure cases for the ToClient method
// specifically related to middleware resolution and API calls.
func TestClientMiddlewareToClientErrors(t *testing.T) {
	tests := []struct {
		name       string
		extensions map[component.ID]component.Component
		config     ClientConfig
		errText    string
	}{
		{
			name:       "extension_not_found",
			extensions: map[component.ID]component.Component{},
			config: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: true,
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
			name: "get_client_options_fails",
			extensions: map[component.ID]component.Component{
				component.MustNewID("errormw"): extensionmiddlewaretest.NewErr(errors.New("get options failed")),
			},
			config: ClientConfig{
				Endpoint: "localhost:1234",
				TLS: configtls.ClientConfig{
					Insecure: true,
				},
				Middlewares: []configmiddleware.Config{
					{
						ID: component.MustNewID("errormw"),
					},
				},
			},
			errText: "get options failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test creating the client with middleware errors
			_, err := tc.config.ToClientConn(context.Background(), tc.extensions, componenttest.NewNopTelemetrySettings())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}
