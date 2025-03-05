// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter // import "go.opentelemetry.io/collector/internal/memorylimiter"

import (
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
)

var (
	errCheckIntervalOutOfRange        = errors.New("'check_interval' must be greater than zero")
	errInconsistentGCMinInterval      = errors.New("'min_gc_interval_when_soft_limited' should be larger than 'min_gc_interval_when_hard_limited'")
	errLimitOutOfRange                = errors.New("'limit_mib' or 'limit_percentage' must be greater than zero")
	errAdmissionLimitOutOfRange       = errors.New("'request_limit_mib' must be greater than zero")
	errSpikeLimitOutOfRange           = errors.New("'spike_limit_mib' must be smaller than 'limit_mib'")
	errSpikeLimitPercentageOutOfRange = errors.New("'spike_limit_percentage' must be smaller than 'limit_percentage'")
	errLimitPercentageOutOfRange      = errors.New(
		"'limit_percentage' and 'spike_limit_percentage' must be greater than zero and less than or equal to hundred")
)

// Config defines configuration for memory memoryLimiter processor.
type Config struct {
	// Model is one of "gcstats" or "admission". Use the
	// corresponding `gcstats` or `admission` sub-configuration
	// objects.
	Model string `mapstructure:"model"`

	// GCStats contains settings that control a memory limiter
	// based on garbage collector stats.  This is the original
	// model supported by this component, therefore the
	// configuration is squashed.
	//
	// Note: can we un-squash this using a feature flag?
	//
	// Note: this is a breaking API change. Should we embed the
	// GCStatsConfig object so that callers can continue to access
	// GCStats config w/o adding `GCStats.` at every field reference?
	GCStats GCStatsConfig `mapstructure:",squash"`

	// Admission contains settings that control a memory limiter
	// based on an exact count of bytes pending and in the
	// pipeline.
	Admission AdmissionConfig `mapstructure:"admission"`
}

// GCStatsConfig is the basis of a garbage-collector statistics-based memory limiter.
type GCStatsConfig struct {
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

// AdmissionConfig is the basis of a memory limiter that counts the
// number of bytes pending and in the pipeline.
type AdmissionConfig struct {
	// RequestLimitMiB limits the number of requests that are received by the stream based on
	// uncompressed request size. Request size is used to control how much traffic we admit
	// for processing.  When this field is zero, admission control is disabled meaning all
	// requests will be immediately accepted.
	RequestLimitMiB uint64 `mapstructure:"request_limit_mib"`

	// WaitingLimitMiB is the limit on the amount of data waiting to be consumed.
	// This is a dimension of memory limiting to ensure waiters are not consuming an
	// unexpectedly large amount of memory in receivers that use it.
	WaitingLimitMiB uint64 `mapstructure:"waiting_limit_mib"`
}

var _ component.Config = (*Config)(nil)

func NewDefaultConfig() *Config {
	// Note that users are required to configure the primary limit in
	// all models.  For GCStats, the critical config is MemoryLimitMiB,
	// for Admission, the critical config is RequestLimitMiB.
	return &Config{
		Model: "gcstats",
		GCStats: GCStatsConfig{
			MemoryLimitMiB:               0,
			MinGCIntervalWhenSoftLimited: 10 * time.Second,
		},
		Admission: AdmissionConfig{
			RequestLimitMiB: 0,
			WaitingLimitMiB: 0,
		},
	}
}

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	switch strings.ToLower(cfg.Model) {
	// The default branch includes the case where Model is unset,
	// for backwards compatibility.
	case "", "gcstats":
		if cfg.GCStats.CheckInterval <= 0 {
			return errCheckIntervalOutOfRange
		}
		if cfg.GCStats.MinGCIntervalWhenSoftLimited < cfg.GCStats.MinGCIntervalWhenHardLimited {
			return errInconsistentGCMinInterval
		}
		if cfg.GCStats.MemoryLimitMiB == 0 && cfg.GCStats.MemoryLimitPercentage == 0 {
			return errLimitOutOfRange
		}
		if cfg.GCStats.MemoryLimitPercentage > 100 || cfg.GCStats.MemorySpikePercentage > 100 {
			return errLimitPercentageOutOfRange
		}
		if cfg.GCStats.MemoryLimitMiB > 0 && cfg.GCStats.MemoryLimitMiB <= cfg.GCStats.MemorySpikeLimitMiB {
			return errSpikeLimitOutOfRange
		}
		if cfg.GCStats.MemoryLimitPercentage > 0 && cfg.GCStats.MemoryLimitPercentage <= cfg.GCStats.MemorySpikePercentage {
			return errSpikeLimitPercentageOutOfRange
		}
	case "admission":
		if cfg.Admission.RequestLimitMiB == 0 {
			return errAdmissionLimitOutOfRange
		}
	}
	return nil
}
