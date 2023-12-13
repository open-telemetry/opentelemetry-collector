// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterprocessor

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(confmap.New(), cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
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
			cfg: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.MemoryLimitMiB = 5722
				cfg.MemorySpikeLimitMiB = 1907
				cfg.CheckInterval = 100 * time.Millisecond
				return cfg
			}(),
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
