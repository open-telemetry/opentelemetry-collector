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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	errMiddlewareNotFound = errors.New("middleware not found")
	errNotMiddleware      = errors.New("requested extension is not a middleware")
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

// GetHTTPServerMiddleware attempts to select the appropriate extensionmiddleware.HTTPServer from the list of extensions,
// based on the requested extension name. If a middleware is not found, an error is returned.
func (m Middleware) GetHTTPServerMiddleware(_ context.Context, extensions map[component.ID]component.Component) (extensionmiddleware.HTTPServer, error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if server, ok := ext.(extensionmiddleware.HTTPServer); ok {
			return server, nil
		}
		return nil, errNotHTTPServer
	}

	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}

// GetGRPCServerMiddleware attempts to select the appropriate extensionmiddleware.GRPCServer from the list of extensions,
// based on the requested extension name. If a middleware is not found, an error is returned.
func (m Middleware) GetGRPCServerMiddleware(_ context.Context, extensions map[component.ID]component.Component) (extensionmiddleware.GRPCServer, error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if server, ok := ext.(extensionmiddleware.GRPCServer); ok {
			return server, nil
		}
		return nil, errNotGRPCServer
	}

	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}

// GetHTTPClientMiddleware attempts to select the appropriate extensionmiddleware.HTTPClient from the list of extensions,
// based on the component id of the extension. If a middleware is not found, an error is returned.
// This should be only used by HTTP clients.
func (m Middleware) GetHTTPClientMiddleware(_ context.Context, extensions map[component.ID]component.Component) (extensionmiddleware.HTTPClient, error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if client, ok := ext.(extensionmiddleware.HTTPClient); ok {
			return client, nil
		}
		return nil, errNotHTTPClient
	}
	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}

// GetGRPCClientMiddleware attempts to select the appropriate extensionmiddleware.GRPCClient from the list of extensions,
// based on the component id of the extension. If a middleware is not found, an error is returned.
// This should be only used by gRPC clients.
func (m Middleware) GetGRPCClientMiddleware(_ context.Context, extensions map[component.ID]component.Component) (extensionmiddleware.GRPCClient, error) {
	if ext, found := extensions[m.MiddlewareID]; found {
		if client, ok := ext.(extensionmiddleware.GRPCClient); ok {
			return client, nil
		}
		return nil, errNotGRPCClient
	}
	return nil, fmt.Errorf("failed to resolve middleware %q: %w", m.MiddlewareID, errMiddlewareNotFound)
}
