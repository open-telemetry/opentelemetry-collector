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
	GetHTTPRoundTripper(context.Context) (func (http.RoundTripper) (http.RoundTripper, error), error)
}

// GRPCClient is an interface for gRPC client middleware extensions.
type GRPCClient interface {
	// GetGRPCClientOptions returns the gRPC dial options to use for client connections.
	GetGRPCClientOptions() ([]grpc.DialOption, error)
}

var _ HTTPClient = (*GetHTTPRoundTripperFunc)(nil)

// GetHTTPRoundTripperFunc is a function that implements HTTPClient.
type GetHTTPRoundTripperFunc func(context.Context) (func(http.RoundTripper) (http.RoundTripper, error), error)

func (f GetHTTPRoundTripperFunc) GetHTTPRoundTripper(ctx context.Context) (func(base http.RoundTripper) (http.RoundTripper, error), error) {
	if f == nil {
		return identityRoundTripper, nil
	}
	return f(ctx)
}

var _ GRPCClient = (*GetGRPCClientOptionsFunc)(nil)

// GetGRPCClientOptionsFunc is a function that implements GRPCClient.
type GetGRPCClientOptionsFunc func() ([]grpc.DialOption, error)

func (f GetGRPCClientOptionsFunc) GetGRPCClientOptions() ([]grpc.DialOption, error) {
	if f == nil {
		return nil, nil
	}
	return f()
}


func identityRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return base, nil
}
