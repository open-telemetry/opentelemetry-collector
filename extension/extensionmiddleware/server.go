// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
)

// HTTPServer defines the interface for HTTP server middleware extensions.
// This is an open interface for capability detection - extensions
// implement this interface to provide HTTP server middleware.
type HTTPServer interface {
	// GetHTTPHandler wraps the provided base http.Handler.
	GetHTTPHandler(base http.Handler) (http.Handler, error)
}

// GRPCServer defines the interface for gRPC server middleware extensions.
// This is an open interface for capability detection - extensions
// implement this interface to provide gRPC server middleware.
type GRPCServer interface {
	// GetGRPCServerOptions returns options for a gRPC server.
	GetGRPCServerOptions() ([]grpc.ServerOption, error)
}

// GRPCServerContext is an extended interface for gRPC server middleware
// that accepts a context parameter. Extensions should implement this interface
// when they need context for operations like fetching TLS credentials from
// a remote source.
//
// This interface is consumed by configgrpc via configmiddleware.
type GRPCServerContext interface {
	// GetGRPCServerOptionsContext returns the gRPC server options with context support.
	GetGRPCServerOptionsContext(context.Context) ([]grpc.ServerOption, error)
}

var _ HTTPServer = (*GetHTTPHandlerFunc)(nil)

// GetHTTPHandlerFunc is a function that implements HTTPServer.
// The nil value is a valid no-op implementation that returns base unchanged.
type GetHTTPHandlerFunc func(base http.Handler) (http.Handler, error)

// GetHTTPHandler implements HTTPServer. A nil function returns base unchanged.
func (f GetHTTPHandlerFunc) GetHTTPHandler(base http.Handler) (http.Handler, error) {
	if f == nil {
		return base, nil
	}
	return f(base)
}

var _ GRPCServer = (*GetGRPCServerOptionsFunc)(nil)

// GetGRPCServerOptionsFunc is a function that implements GRPCServer.
// The nil value is a valid no-op implementation that returns no options.
type GetGRPCServerOptionsFunc func() ([]grpc.ServerOption, error)

// GetGRPCServerOptions implements GRPCServer. A nil function returns nil options.
func (f GetGRPCServerOptionsFunc) GetGRPCServerOptions() ([]grpc.ServerOption, error) {
	if f == nil {
		return nil, nil
	}
	return f()
}

var _ GRPCServerContext = (*GetGRPCServerOptionsContextFunc)(nil)

// GetGRPCServerOptionsContextFunc is a function that implements GRPCServerContext.
// The nil value is a valid no-op implementation that returns no options.
type GetGRPCServerOptionsContextFunc func(context.Context) ([]grpc.ServerOption, error)

// GetGRPCServerOptionsContext implements GRPCServerContext. A nil function returns nil options.
func (f GetGRPCServerOptionsContextFunc) GetGRPCServerOptionsContext(ctx context.Context) ([]grpc.ServerOption, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}
