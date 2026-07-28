// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	cfg := &Config{}
	assert.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t,
		&Config{
			CheckInterval:       5 * time.Second,
			MemoryLimitMiB:      4000,
			MemorySpikeLimitMiB: 500,
		}, cfg)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		err  error
	}{
		{
			name: "valid",
			cfg: &Config{
				MemoryLimitMiB:      5722,
				MemorySpikeLimitMiB: 1907,
				CheckInterval:       100 * time.Millisecond,
			},
			err: nil,
		},
		{
			name: "zero check interval",
			cfg: &Config{
				CheckInterval: 0,
			},
			err: errCheckIntervalOutOfRange,
		},
		{
			name: "unset memory limit",
			cfg: &Config{
				CheckInterval:         1 * time.Second,
				MemoryLimitMiB:        0,
				MemoryLimitPercentage: 0,
			},
			err: errLimitOutOfRange,
		},
		{
			name: "invalid memory spike limit",
			cfg: &Config{
				CheckInterval:       1 * time.Second,
				MemoryLimitMiB:      10,
				MemorySpikeLimitMiB: 10,
			},
			err: errSpikeLimitOutOfRange,
		},
		{
			name: "invalid memory percentage limit",
			cfg: &Config{
				CheckInterval:         1 * time.Second,
				MemoryLimitPercentage: 101,
			},
			err: errLimitPercentageOutOfRange,
		},
		{
			name: "invalid memory spike percentage limit",
			cfg: &Config{
				CheckInterval:         1 * time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 60,
			},
			err: errSpikeLimitPercentageOutOfRange,
		},
		{
			name: "invalid gc intervals",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MinGCIntervalWhenSoftLimited: 50 * time.Millisecond,
				MinGCIntervalWhenHardLimited: 100 * time.Millisecond,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: errInconsistentGCMinInterval,
		},
		{
			name: "negative max soft interval",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MaxGCIntervalWhenSoftLimited: -1 * time.Second,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: errNegativeGCMaxSoftInterval,
		},
		{
			name: "negative max hard interval",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MaxGCIntervalWhenHardLimited: -1 * time.Second,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: errNegativeGCMaxHardInterval,
		},
		{
			name: "max soft less than min soft",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MinGCIntervalWhenSoftLimited: 10 * time.Second,
				MaxGCIntervalWhenSoftLimited: 5 * time.Second,
				MaxGCIntervalWhenHardLimited: 5 * time.Second,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: errInconsistentGCMaxSoftInterval,
		},
		{
			name: "max hard less than min hard",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MinGCIntervalWhenHardLimited: 20 * time.Second,
				MinGCIntervalWhenSoftLimited: 20 * time.Second,
				MaxGCIntervalWhenHardLimited: 5 * time.Second,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: errInconsistentGCMaxHardInterval,
		},
		{
			name: "max soft less than max hard",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MaxGCIntervalWhenSoftLimited: 5 * time.Second,
				MaxGCIntervalWhenHardLimited: 30 * time.Second,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: errInconsistentGCMaxInterval,
		},
		{
			name: "valid max gc intervals: both zero (disabled)",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MaxGCIntervalWhenSoftLimited: 0,
				MaxGCIntervalWhenHardLimited: 0,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: nil,
		},
		{
			name: "valid max gc intervals: only hard disabled",
			cfg: &Config{
				CheckInterval:                100 * time.Millisecond,
				MaxGCIntervalWhenSoftLimited: 30 * time.Second,
				MaxGCIntervalWhenHardLimited: 0,
				MemoryLimitMiB:               5722,
				MemorySpikeLimitMiB:          1907,
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestUnmarshalInvalidConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "negative_unsigned_limits_config.yaml"))
	require.NoError(t, err)
	cfg := &Config{}
	err = cm.Unmarshal(&cfg)
	require.ErrorContains(t, err, "'limit_mib' cannot parse value as 'uint32': -2000 overflows uint")
	require.ErrorContains(t, err, "'spike_limit_mib' cannot parse value as 'uint32': -2300 overflows uint")
}
