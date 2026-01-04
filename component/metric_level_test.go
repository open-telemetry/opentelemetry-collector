// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterMetricLevelConfigs(t *testing.T) {
	t.Cleanup(ResetMetricLevelRegistryForTesting)

	RegisterMetricLevelConfigs(
		MetricLevelConfig{
			MeterName: "test/meter",
			Level:     MetricLevel(2),
		},
		MetricLevelConfig{
			MeterName:      "test/meter",
			InstrumentName: "metric",
			Level:          MetricLevel(1),
		},
	)

	configs := RegisteredMetricLevelConfigs()
	require.Len(t, configs, 2)
	assert.Equal(t, "test/meter", configs[0].MeterName)
	assert.Equal(t, MetricLevel(2), configs[0].Level)
	assert.Equal(t, "metric", configs[1].InstrumentName)
	assert.Equal(t, MetricLevel(1), configs[1].Level)
}

func TestRegisterMetricLevelConfigsPanicsWithoutMeter(t *testing.T) {
	t.Cleanup(ResetMetricLevelRegistryForTesting)

	assert.Panics(t, func() {
		RegisterMetricLevelConfigs(MetricLevelConfig{Level: MetricLevel(2)})
	})
}

func TestRegisteredMetricLevelConfigsByMeter(t *testing.T) {
	t.Cleanup(ResetMetricLevelRegistryForTesting)

	RegisterMetricLevelConfigs(
		MetricLevelConfig{
			MeterName: "test/meter1",
			Level:     MetricLevel(1),
		},
		MetricLevelConfig{
			MeterName:      "test/meter1",
			InstrumentName: "instrument1",
			Level:          MetricLevel(2),
		},
		MetricLevelConfig{
			MeterName: "test/meter2",
			Level:     MetricLevel(1),
		},
	)

	tests := []struct {
		name       string
		meterName  string
		wantCount  int
		wantLevels []MetricLevel
	}{
		{
			name:       "existing meter with multiple configs",
			meterName:  "test/meter1",
			wantCount:  2,
			wantLevels: []MetricLevel{1, 2},
		},
		{
			name:       "existing meter with single config",
			meterName:  "test/meter2",
			wantCount:  1,
			wantLevels: []MetricLevel{1},
		},
		{
			name:      "non-existent meter",
			meterName: "test/nonexistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs := RegisteredMetricLevelConfigsByMeter(tt.meterName)
			require.Len(t, configs, tt.wantCount)
			for i, cfg := range configs {
				assert.Equal(t, tt.meterName, cfg.MeterName)
				if len(tt.wantLevels) > i {
					assert.Equal(t, tt.wantLevels[i], cfg.Level)
				}
			}
		})
	}
}

func TestRegisterMetricLevelConfigsEmptySlice(t *testing.T) {
	t.Cleanup(ResetMetricLevelRegistryForTesting)

	initialCount := len(RegisteredMetricLevelConfigs())
	RegisterMetricLevelConfigs()
	assert.Len(t, RegisteredMetricLevelConfigs(), initialCount)
}

func TestRegisterMetricLevelConfigsConcurrent(t *testing.T) {
	t.Cleanup(ResetMetricLevelRegistryForTesting)

	const numGoroutines = 10
	const configsPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			configs := make([]MetricLevelConfig, configsPerGoroutine)
			for j := range configsPerGoroutine {
				configs[j] = MetricLevelConfig{
					MeterName:      "test/concurrent",
					InstrumentName: "instrument",
					Level:          MetricLevel(id*configsPerGoroutine + j),
				}
			}
			RegisterMetricLevelConfigs(configs...)
		}(i)
	}

	wg.Wait()

	allConfigs := RegisteredMetricLevelConfigs()
	require.GreaterOrEqual(t, len(allConfigs), numGoroutines*configsPerGoroutine)

	var readWg sync.WaitGroup
	readWg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer readWg.Done()
			configs := RegisteredMetricLevelConfigs()
			assert.GreaterOrEqual(t, len(configs), numGoroutines*configsPerGoroutine)
		}()
	}
	readWg.Wait()
}

func TestRegisteredMetricLevelConfigsReturnsCopy(t *testing.T) {
	t.Cleanup(ResetMetricLevelRegistryForTesting)

	RegisterMetricLevelConfigs(
		MetricLevelConfig{
			MeterName: "test/copy",
			Level:     MetricLevel(1),
		},
	)

	configs1 := RegisteredMetricLevelConfigs()
	configs2 := RegisteredMetricLevelConfigs()

	configs1[0].MeterName = "modified"

	assert.Equal(t, "test/copy", configs2[0].MeterName)
	assert.Equal(t, "modified", configs1[0].MeterName)
}

func TestRegisteredMetricLevelConfigsByMeterReturnsCopy(t *testing.T) {
	t.Cleanup(ResetMetricLevelRegistryForTesting)

	RegisterMetricLevelConfigs(
		MetricLevelConfig{
			MeterName: "test/copy",
			Level:     MetricLevel(1),
		},
	)

	configs1 := RegisteredMetricLevelConfigsByMeter("test/copy")
	configs2 := RegisteredMetricLevelConfigsByMeter("test/copy")

	require.Len(t, configs1, 1)
	require.Len(t, configs2, 1)

	configs1[0].MeterName = "modified"

	assert.Equal(t, "test/copy", configs2[0].MeterName)
	assert.Equal(t, "modified", configs1[0].MeterName)
}
