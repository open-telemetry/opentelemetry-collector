// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
)

// HTTPClient is an interface for HTTP client middleware extensions.
// This is an open interface for capability detection - extensions
// implement this interface to provide HTTP client middleware.
type HTTPClient interface {
	// GetHTTPRoundTripper wraps the provided client RoundTripper.
	GetHTTPRoundTripper(http.RoundTripper) (http.RoundTripper, error)
}

// GRPCClient is an interface for gRPC client middleware extensions.
// This is an open interface for capability detection - extensions
// implement this interface to provide gRPC client middleware.
type GRPCClient interface {
	// GetGRPCClientOptions returns the gRPC dial options to use for client connections.
	GetGRPCClientOptions() ([]grpc.DialOption, error)
}

// GRPCClientContext is an extended interface for gRPC client middleware
// that accepts a context parameter. Extensions should implement this interface
// when they need context for operations like fetching TLS credentials from
// a remote source.
//
// This interface is consumed by configgrpc via configmiddleware.
type GRPCClientContext interface {
	// GetGRPCClientOptionsContext returns the gRPC dial options with context support.
	GetGRPCClientOptionsContext(context.Context) ([]grpc.DialOption, error)
}

var _ HTTPClient = (*GetHTTPRoundTripperFunc)(nil)

// GetHTTPRoundTripperFunc is a function that implements HTTPClient.
// The nil value is a valid no-op implementation that returns the base unchanged.
type GetHTTPRoundTripperFunc func(base http.RoundTripper) (http.RoundTripper, error)

// GetHTTPRoundTripper implements HTTPClient. A nil function returns base unchanged.
func (f GetHTTPRoundTripperFunc) GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if f == nil {
		return base, nil
	}
	return f(base)
}

var _ GRPCClient = (*GetGRPCClientOptionsFunc)(nil)

// GetGRPCClientOptionsFunc is a function that implements GRPCClient.
// The nil value is a valid no-op implementation that returns no options.
type GetGRPCClientOptionsFunc func() ([]grpc.DialOption, error)

// GetGRPCClientOptions implements GRPCClient. A nil function returns nil options.
func (f GetGRPCClientOptionsFunc) GetGRPCClientOptions() ([]grpc.DialOption, error) {
	if f == nil {
		return nil, nil
	}
	return f()
}

var _ GRPCClientContext = (*GetGRPCClientOptionsContextFunc)(nil)

// GetGRPCClientOptionsContextFunc is a function that implements GRPCClientContext.
// The nil value is a valid no-op implementation that returns no options.
type GetGRPCClientOptionsContextFunc func(context.Context) ([]grpc.DialOption, error)

// GetGRPCClientOptionsContext implements GRPCClientContext. A nil function returns nil options.
func (f GetGRPCClientOptionsContextFunc) GetGRPCClientOptionsContext(ctx context.Context) ([]grpc.DialOption, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}
