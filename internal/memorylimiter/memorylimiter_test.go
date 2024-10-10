// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter

import (
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/internal/iruntime"
)

// TestMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &MemoryLimiter{
		usageChecker: memUsageChecker{
			memAllocLimit: 1024,
		},
		mustRefuse: &atomic.Bool{},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
		logger: zap.NewNop(),
	}

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse())

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())

	// Check spike limit
	ml.usageChecker.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse())

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())
}

func TestGetDecision(t *testing.T) {
	t.Run("fixed_limit", func(t *testing.T) {
		d, err := getMemUsageChecker(&Config{MemoryLimitMiB: 100, MemorySpikeLimitMiB: 20}, zap.NewNop())
		require.NoError(t, err)
		assert.Equal(t, &memUsageChecker{
			memAllocLimit: 100 * mibBytes,
			memSpikeLimit: 20 * mibBytes,
		}, d)
	})

	t.Cleanup(func() {
		GetMemoryFn = iruntime.TotalMemory
	})
	GetMemoryFn = func() (uint64, error) {
		return 100 * mibBytes, nil
	}
	t.Run("percentage_limit", func(t *testing.T) {
		d, err := getMemUsageChecker(&Config{MemoryLimitPercentage: 50, MemorySpikePercentage: 10}, zap.NewNop())
		require.NoError(t, err)
		assert.Equal(t, &memUsageChecker{
			memAllocLimit: 50 * mibBytes,
			memSpikeLimit: 10 * mibBytes,
		}, d)
	})
}

func TestRefuseDecision(t *testing.T) {
	decision1000Limit30Spike30 := newPercentageMemUsageChecker(1000, 60, 30)
	decision1000Limit60Spike50 := newPercentageMemUsageChecker(1000, 60, 50)
	decision1000Limit40Spike20 := newPercentageMemUsageChecker(1000, 40, 20)

	tests := []struct {
		name         string
		usageChecker memUsageChecker
		ms           *runtime.MemStats
		shouldRefuse bool
	}{
		{
			name:         "should refuse over limit",
			usageChecker: *decision1000Limit30Spike30,
			ms:           &runtime.MemStats{Alloc: 600},
			shouldRefuse: true,
		},
		{
			name:         "should not refuse",
			usageChecker: *decision1000Limit30Spike30,
			ms:           &runtime.MemStats{Alloc: 100},
			shouldRefuse: false,
		},
		{
			name: "should not refuse spike, fixed usageChecker",
			usageChecker: memUsageChecker{
				memAllocLimit: 600,
				memSpikeLimit: 500,
			},
			ms:           &runtime.MemStats{Alloc: 300},
			shouldRefuse: true,
		},
		{
			name:         "should refuse, spike, percentage usageChecker",
			usageChecker: *decision1000Limit60Spike50,
			ms:           &runtime.MemStats{Alloc: 300},
			shouldRefuse: true,
		},
		{
			name:         "should refuse, spike, percentage usageChecker",
			usageChecker: *decision1000Limit40Spike20,
			ms:           &runtime.MemStats{Alloc: 250},
			shouldRefuse: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shouldRefuse := test.usageChecker.aboveSoftLimit(test.ms)
			assert.Equal(t, test.shouldRefuse, shouldRefuse)
		})
	}
}
