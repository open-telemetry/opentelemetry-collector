// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

// Config has the configuration for the extension enabling the zPages extension.
type Config struct {
	confighttp.ServerConfig `mapstructure:",squash"`

	Expvar ExpvarConfig `mapstructure:"expvar"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// ExpvarConfig has the configuration for the expvar service.
type ExpvarConfig struct {
	// Enabled indicates whether to enable expvar service.
	// (default = false)
	Enabled bool `mapstructure:"enabled"`
	// prevent unkeyed literal initialization
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Endpoint == "" {
		return errors.New("\"endpoint\" is required when using the \"zpages\" extension")
	}
	return nil
}
