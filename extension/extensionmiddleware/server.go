// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
)

// HTTPServer defines the interface for HTTP server middleware extensions.
type HTTPServer interface {
	// GetHTTPHandler wraps the provided base http.Handler.
	GetHTTPHandler(_ context.Context) (func(base http.Handler) (http.Handler, error), error)
}

// GRPCServer defines the interface for gRPC server middleware extensions.
type GRPCServer interface {
	// GetGRPCServerOptions returns options for a gRPC server.
	GetGRPCServerOptions() ([]grpc.ServerOption, error)
}

var _ HTTPServer = (*GetHTTPHandlerFunc)(nil)

// GetHTTPHandlerFunc is a function that implements HTTPServer.
type GetHTTPHandlerFunc func(_ context.Context) (func(http.Handler) (http.Handler, error), error)

func (f GetHTTPHandlerFunc) GetHTTPHandler(ctx context.Context) (func(http.Handler) (http.Handler, error), error) {
	if f == nil {
		return identityHandler, nil
	}
	return f(ctx)
}

var _ GRPCServer = (*GetGRPCServerOptionsFunc)(nil)

// GetGRPCServerOptionsFunc is a function that implements GRPCServer.
type GetGRPCServerOptionsFunc func() ([]grpc.ServerOption, error)

func (f GetGRPCServerOptionsFunc) GetGRPCServerOptions() ([]grpc.ServerOption, error) {
	if f == nil {
		return nil, nil
	}
	return f()
}

func identityHandler(base http.Handler) (http.Handler, error) {
	return base, nil
}
