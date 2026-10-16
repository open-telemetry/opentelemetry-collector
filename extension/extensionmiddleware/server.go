// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/tap"
)

// HTTPServer defines the interface for HTTP server middleware extensions.
type HTTPServer interface {
	// GetHTTPHandler wraps the provided base http.Handler.
	GetHTTPHandler(_ context.Context) (WrapHTTPHandlerFunc, error)
}

// GRPCServer defines the interface for gRPC server middleware extensions.
type GRPCServer interface {
	// GetGRPCServerOptions returns options for a gRPC server.
	//
	// The options must not include grpc.InTapHandle, because a gRPC server
	// accepts only one tap handle. Implement GRPCServerInTap instead.
	GetGRPCServerOptions(context.Context) ([]grpc.ServerOption, error)
}

// GRPCServerInTap is an optional interface for gRPC server middleware
// extensions that need to check a request before the server reads the
// request message.
//
// gRPC servers accept a single tap handle, so middleware extensions must
// implement this interface instead of returning grpc.InTapHandle from
// GetGRPCServerOptions. The tap handles of all configured middlewares are
// combined into one and called in the configured order.
type GRPCServerInTap interface {
	// GetGRPCServerInTapHandle returns a tap handle for a gRPC server,
	// or nil if the extension does not need one.
	GetGRPCServerInTapHandle(context.Context) (tap.ServerInHandle, error)
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

var _ GRPCServerInTap = (*GetGRPCServerInTapHandleFunc)(nil)

// GetGRPCServerInTapHandleFunc is a function that implements GRPCServerInTap.
type GetGRPCServerInTapHandleFunc func(context.Context) (tap.ServerInHandle, error)

func (f GetGRPCServerInTapHandleFunc) GetGRPCServerInTapHandle(ctx context.Context) (tap.ServerInHandle, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}

// WrapHTTPHandlerFunc is called to initialize a new instance of
// HTTP server middleware at runtime.
type WrapHTTPHandlerFunc = func(context.Context, http.Handler) (http.Handler, error)
