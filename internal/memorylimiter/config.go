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
	errInconsistentGCMaxInterval      = errors.New("'max_gc_interval_when_soft_limited' should be larger than or equal to 'max_gc_interval_when_hard_limited' (when both are set)")
	errInconsistentGCMaxSoftInterval  = errors.New("'max_gc_interval_when_soft_limited' must be greater than or equal to 'min_gc_interval_when_soft_limited' (or 0 to disable)")
	errInconsistentGCMaxHardInterval  = errors.New("'max_gc_interval_when_hard_limited' must be greater than or equal to 'min_gc_interval_when_hard_limited' (or 0 to disable)")
	errNegativeGCMaxSoftInterval      = errors.New("'max_gc_interval_when_soft_limited' must be non-negative")
	errNegativeGCMaxHardInterval      = errors.New("'max_gc_interval_when_hard_limited' must be non-negative")
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

	// MaxGCIntervalWhenSoftLimited caps the exponential backoff between forced
	// GC calls in soft-limited mode. When a forced GC fails to reclaim memory
	// (e.g., held by live references in exporter queues during a downstream
	// outage), the interval doubles starting from min_gc_interval_when_soft_limited
	// (or 95% of check_interval, whichever is larger) up to this cap. It resets
	// to zero whenever a GC is effective or memory drops on its own.
	// Set to 0 to disable the exponential backoff on this path; the interval
	// will not grow beyond the configured min.
	MaxGCIntervalWhenSoftLimited time.Duration `mapstructure:"max_gc_interval_when_soft_limited"`

	// MaxGCIntervalWhenHardLimited caps the exponential backoff between forced
	// GC calls in hard-limited mode. Same semantics as
	// MaxGCIntervalWhenSoftLimited but applies to the hard-limit path.
	// Set to 0 to disable the exponential backoff on this path.
	MaxGCIntervalWhenHardLimited time.Duration `mapstructure:"max_gc_interval_when_hard_limited"`

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
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
	if cfg.MaxGCIntervalWhenSoftLimited < 0 {
		return errNegativeGCMaxSoftInterval
	}
	if cfg.MaxGCIntervalWhenHardLimited < 0 {
		return errNegativeGCMaxHardInterval
	}
	if cfg.MaxGCIntervalWhenSoftLimited > 0 && cfg.MaxGCIntervalWhenSoftLimited < cfg.MinGCIntervalWhenSoftLimited {
		return errInconsistentGCMaxSoftInterval
	}
	if cfg.MaxGCIntervalWhenHardLimited > 0 && cfg.MaxGCIntervalWhenHardLimited < cfg.MinGCIntervalWhenHardLimited {
		return errInconsistentGCMaxHardInterval
	}
	// Mirror the MinSoft >= MinHard invariant for the max pair: when both
	// caps are enabled, the soft cap should not be tighter than the hard cap
	// (it would mean a smaller backoff ceiling for the less-urgent path,
	// which is almost always a typo).
	if cfg.MaxGCIntervalWhenSoftLimited > 0 && cfg.MaxGCIntervalWhenHardLimited > 0 &&
		cfg.MaxGCIntervalWhenSoftLimited < cfg.MaxGCIntervalWhenHardLimited {
		return errInconsistentGCMaxInterval
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
