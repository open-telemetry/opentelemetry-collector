// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
	"errors"
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

// HTTPServerProtocols is an optional interface for HTTP server middleware
// extensions that need the HTTP server to accept more protocols, such as
// unencrypted HTTP/2. The returned protocols are added to the server's default
// protocols. An extension cannot disable a protocol.
type HTTPServerProtocols interface {
	// GetHTTPServerProtocols returns the protocols to enable on the HTTP server.
	GetHTTPServerProtocols(context.Context) (http.Protocols, error)
}

// HTTPServerProtocolConfig is an optional interface for middleware configuration
// to declare protocol requirements without creating or starting the extension.
// The declaration must match the runtime HTTPServerProtocols request.
type HTTPServerProtocolConfig interface {
	RequiredHTTPServerProtocols() http.Protocols
}

var _ HTTPServerProtocols = (*GetHTTPServerProtocolsFunc)(nil)

// GetHTTPServerProtocolsFunc is a function that implements HTTPServerProtocols.
type GetHTTPServerProtocolsFunc func(context.Context) (http.Protocols, error)

func (f GetHTTPServerProtocolsFunc) GetHTTPServerProtocols(ctx context.Context) (http.Protocols, error) {
	if f == nil {
		return http.Protocols{}, nil
	}
	return f(ctx)
}

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
		return nil, errors.New("listener is not initialized")
	}
	return f(ctx)
}
