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
	GetHTTPHandler(_ context.Context) (WrapHTTPHandlerFunc, error)
}

// GRPCServer defines the interface for gRPC server middleware extensions.
type GRPCServer interface {
	// GetGRPCServerOptions returns options for a gRPC server.
	GetGRPCServerOptions() ([]grpc.ServerOption, error)
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
type GetGRPCServerOptionsFunc func() ([]grpc.ServerOption, error)

func (f GetGRPCServerOptionsFunc) GetGRPCServerOptions() ([]grpc.ServerOption, error) {
	if f == nil {
		return nil, nil
	}
	return f()
}

// WrapHTTPHandlerFunc is called to initialize a new instance of
// HTTP server middleware at runtime.
type WrapHTTPHandlerFunc = func(context.Context, http.Handler) (http.Handler, error)
