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
		name        string
		mlCfg       *Config
		memAlloc    uint64
		expectError bool
	}{
		{
			name: "Below memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memAllocLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 1,
			},
			memAlloc:    1800,
			expectError: true,
		},
		{
			name: "Below memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 10,
			},
			memAlloc:    800,
			expectError: false,
		},
		{
			name: "Above memSpikeLimit",
			mlCfg: &Config{
				CheckInterval:         time.Second,
				MemoryLimitPercentage: 50,
				MemorySpikePercentage: 11,
			},
			memAlloc:    800,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memorylimiter.GetMemoryFn = totalMemory
			memorylimiter.ReadMemStatsFn = func(ms *runtime.MemStats) {
				ms.Alloc = tt.memAlloc
			}
			ml, err := newMemoryLimiter(tt.mlCfg, zap.NewNop())
			assert.NoError(t, err)

			assert.NoError(t, ml.Start(ctx, &mockHost{}))
			ml.memLimiter.CheckMemLimits()
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
	return uint64(2048), nil
}
