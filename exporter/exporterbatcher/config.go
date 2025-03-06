// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher // import "go.opentelemetry.io/collector/exporter/exporterbatcher"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/confmap"
)

// Config defines a configuration for batching requests based on a timeout and a minimum number of items.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Config struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`

	// FlushTimeout sets the time after which a batch will be sent regardless of its size.
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`

	// SizeConfig sets the size limits for a batch.
	SizeConfig `mapstructure:",squash"`

	// Deprecated: [v0.121.0] Ignored if SizeConfig is set.
	MinSizeConfig `mapstructure:",squash"`
	// Deprecated: [v0.121.0] Ignored if SizeConfig is set.
	MaxSizeConfig `mapstructure:",squash"`
}

// SizeConfig sets the size limits for a batch.
type SizeConfig struct {
	Sizer SizerType `mapstructure:"sizer"`

	// MinSize defines the configuration for the minimum size of a batch.
	MinSize int `mapstructure:"min_size"`
	// MaxSize defines the configuration for the maximum size of a batch.
	MaxSize int `mapstructure:"max_size"`
}

// Deprecated: [v0.121.0] use SizeConfig.
type MinSizeConfig struct {
	// Deprecated: [v0.121.0] use SizeConfig.MinSize.
	MinSizeItems *int `mapstructure:"min_size_items"`
}

// Deprecated: [v0.121.0] use SizeConfig.
type MaxSizeConfig struct {
	// Deprecated: [v0.121.0] use SizeConfig.MaxSize.
	MaxSizeItems *int `mapstructure:"max_size_items"`
}

func (c *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(c); err != nil {
		return err
	}

	if c.MinSizeItems != nil && !conf.IsSet("min_size") {
		c.SizeConfig.MinSize = *c.MinSizeItems
	}

	if c.MaxSizeItems != nil && !conf.IsSet("max_size") {
		c.SizeConfig.MaxSize = *c.MaxSizeItems
	}

	return nil
}

func (c *Config) Validate() error {
	if c.FlushTimeout <= 0 {
		return errors.New("`flush_timeout` must be greater than zero")
	}

	return nil
}

func (c SizeConfig) Validate() error {
	if c.MinSize < 0 {
		return errors.New("`min_size` must be greater than or equal to zero")
	}
	if c.MaxSize < 0 {
		return errors.New("`max_size` must be greater than or equal to zero")
	}
	if c.MaxSize != 0 && c.MaxSize < c.MinSize {
		return errors.New("`max_size` must be greater than or equal to mix_size")
	}
	return nil
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   SizerTypeItems,
			MinSize: 8192,
		},
	}
}
