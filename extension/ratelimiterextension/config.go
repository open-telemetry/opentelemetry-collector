// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ratelimiterextension // import "go.opentelemetry.io/collector/extension/ratelimiterextension"

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// LimitConfig defines parameters for a token bucket rate limiter
type LimitConfig struct {
	// Key specifies the weight key to limit (e.g. "network_bytes", "request_count", etc)
	Key extensionlimiter.WeightKey `mapstructure:"key"`
	// Rate is the rate at which tokens are replenished (e.g. X tokens per second)
	Rate float64 `mapstructure:"rate"`
	// BurstSize is the maximum number of tokens that can be consumed in a single burst
	BurstSize int64 `mapstructure:"burst_size"`
}

// Config defines the configuration for the token bucket rate limiter extension
type Config struct {
	// Limits is a list of token bucket configurations, one per weight key
	Limits []LimitConfig `mapstructure:"limits"`
	// FillingInterval is the interval at which tokens are added to the bucket
	FillingInterval time.Duration `mapstructure:"filling_interval"`
}

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if len(cfg.Limits) == 0 {
		return errors.New("at least one limit must be configured")
	}
	if cfg.FillingInterval <= 0 {
		return fmt.Errorf("filling_interval must be positive, got %v", cfg.FillingInterval)
	}

	seenKeys := make(map[extensionlimiter.WeightKey]bool)
	for i, limit := range cfg.Limits {
		if limit.Key == "" {
			return fmt.Errorf("limit[%d].key cannot be empty", i)
		}
		if seenKeys[limit.Key] {
			return fmt.Errorf("duplicate limit key: %s", limit.Key)
		}
		seenKeys[limit.Key] = true

		if limit.Rate <= 0 {
			return fmt.Errorf("limit[%d].rate must be positive, got %v", i, limit.Rate)
		}
		if limit.BurstSize <= 0 {
			return fmt.Errorf("limit[%d].burst_size must be positive, got %v", i, limit.BurstSize)
		}
	}
	return nil
}
