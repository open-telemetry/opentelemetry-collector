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

// GetLimiter attempts to select the appropriate extensionlimiter.Limiter from the list of extensions,
// based on the requested extension name. If a limiter is not found, an error is returned.
func (l Limiter) GetRateLimiter(_ context.Context, extensions map[component.ID]component.Component) (extensionlimiter.RateLimiter, error) {
	if ext, found := extensions[l.LimiterID]; found {
		if limiter, ok := ext.(extensionlimiter.RateLimiter); ok {
			return limiter, nil
		}
		return nil, errNotLimiter
	}

	return nil, fmt.Errorf("failed to resolve limiter %q: %w", l.LimiterID, errLimiterNotFound)
}

// GetResourceLimiter attempts to select the appropriate extensionlimiter.ResourceLimiter from the list of extensions,
// based on the requested extension name. If a limiter is not found, an error is returned.
func (l Limiter) GetResourceLimiter(_ context.Context, extensions map[component.ID]component.Component) (extensionlimiter.ResourceLimiter, error) {
	if ext, found := extensions[l.LimiterID]; found {
		if limiter, ok := ext.(extensionlimiter.ResourceLimiter); ok {
			return limiter, nil
		}
		return nil, errNotLimiter
	}

	return nil, fmt.Errorf("failed to resolve resource limiter %q: %w", l.LimiterID, errLimiterNotFound)
}
