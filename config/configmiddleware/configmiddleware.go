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

	"go.uber.org/multierr"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"
	grpclimiter "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper/grpc"
	httplimiter "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper/http"
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

func resolveFailed(id component.ID) error {
	return fmt.Errorf("failed to resolve middleware %q: %w", id, errMiddlewareNotFound)
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
		if limiter, ok := ext.(limiterhelper.AnyProvider); ok {
			limiter, err := httplimiter.NewClientLimiter(limiter)
			if err != nil {
				return nil, err
			}
			return limiter.GetHTTPRoundTripper, nil
		}
		return nil, errNotHTTPClient
	}
	return nil, resolveFailed(m.ID)
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
		if limiter, ok := ext.(limiterhelper.AnyProvider); ok {
			limiter, err := httplimiter.NewServerLimiter(limiter)
			if err != nil {
				return nil, err
			}
			return limiter.GetHTTPHandler, nil
		}
		return nil, errNotHTTPServer
	}

	return nil, resolveFailed(m.ID)
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
		if limiter, ok := ext.(limiterhelper.AnyProvider); ok {
			lim, err := grpclimiter.NewClientLimiter(limiter)
			if err != nil {
				return nil, err
			}
			return lim.GetGRPCClientOptions()
		}
		return nil, errNotGRPCClient
	}
	return nil, resolveFailed(m.ID)
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
		if limiter, ok := ext.(limiterhelper.AnyProvider); ok {
			lim, err := grpclimiter.NewServerLimiter(limiter)
			if err != nil {
				return nil, err
			}
			return lim.GetGRPCServerOptions()
		}
		return nil, errNotGRPCServer
	}

	return nil, resolveFailed(m.ID)
}

// GetBaseLimiters gets a list of basic limiters.  These can be
// upgraded to any kind of limiter, subject to restrictions documented
// in that extension interface, using limiterhelper.MiddlewaresToLimiterProvider.
func GetBaseLimiters(host component.Host, cfgs []Config) ([]limiterhelper.AnyProvider, error) {
	var err error
	var lims []limiterhelper.AnyProvider
	all := host.GetExtensions()
	for _, m := range cfgs {
		ext, ok := all[m.ID]
		if !ok {
			err = multierr.Append(err, resolveFailed(m.ID))
			continue
		}
		if lim, ok := ext.(limiterhelper.AnyProvider); ok {
			lims = append(lims, lim)
		} else {
			// Note: In this case, we skip the middleware
			// that is not a limiter.
		}
	}
	return lims, err
}
