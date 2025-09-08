// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal"
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

	// OutputConfig defines how the exporter would output attributes
	Output internal.OutputConfig `mapstructure:"output"`

	// prevent unkeyed literal initialization
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if _, ok := supportedLevels[cfg.Verbosity]; !ok {
		return fmt.Errorf("verbosity level %q is not supported", cfg.Verbosity)
	}

	if len(cfg.Output.Record.Exclude) > 0 && len(cfg.Output.Record.Include) > 0 {
		return errors.New("cannot specify both include and exclude attributes for record")
	}

	if len(cfg.Output.Scope.Exclude) > 0 && len(cfg.Output.Scope.Include) > 0 {
		return errors.New("cannot specify both include and exclude attributes for scope")
	}

	if len(cfg.Output.Resource.AttributesOutputConfig.Exclude) > 0 && len(cfg.Output.Resource.AttributesOutputConfig.Include) > 0 {
		return errors.New("cannot specify both include and exclude attributes for resource")
	}

	return nil
}
