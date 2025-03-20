// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"net/http"

	"google.golang.org/grpc"
)

// HTTPServer defines the interface for HTTP server middleware extensions.
type HTTPServer interface {
	// Handler wraps the provided base http.Handler.
	Handler(base http.Handler) (http.Handler, error)
}

// GRPCServer defines the interface for gRPC server middleware extensions.
type GRPCServer interface {
	// UnaryInterceptor returns a gRPC unary server interceptor.
	UnaryInterceptor() (grpc.UnaryServerInterceptor, error)

	// StreamInterceptor returns a gRPC stream server interceptor.
	StreamInterceptor() (grpc.StreamServerInterceptor, error)
}
