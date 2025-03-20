// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"net/http"

	"google.golang.org/grpc"
)

// HTTPClient is an interface for HTTP client middleware extensions.
type HTTPClient interface {
	// RoundTripper wraps the provided base RoundTripper.
	RoundTripper(base http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClient is an interface for gRPC client middleware extensions.
type GRPCClient interface {
	// UnaryInterceptor returns a gRPC unary client interceptor.
	UnaryInterceptor() (grpc.UnaryClientInterceptor, error)
	// StreamInterceptor returns a gRPC stream client interceptor.
	StreamInterceptor() (grpc.StreamClientInterceptor, error)
}
