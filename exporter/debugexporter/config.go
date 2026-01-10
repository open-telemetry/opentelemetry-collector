// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// supportedLevels in this exporter's configuration.
// configtelemetry.LevelNone and other future values are not supported.
var supportedLevels map[configtelemetry.Level]struct{} = map[configtelemetry.Level]struct{}{
	configtelemetry.LevelBasic:    {},
	configtelemetry.LevelNormal:   {},
	configtelemetry.LevelDetailed: {},
}

// Config defines configuration for debug exporter.
type Config struct {
	// Verbosity defines the debug exporter verbosity.
	Verbosity configtelemetry.Level `mapstructure:"verbosity,omitempty"`

	// SamplingInitial defines how many samples are initially logged during each second.
	SamplingInitial int `mapstructure:"sampling_initial"`

	// SamplingThereafter defines the sampling rate after the initial samples are logged.
	SamplingThereafter int `mapstructure:"sampling_thereafter"`

	// UseInternalLogger defines whether the exporter sends the output to the collector's internal logger.
	UseInternalLogger bool `mapstructure:"use_internal_logger"`

	// OutputPaths holds the list of destinations for the exporter's output when UseInternalLogger is false.
	OutputPaths []string `mapstructure:"output_paths"`

	QueueConfig configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`

	// prevent unkeyed literal initialization
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if _, ok := supportedLevels[cfg.Verbosity]; !ok {
		return fmt.Errorf("verbosity level %q is not supported", cfg.Verbosity)
	}

	if len(cfg.OutputPaths) > 0 {
		if cfg.UseInternalLogger {
			return fmt.Errorf("output_paths requires use_internal_logger to be false")
		}
		for _, path := range cfg.OutputPaths {
			if path == "" {
				return fmt.Errorf("output_paths cannot contain empty values")
			}
		}
	}

	return nil
}
