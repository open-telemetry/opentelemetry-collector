// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	cfg := &Config{}
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			assert.Equal(t, tt.err, err)
		})
	}
}
