// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configauth implements the configuration settings to
// ensure authentication on incoming requests, and allows
// exporters to add authentication on outgoing requests.
package configauth // import "go.opentelemetry.io/collector/config/configauth"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/limiter"
)

var (
	errLimiterNotFound = errors.New("limiter not found")
	errNotLimiter      = errors.New("requested component is not a limiter")
)

// Limitation defines the limit settings for a component.
type Limitation struct {
	// LimiterID specifies the name of the extension to use in
	// order to limit incoming requests.
	LimiterID component.ID `mapstructure:"limiter,omitempty"`
}

// GetLimiter attempts to select the appropriate extensionauth.Server
// from the list of extensions, based on the requested extension
// name. If an authenticator is not found, an error is returned.
func (l Limitation) GetLimiter(ctx context.Context, extensions map[component.ID]component.Component, kind component.Kind, id component.ID) (limiter.Limiter, error) {
	if ext, found := extensions[l.LimiterID]; found {
		if ext, ok := ext.(limiter.Extension); ok {
			return ext.GetLimiter(ctx, kind, id)
		}
		return nil, errNotLimiter
	}

	return nil, fmt.Errorf("failed to resolve limiter %q: %w", l.LimiterID, errLimiterNotFound)
}
