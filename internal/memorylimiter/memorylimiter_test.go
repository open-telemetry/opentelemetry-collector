// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
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

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse())

	// Above memAllocLimit even accounting for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())

	// Restore ballast to default.
	ml.ballastSize = 0

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
	decison1000Limit30Spike30 := newPercentageMemUsageChecker(1000, 60, 30)
	decison1000Limit60Spike50 := newPercentageMemUsageChecker(1000, 60, 50)
	decison1000Limit40Spike20 := newPercentageMemUsageChecker(1000, 40, 20)

	tests := []struct {
		name         string
		usageChecker memUsageChecker
		ms           *runtime.MemStats
		shouldRefuse bool
	}{
		{
			name:         "should refuse over limit",
			usageChecker: *decison1000Limit30Spike30,
			ms:           &runtime.MemStats{Alloc: 600},
			shouldRefuse: true,
		},
		{
			name:         "should not refuse",
			usageChecker: *decison1000Limit30Spike30,
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
			usageChecker: *decison1000Limit60Spike50,
			ms:           &runtime.MemStats{Alloc: 300},
			shouldRefuse: true,
		},
		{
			name:         "should refuse, spike, percentage usageChecker",
			usageChecker: *decison1000Limit40Spike20,
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

func TestBallastSize(t *testing.T) {
	cfg := &Config{
		CheckInterval:  10 * time.Second,
		MemoryLimitMiB: 1024,
	}
	got, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	got.startMonitoring()
	require.NoError(t, got.Start(context.Background(), &host{ballastSize: 113}))
	assert.Equal(t, uint64(113), got.ballastSize)
	require.NoError(t, got.Shutdown(context.Background()))
}

type host struct {
	ballastSize uint64
	component.Host
}

func (h *host) GetExtensions() map[component.ID]component.Component {
	ret := make(map[component.ID]component.Component)
	ret[component.MustNewID("ballast")] = &ballastExtension{ballastSize: h.ballastSize}
	return ret
}

type ballastExtension struct {
	ballastSize uint64
	component.StartFunc
	component.ShutdownFunc
}

func (be *ballastExtension) GetBallastSize() uint64 {
	return be.ballastSize
}
