// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"net/http"

	"google.golang.org/grpc"
)

// HTTPClient is an interface for HTTP client middleware extensions.
type HTTPClient interface {
	// ClientRoundTripper wraps the provided client RoundTripper.
	ClientRoundTripper(http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClient is an interface for gRPC client middleware extensions.
type GRPCClient interface {
	// ClientUnaryInterceptor returns a gRPC unary client interceptor.
	ClientUnaryInterceptor() (grpc.UnaryClientInterceptor, error)
	// ClientStreamInterceptor returns a gRPC stream client interceptor.
	ClientStreamInterceptor() (grpc.StreamClientInterceptor, error)
}
