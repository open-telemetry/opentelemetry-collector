// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configmiddleware implements a configuration struct to
// name middleware extensions.
package configmiddleware // import "go.opentelemetry.io/collector/config/configmiddleware"

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	errMiddlewareNotFound = errors.New("middleware not found")
	errNotHTTPServer      = errors.New("requested extension is not an HTTP server middleware")
	errNotGRPCServer      = errors.New("requested extension is not a gRPC server middleware")
	errNotHTTPClient      = errors.New("requested extension is not an HTTP client middleware")
	errNotGRPCClient      = errors.New("requested extension is not a gRPC client middleware")
)

// Middleware defines the extension ID for a middleware component.
type Config struct {
	// ID specifies the name of the extension to use.
	ID component.ID `mapstructure:"id,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// GetHTTPClientRoundTripper attempts to select the appropriate
// extensionmiddleware.HTTPClient from the map of extensions, and
// returns the HTTP client wrapper function. If a middleware is not
// found, an error is returned.  This should only be used by HTTP
// clients.
func (m Config) GetHTTPClientRoundTripper(_ context.Context, extensions map[component.ID]component.Component) (func(http.RoundTripper) (http.RoundTripper, error), error) {
	if ext, found := extensions[m.ID]; found {
		if client, ok := ext.(extensionmiddleware.HTTPClient); ok {
			return client.GetHTTPRoundTripper, nil
		}
		return nil, errNotHTTPClient
	}
	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.ID, errMiddlewareNotFound)
}

// GetHTTPServerHandler attempts to select the appropriate
// extensionmiddleware.HTTPServer from the map of extensions, and
// returns the http.Handler wrapper function. If a middleware is not
// found, an error is returned.  This should only be used by HTTP
// servers.
func (m Config) GetHTTPServerHandler(_ context.Context, extensions map[component.ID]component.Component) (func(http.Handler) (http.Handler, error), error) {
	if ext, found := extensions[m.ID]; found {
		if server, ok := ext.(extensionmiddleware.HTTPServer); ok {
			return server.GetHTTPHandler, nil
		}
		return nil, errNotHTTPServer
	}

	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.ID, errMiddlewareNotFound)
}

// GetGRPCClientOptions attempts to select the appropriate
// extensionmiddleware.GRPCClient from the map of extensions, and
// returns the gRPC dial options. If a middleware is not found, an
// error is returned.  This should only be used by gRPC clients.
func (m Config) GetGRPCClientOptions(_ context.Context, extensions map[component.ID]component.Component) ([]grpc.DialOption, error) {
	if ext, found := extensions[m.ID]; found {
		if client, ok := ext.(extensionmiddleware.GRPCClient); ok {
			return client.GetGRPCClientOptions()
		}
		return nil, errNotGRPCClient
	}
	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.ID, errMiddlewareNotFound)
}

// GetGRPCServerOptions attempts to select the appropriate
// extensionmiddleware.GRPCServer from the map of extensions, and
// returns the gRPC server options. If a middleware is not found, an
// error is returned.  This should only be used by gRPC servers.
func (m Config) GetGRPCServerOptions(_ context.Context, extensions map[component.ID]component.Component) ([]grpc.ServerOption, error) {
	if ext, found := extensions[m.ID]; found {
		if server, ok := ext.(extensionmiddleware.GRPCServer); ok {
			return server.GetGRPCServerOptions()
		}
		return nil, errNotGRPCServer
	}

	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.ID, errMiddlewareNotFound)
}
