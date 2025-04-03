// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"net/http"

	"google.golang.org/grpc"
)

// HTTPClient is an interface for HTTP client middleware extensions.
type HTTPClient interface {
	// GetHTTPRoundTripper wraps the provided client RoundTripper.
	GetHTTPRoundTripper(http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClient is an interface for gRPC client middleware extensions.
type GRPCClient interface {
	// GetGRPCClientOptions returns the gRPC dial options to use for client connections.
	GetGRPCClientOptions() ([]grpc.DialOption, error)
}
