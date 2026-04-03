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
	errInconsistentGCMinInterval      = errors.New("'min_gc_interval_when_soft_limited' should be larger than 'min_gc_interval_when_hard_limited'")
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

	// MinGCIntervalWhenSoftLimited minimum interval between forced GC when in soft (=limit_mib - spike_limit_mib) limited mode.
	// Zero value means no minimum interval.
	// GCs is a CPU-heavy operation and executing it too frequently may affect the recovery capabilities of the collector.
	MinGCIntervalWhenSoftLimited time.Duration `mapstructure:"min_gc_interval_when_soft_limited"`

	// MinGCIntervalWhenHardLimited minimum interval between forced GC when in hard (=limit_mib) limited mode.
	// Zero value means no minimum interval.
	// GCs is a CPU-heavy operation and executing it too frequently may affect the recovery capabilities of the collector.
	MinGCIntervalWhenHardLimited time.Duration `mapstructure:"min_gc_interval_when_hard_limited"`

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

func NewDefaultConfig() *Config {
	return &Config{
		MinGCIntervalWhenSoftLimited: 10 * time.Second,
	}
}

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.CheckInterval <= 0 {
		return errCheckIntervalOutOfRange
	}
	if cfg.MinGCIntervalWhenSoftLimited < cfg.MinGCIntervalWhenHardLimited {
		return errInconsistentGCMinInterval
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
