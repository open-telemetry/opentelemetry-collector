// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/extensionmiddleware"

import (
	"context"
	"net"
	"net/http"

	"google.golang.org/grpc"
)

// HTTPClient is an interface for HTTP client middleware extensions.
type HTTPClient interface {
	// GetHTTPRoundTripper initializes a client HTTP RoundTripper
	// wrapper, this typically called when the Collector
	// starts. If there is an error this returns a nil
	// function. If the error is nil, the WrapHTTPRoundTripperFunc
	// will never be nil.
	GetHTTPRoundTripper(context.Context) (WrapHTTPRoundTripperFunc, error)
}

// GRPCClient is an interface for gRPC client middleware extensions.
type GRPCClient interface {
	// GetGRPCClientOptions returns the gRPC dial options to use for client connections.
	GetGRPCClientOptions(context.Context) ([]grpc.DialOption, error)
}

var _ HTTPClient = (*GetHTTPRoundTripperFunc)(nil)

// GetHTTPRoundTripperFunc is a function that implements HTTPClient.
type GetHTTPRoundTripperFunc func(context.Context) (WrapHTTPRoundTripperFunc, error)

func (f GetHTTPRoundTripperFunc) GetHTTPRoundTripper(ctx context.Context) (WrapHTTPRoundTripperFunc, error) {
	if f == nil {
		return func(_ context.Context, rt http.RoundTripper) (http.RoundTripper, error) { return rt, nil }, nil
	}
	return f(ctx)
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

// WrapHTTPRoundTripperFunc is called to initialize a new instance of
// HTTP client middleware.
type WrapHTTPRoundTripperFunc = func(context.Context, http.RoundTripper) (http.RoundTripper, error)

// Dialer is an interface for network middleware extensions.
type Dialer interface {
	// GetDialContext returns the function to dial network connections.
	GetDialContext(context.Context) (func(ctx context.Context, network, address string) (net.Conn, error), error)
}

// GetDialerFunc is called to initialize a new instance of network middleware extension.
type GetDialerFunc func(context.Context) (func(ctx context.Context, network, address string) (net.Conn, error), error)

func (f GetDialerFunc) GetDialContext(ctx context.Context) (func(ctx context.Context, network, address string) (net.Conn, error), error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}
