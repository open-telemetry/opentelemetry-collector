// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limitermiddlewareextension // import "go.opentelemetry.io/collector/extension/limitermiddlewareextension"

import (
	"errors"

	"go.opentelemetry.io/collector/config/configlimiter"
)

var errNoLimiterSpecified = errors.New("no limiter extension specified")

// Config defines configuration for the limiter middleware extension.
type Config struct {
	// Limiter configures the underlying extension used for limiting.
	Limiter configlimiter.Limiter `mapstructure:",squash"`
}

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	var noID configlimiter.Limiter
	if cfg.Limiter == noID {
		return errNoLimiterSpecified
	}
	return nil
}
