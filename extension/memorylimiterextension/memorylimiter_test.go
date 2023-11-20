// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterextension

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/iruntime"
	"go.opentelemetry.io/collector/internal/memorylimiter"
)

func TestMemoryPressureResponse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		mlCfg          *Config
		getMemFn       func() (uint64, error)
		readMemStatsFn func(m *runtime.MemStats)
		expectError    bool
	}{
		{
			name: "fixed_limit",
			mlCfg: &Config{
				CheckInterval:       1 * time.Nanosecond,
				MemoryLimitMiB:      1024,
				MemorySpikeLimitMiB: 512,
			},
			expectError: false,
		},
		{
			name: "percent_limit",
			mlCfg: &Config{
				CheckInterval:         1 * time.Nanosecond,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 30,
			},
			expectError: false,
		},
		{
			name: "fixed_limit",
			mlCfg: &Config{
				CheckInterval:         1 * time.Nanosecond,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 30,
			},
			getMemFn:       totalMemory,
			readMemStatsFn: readMemStats,
			expectError:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.getMemFn != nil {
				memorylimiter.GetMemoryFn = tt.getMemFn
			}
			if tt.readMemStatsFn != nil {
				memorylimiter.ReadMemStatsFn = tt.readMemStatsFn
			}
			ml, err := newMemoryLimiter(tt.mlCfg, zap.NewNop())

			assert.NoError(t, err)
			assert.NoError(t, ml.Start(ctx, &mockHost{}))
			time.Sleep(50 * time.Millisecond)
			mustRefuse := ml.MustRefuse()
			if tt.expectError {
				assert.True(t, mustRefuse)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, ml.Shutdown(ctx))
		})
	}
	t.Cleanup(func() {
		memorylimiter.GetMemoryFn = iruntime.TotalMemory
		memorylimiter.ReadMemStatsFn = runtime.ReadMemStats
	})
}

type mockHost struct {
	component.Host
}

func (h *mockHost) GetExtensions() map[component.ID]component.Component {
	return make(map[component.ID]component.Component)
}

func totalMemory() (uint64, error) {
	return uint64(4096), nil
}

func readMemStats(m *runtime.MemStats) {
	m.Alloc = 2000
}
