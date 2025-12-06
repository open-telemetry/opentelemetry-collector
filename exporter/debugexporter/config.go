// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"errors"
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

	// OutputPaths is a list of file paths to write logging output to.
	// This option is only used when use_internal_logger is false.
	// Special strings "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr respectively.
	// All other values are treated as file paths.
	// (default = ["stdout"])
	// OutputPaths will be ignored when UseInternalLogger is true
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

	// If use_internal_logger is false, output_paths must be specified and non-empty
	if !cfg.UseInternalLogger && len(cfg.OutputPaths) == 0 {
		return errors.New("output_paths must be specified and non-empty when use_internal_logger is false")
	}

	return nil
}
