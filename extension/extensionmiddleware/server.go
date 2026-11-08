// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
	"net"
	"net/http"

	"google.golang.org/grpc"
)

// HTTPServer defines the interface for HTTP server middleware extensions.
type HTTPServer interface {
	// GetHTTPHandler wraps the provided base http.Handler.
	GetHTTPHandler(_ context.Context) (WrapHTTPHandlerFunc, error)
}

// GRPCServer defines the interface for gRPC server middleware extensions.
type GRPCServer interface {
	// GetGRPCServerOptions returns options for a gRPC server.
	GetGRPCServerOptions(context.Context) ([]grpc.ServerOption, error)
}

var _ HTTPServer = (*GetHTTPHandlerFunc)(nil)

// GetHTTPHandlerFunc is a function that implements HTTPServer.
type GetHTTPHandlerFunc func(_ context.Context) (WrapHTTPHandlerFunc, error)

func (f GetHTTPHandlerFunc) GetHTTPHandler(ctx context.Context) (WrapHTTPHandlerFunc, error) {
	if f == nil {
		return func(_ context.Context, h http.Handler) (http.Handler, error) {
			return h, nil
		}, nil
	}
	return f(ctx)
}

var _ GRPCServer = (*GetGRPCServerOptionsFunc)(nil)

// GetGRPCServerOptionsFunc is a function that implements GRPCServer.
type GetGRPCServerOptionsFunc func(context.Context) ([]grpc.ServerOption, error)

func (f GetGRPCServerOptionsFunc) GetGRPCServerOptions(ctx context.Context) ([]grpc.ServerOption, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}

// WrapHTTPHandlerFunc is called to initialize a new instance of
// HTTP server middleware at runtime.
type WrapHTTPHandlerFunc = func(context.Context, http.Handler) (http.Handler, error)

// Listener is an interface for network listener extensions.
type Listener interface {
	// GetListenContext returns the function to create network listeners.
	GetListenContext(context.Context) (ListenContextFunc, error)
}

// ListenContextFunc creates a listener for the configured network and address.
type ListenContextFunc = func(ctx context.Context, network, address string) (net.Listener, error)

// GetListenerFunc is called to initialize a network listener extension.
type GetListenerFunc func(context.Context) (ListenContextFunc, error)

func (f GetListenerFunc) GetListenContext(ctx context.Context) (ListenContextFunc, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}
