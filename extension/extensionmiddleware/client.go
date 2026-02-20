// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
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
	GetGRPCClientOptions(context.Context) ([]grpc.DialOption, error)
}

var _ HTTPClient = (*GetHTTPRoundTripperFunc)(nil)

// GetHTTPRoundTripperFunc is a function that implements HTTPClient.
type GetHTTPRoundTripperFunc func(base http.RoundTripper) (http.RoundTripper, error)

func (f GetHTTPRoundTripperFunc) GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if f == nil {
		return base, nil
	}
	return f(base)
}

var _ GRPCClient = (*GetGRPCClientOptionsFunc)(nil)

// GetGRPCClientOptionsFunc is a function that implements GRPCClient.
type GetGRPCClientOptionsFunc func(context.Context) ([]grpc.DialOption, error)

func (f GetGRPCClientOptionsFunc) GetGRPCClientOptions(ctx context.Context) ([]grpc.DialOption, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}
