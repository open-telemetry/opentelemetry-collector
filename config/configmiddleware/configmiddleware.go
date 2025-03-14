// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configauth implements the configuration settings to
// ensure authentication on incoming requests, and allows
// exporters to add authentication on outgoing requests.
package configmiddleware // import "go.opentelemetry.io/collector/config/configmiddleware"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	errExactlyOneID       = fmt.Errorf("middleware must configure a single one component")
	errNotALimiter        = fmt.Errorf("middleware must configure a single one component")
	errMiddlewareNotFound = fmt.Errorf("middleware not found")

	emptyID = component.ID{}
)

// Middleware defines the a middleware component.  Middleware can be more than
// one type of extension.  Each extension uses a single field, and exactly one
// field should be set.
//
// The typical way Middleware configuration is used will be in a list contained
// within another configuration object.  For example, the `configgrpc.ServerConfig`
// inside the OTLP receiver lists middleware:
//
//	otlp:
//	  protocols:
//	    grpc:
//	      middleware:
//	      - interceptor: blocker
//	      - limiter: admission
//	      - interceptor: decorator
type Middleware struct {
	// LimiterID specifies the name of a limiter extension to be used.
	LimiterID component.ID `mapstructure:"limiter,omitempty"`

	// InterceptorID specifies the name of an interceptor extension to be used.
	InterceptorID component.ID `mapstructure:"interceptor,omitempty"`
}

// Validate ensures that exactly one IDis set.
func (m Middleware) Validate() error {
	hasLim := m.LimiterID.String() != ""
	hasInt := m.InterceptorID.String() != ""
	if hasLim == hasInt {
		return errExactlyOneID
	}
	return nil
}

// GetLimiter requests to locate a named limiter extension.
func (m Middleware) GetLimiter(_ context.Context, extensions map[component.ID]component.Component) (extensionlimiter.Limiter, error) {
	if m.LimiterID == emptyID {
		return nil, errNotALimiter
	}
	if ext, found := extensions[m.LimiterID]; found {
		if ext, ok := ext.(extensionlimiter.Limiter); ok {
			return ext, nil
		}
		return nil, errNotALimiter
	}
	return nil, fmt.Errorf("failed to resolve limiter %q: %w", m.LimiterID, errMiddlewareNotFound)
}
