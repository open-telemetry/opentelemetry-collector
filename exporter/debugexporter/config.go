// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

var (
	// supportedLevels in this exporter's configuration.
	// configtelemetry.LevelNone and other future values are not supported.
	supportedLevels map[configtelemetry.Level]struct{} = map[configtelemetry.Level]struct{}{
		configtelemetry.LevelBasic:    {},
		configtelemetry.LevelNormal:   {},
		configtelemetry.LevelDetailed: {},
	}
)

// Config defines configuration for logging exporter.
type Config struct {
	// Verbosity defines the logging exporter verbosity.
	Verbosity configtelemetry.Level `mapstructure:"verbosity,omitempty"`

	// SamplingInitial defines how many samples are initially logged during each second.
	SamplingInitial int `mapstructure:"sampling_initial"`

	// SamplingThereafter defines the sampling rate after the initial samples are logged.
	SamplingThereafter int `mapstructure:"sampling_thereafter"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if _, ok := supportedLevels[cfg.Verbosity]; !ok {
		return fmt.Errorf("verbosity level %q is not supported", cfg.Verbosity)
	}

	return nil
}
