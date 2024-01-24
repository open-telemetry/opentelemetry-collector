// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter // import "go.opentelemetry.io/collector/internal/memorylimiter"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

var (
	errCheckIntervalOutOfRange        = errors.New("'check_interval' must be greater than zero")
	errLimitOutOfRange                = errors.New("'limit_mib' or 'limit_percentage' must be greater than zero")
	errSpikeLimitOutOfRange           = errors.New("'spike_limit_mib' must be smaller than 'limit_mib'")
	errSpikeLimitPercentageOutOfRange = errors.New("'spike_limit_percentage' must be smaller than 'limit_percentage'")
	errLimitPercentageOutOfRange      = errors.New(
		"'limit_percentage' and 'spike_limit_percentage' must be greater than zero and less than or equal to hundred")
)

// Config defines configuration for memory memoryLimiter processor.
type Config struct {
	// CheckInterval is the time between measurements of memory usage for the
	// purposes of avoiding going over the limits. Defaults to zero, so no
	// checks will be performed.
	CheckInterval time.Duration `mapstructure:"check_interval"`

	// MemoryLimitMiB is the maximum amount of memory, in MiB, targeted to be
	// allocated by the process.
	MemoryLimitMiB uint32 `mapstructure:"limit_mib"`

	// MemorySpikeLimitMiB is the maximum, in MiB, spike expected between the
	// measurements of memory usage.
	MemorySpikeLimitMiB uint32 `mapstructure:"spike_limit_mib"`

	// MemoryLimitPercentage is the maximum amount of memory, in %, targeted to be
	// allocated by the process. The fixed memory settings MemoryLimitMiB has a higher precedence.
	MemoryLimitPercentage uint32 `mapstructure:"limit_percentage"`

	// MemorySpikePercentage is the maximum, in percents against the total memory,
	// spike expected between the measurements of memory usage.
	MemorySpikePercentage uint32 `mapstructure:"spike_limit_percentage"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.CheckInterval <= 0 {
		return errCheckIntervalOutOfRange
	}
	if cfg.MemoryLimitMiB == 0 && cfg.MemoryLimitPercentage == 0 {
		return errLimitOutOfRange
	}
	if cfg.MemoryLimitPercentage > 100 || cfg.MemorySpikePercentage > 100 {
		return errLimitPercentageOutOfRange
	}
	if cfg.MemoryLimitMiB > 0 && cfg.MemoryLimitMiB <= cfg.MemorySpikeLimitMiB {
		return errSpikeLimitOutOfRange
	}
	if cfg.MemoryLimitPercentage > 0 && cfg.MemoryLimitPercentage <= cfg.MemorySpikePercentage {
		return errSpikeLimitPercentageOutOfRange
	}
	return nil
}
