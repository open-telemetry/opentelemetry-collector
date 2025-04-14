// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configlimiter implements the configuration settings to
// apply rate limiting on incoming requests, and allows
// components to configure rate limiting behavior.
package configlimiter // import "go.opentelemetry.io/collector/config/configlimiter"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	errLimiterNotFound = errors.New("limiter not found")
	errNotLimiter      = errors.New("requested extension is not a limiter")
)

// Limiter defines the rate limiting settings for a component.
type Limiter struct {
	// LimiterID specifies the name of the extension to use in order to apply rate limiting.
	LimiterID component.ID `mapstructure:"limiter,omitempty"`
}

// GetProvider attempts to select the appropriate extensionlimiter.Provider from the list of extensions,
// based on the requested extension name. If a limiter is not found, an error is returned.
// Callers will use the returned Provider to get access to the specific rate- and
// resource-limiter weights they are capable of limiting.
func (l Limiter) GetProvider(_ context.Context, extensions map[component.ID]component.Component) (extensionlimiter.Provider, error) {
	if ext, found := extensions[l.LimiterID]; found {
		if limiter, ok := ext.(extensionlimiter.Provider); ok {
			return limiter, nil
		}
		return nil, errNotLimiter
	}

	return nil, fmt.Errorf("failed to resolve limiter provider %q: %w", l.LimiterID, errLimiterNotFound)
}
