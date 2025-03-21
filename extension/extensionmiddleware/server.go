// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// HTTPServer defines the interface for HTTP server middleware extensions.
type HTTPServer interface {
	// Handler wraps the provided base http.Handler.
	ServerHandler(base http.Handler) (http.Handler, error)
}

// GRPCServer defines the interface for gRPC server middleware extensions.
type GRPCServer interface {
	// ServerUnaryInterceptor returns a gRPC unary server interceptor.
	ServerUnaryInterceptor() (grpc.UnaryServerInterceptor, error)

	// ServerStreamInterceptor returns a gRPC stream server interceptor.
	ServerStreamInterceptor() (grpc.StreamServerInterceptor, error)

	// ServerStatsHandler returns a gRPC stats handler for server-side operations.
	ServerStatsHandler() (stats.Handler, error)
}
