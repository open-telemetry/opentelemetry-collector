// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configmiddleware implements the configuration settings to
// apply middleware processing on telemetry data, and allows
// components to configure middleware behavior.
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

// Middleware defines the middleware settings for a component.
type Middleware struct {
	// MiddlewareID specifies the name of the extension to use in order to apply middleware processing.
	MiddlewareID component.ID `mapstructure:"middleware,omitempty"`
}

// GetHTTPServerHandler attempts to select the appropriate extensionmiddleware.HTTPServer from the list of extensions,
// and returns the http.Handler wrapper function. If a middleware is not found, an error is returned.
func (m Middleware) GetHTTPServerHandler(_ context.Context, extensions map[component.ID]component.Component) (func(http.Handler) (http.Handler, error), error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if server, ok := ext.(extensionmiddleware.HTTPServer); ok {
			return server.GetHTTPHandler, nil
		}
		return nil, errNotHTTPServer
	}

	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}

// GetGRPCServerOptions attempts to select the appropriate extensionmiddleware.GRPCServer from the list of extensions,
// and returns the gRPC server options. If a middleware is not found, an error is returned.
func (m Middleware) GetGRPCServerOptions(_ context.Context, extensions map[component.ID]component.Component) ([]grpc.ServerOption, error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if server, ok := ext.(extensionmiddleware.GRPCServer); ok {
			return server.GetGRPCServerOptions()
		}
		return nil, errNotGRPCServer
	}

	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}

// GetHTTPClientRoundTripper attempts to select the appropriate extensionmiddleware.HTTPClient from the list of extensions,
// and returns the HTTP client transport options. If a middleware is not found, an error is returned.
// This should be only used by HTTP clients.
func (m Middleware) GetHTTPClientRoundTripper(_ context.Context, extensions map[component.ID]component.Component) (func(http.RoundTripper) (http.RoundTripper, error), error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if client, ok := ext.(extensionmiddleware.HTTPClient); ok {
			return client.GetHTTPRoundTripper, nil
		}
		return nil, errNotHTTPClient
	}
	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}

// GetGRPCClientOptions attempts to select the appropriate extensionmiddleware.GRPCClient from the list of extensions,
// and returns the gRPC dial options. If a middleware is not found, an error is returned.
// This should be only used by gRPC clients.
func (m Middleware) GetGRPCClientOptions(_ context.Context, extensions map[component.ID]component.Component) ([]grpc.DialOption, error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if client, ok := ext.(extensionmiddleware.GRPCClient); ok {
			return client.GetGRPCClientOptions()
		}
		return nil, errNotGRPCClient
	}
	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}
