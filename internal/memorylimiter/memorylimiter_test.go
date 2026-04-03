// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/internal/memorylimiter/iruntime"
)

// TestMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	cfg := &Config{
		CheckInterval:       1 * time.Minute,
		MemoryLimitMiB:      1024,
		MemorySpikeLimitMiB: 0,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = currentMemAlloc * mibBytes
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
	ml.usageChecker.memSpikeLimit = 512 * mibBytes

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

func TestCallGCWhenSoftLimit(t *testing.T) {
	tests := []struct {
		name        string
		mlCfg       *Config
		memAllocMiB [2]uint64
		numGCs      int
	}{
		{
			name: "GC when first soft limit and not immediately",
			mlCfg: &Config{
				CheckInterval:                1 * time.Minute,
				MinGCIntervalWhenSoftLimited: 10 * time.Second,
				MemoryLimitMiB:               50,
				MemorySpikeLimitMiB:          10,
			},
			memAllocMiB: [2]uint64{45, 45},
			numGCs:      1,
		},
		{
			name: "GC always when soft limit min interval is 0",
			mlCfg: &Config{
				CheckInterval:                1 * time.Minute,
				MinGCIntervalWhenSoftLimited: 0,
				MemoryLimitMiB:               50,
				MemorySpikeLimitMiB:          10,
			},
			memAllocMiB: [2]uint64{45, 45},
			numGCs:      2,
		},
		{
			name: "GC when first hard limit and not immediately",
			mlCfg: &Config{
				CheckInterval:                1 * time.Minute,
				MinGCIntervalWhenHardLimited: 10 * time.Second,
				MemoryLimitMiB:               50,
				MemorySpikeLimitMiB:          10,
			},
			memAllocMiB: [2]uint64{55, 55},
			numGCs:      1,
		},
		{
			name: "GC always when hard limit min interval is 0",
			mlCfg: &Config{
				CheckInterval:                1 * time.Minute,
				MinGCIntervalWhenHardLimited: 0,
				MemoryLimitMiB:               50,
				MemorySpikeLimitMiB:          10,
			},
			memAllocMiB: [2]uint64{55, 55},
			numGCs:      2,
		},
		{
			name: "GC based on soft then based on hard limit",
			mlCfg: &Config{
				CheckInterval:                1 * time.Minute,
				MinGCIntervalWhenSoftLimited: 10 * time.Second,
				MinGCIntervalWhenHardLimited: 0,
				MemoryLimitMiB:               50,
				MemorySpikeLimitMiB:          10,
			},
			memAllocMiB: [2]uint64{45, 55},
			numGCs:      2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml, err := NewMemoryLimiter(tt.mlCfg, zap.NewNop())
			require.NoError(t, err)
			memAllocMiB := uint64(0)
			// Track whether GC was called so that readMemStatsFn can simulate
			// effective GC by reporting lower memory after collection. Without
			// this, the GC effectiveness backoff would suppress the second GC
			// call in tests that expect GC to fire on every check.
			gcCalled := false
			ml.readMemStatsFn = func(ms *runtime.MemStats) {
				if gcCalled {
					// Simulate GC reclaiming ~10% of memory. This is enough
					// to be considered effective (above the 5% threshold) but
					// still above the soft limit, so mustRefuse remains true.
					ms.Alloc = memAllocMiB * mibBytes * 9 / 10
					gcCalled = false
				} else {
					ms.Alloc = memAllocMiB * mibBytes
				}
			}
			// Mark last GC in the past so that even first call can trigger GC
			// Not updating the initialization code, since at the beginning of the collector no need to GC.
			ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
			numGCs := 0
			ml.runGCFn = func() {
				numGCs++
				gcCalled = true
			}

			memAllocMiB = tt.memAllocMiB[0]
			ml.CheckMemLimits()
			assert.True(t, ml.MustRefuse())

			// On windows, time has larger precision, and checking here again may return same time as "lastGCDone"
			// which will not trigger a new GC for 0 duration, update last GC with -1 millis.
			ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
			memAllocMiB = tt.memAllocMiB[1]
			ml.CheckMemLimits()
			assert.True(t, ml.MustRefuse())

			assert.Equal(t, tt.numGCs, numGCs)
		})
	}
}

func TestGCBackoffWhenIneffective(t *testing.T) {
	// When GC cannot reclaim memory (e.g., held by exporter queues), the
	// memory limiter should back off GC frequency to avoid burning CPU.
	// This test reproduces the scenario from issue #4981 where the collector
	// entered a degenerate state with force-GC on every tick.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MemoryLimitMiB:               50,
		MemorySpikeLimitMiB:          10,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	// Simulate memory stuck at 55 MiB (above hard limit of 50 MiB).
	// GC has no effect — memory stays the same after collection.
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = 55 * mibBytes
	}
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	numGCs := 0
	ml.runGCFn = func() {
		numGCs++
	}

	// First check: GC should fire (no backoff yet).
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())
	assert.Equal(t, 1, numGCs)
	assert.Equal(t, 1, ml.consecutiveIneffectiveGCs)

	// Subsequent checks with only 1ms between them: GC should NOT fire
	// because backoff has been applied due to ineffective GC.
	for range 5 {
		ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
		ml.CheckMemLimits()
	}
	assert.True(t, ml.MustRefuse())
	assert.Equal(t, 1, numGCs, "GC should not fire again due to backoff")

	// But after enough time passes (exceeding the backed-off interval),
	// GC should fire again.
	ml.lastGCDone = ml.lastGCDone.Add(-3 * time.Minute)
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())
	assert.Equal(t, 2, numGCs, "GC should fire after backoff interval expires")
	assert.Equal(t, 2, ml.consecutiveIneffectiveGCs)
}

func TestGCBackoffResetOnRecovery(t *testing.T) {
	// When memory drops below the soft limit (e.g., exporter queue drains),
	// the GC backoff counter should reset so the next pressure event starts
	// with fresh GC behavior.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MemoryLimitMiB:               50,
		MemorySpikeLimitMiB:          10,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	var currentMemAllocMiB uint64
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = currentMemAllocMiB * mibBytes
	}
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	numGCs := 0
	ml.runGCFn = func() {
		numGCs++
	}

	// Trigger ineffective GC (memory above hard limit, GC doesn't help).
	currentMemAllocMiB = 55
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())
	assert.Equal(t, 1, numGCs)
	assert.Equal(t, 1, ml.consecutiveIneffectiveGCs)

	// Simulate recovery: memory drops below soft limit.
	currentMemAllocMiB = 30
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse())
	assert.Equal(t, 0, ml.consecutiveIneffectiveGCs, "backoff should reset on recovery")

	// New pressure event: GC should fire immediately (no backoff from
	// the previous incident).
	currentMemAllocMiB = 55
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())
	assert.Equal(t, 2, numGCs, "GC should fire immediately after backoff reset")
}

func TestGCBackoffEarlyResetWhenMemoryBecomesReclaimable(t *testing.T) {
	// After an exporter recovers and its queue drains, the garbage exists but
	// runtime.MemStats.Alloc does not fall until another GC runs. If the
	// backoff has grown large, the forced GC is delayed and mustRefuse stays
	// true while the heap is reclaimable. This test verifies that the backoff
	// resets early when Alloc drops significantly compared to the last forced
	// GC observation (e.g., the Go runtime ran its own GC cycle).
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MemoryLimitMiB:               50,
		MemorySpikeLimitMiB:          10,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	var currentMemAllocMiB uint64
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = currentMemAllocMiB * mibBytes
	}
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	numGCs := 0
	ml.runGCFn = func() {
		numGCs++
	}

	// Build up backoff: memory stuck at 55 MiB, GC ineffective.
	currentMemAllocMiB = 55
	ml.CheckMemLimits()
	assert.Equal(t, 1, numGCs)
	assert.Equal(t, 1, ml.consecutiveIneffectiveGCs)
	assert.Equal(t, uint64(55*mibBytes), ml.lastAllocAfterGC)

	// Simulate Go runtime GC running and freeing memory: Alloc drops from
	// 55 to 48 MiB (still above soft limit of 40, but dropped >5% from the
	// 55 MiB observed after the last forced GC).
	currentMemAllocMiB = 48
	// Only 1ms has passed — normally backoff would prevent forced GC.
	ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
	ml.CheckMemLimits()

	// Backoff should have been reset because Alloc dropped >5% vs
	// lastAllocAfterGC, and a forced GC should have fired immediately.
	assert.Equal(t, 2, numGCs, "GC should fire after early backoff reset")

	// The new forced GC sees 48 MiB before and after, so it's ineffective,
	// but the counter should be 1 (reset then incremented).
	assert.Equal(t, 1, ml.consecutiveIneffectiveGCs)
}

func TestEffectiveGCInterval(t *testing.T) {
	cfg := &Config{
		CheckInterval:  1 * time.Second,
		MemoryLimitMiB: 50,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	// No backoff: returns base interval.
	ml.consecutiveIneffectiveGCs = 0
	assert.Equal(t, 10*time.Second, ml.effectiveGCInterval(10*time.Second))
	assert.Equal(t, time.Duration(0), ml.effectiveGCInterval(0))

	// With base interval of 0, uses CheckInterval as the base.
	ml.consecutiveIneffectiveGCs = 1
	assert.Equal(t, 2*time.Second, ml.effectiveGCInterval(0)) // 1s * 2^1

	ml.consecutiveIneffectiveGCs = 2
	assert.Equal(t, 4*time.Second, ml.effectiveGCInterval(0)) // 1s * 2^2

	ml.consecutiveIneffectiveGCs = 3
	assert.Equal(t, 8*time.Second, ml.effectiveGCInterval(0)) // 1s * 2^3

	// With explicit base interval, doubles from there.
	ml.consecutiveIneffectiveGCs = 1
	assert.Equal(t, 20*time.Second, ml.effectiveGCInterval(10*time.Second)) // 10s * 2^1

	ml.consecutiveIneffectiveGCs = 2
	assert.Equal(t, maxGCBackoffInterval, ml.effectiveGCInterval(10*time.Second)) // 10s * 2^2 = 40s, capped at 30s

	// Caps at maxGCBackoffInterval.
	ml.consecutiveIneffectiveGCs = 20
	assert.Equal(t, maxGCBackoffInterval, ml.effectiveGCInterval(10*time.Second))
	assert.Equal(t, maxGCBackoffInterval, ml.effectiveGCInterval(0))

	// Cap never reduces below the configured baseInterval. A user who sets
	// min_gc_interval_when_hard_limited: 60s should never see GC fire more
	// often than every 60s, even during backoff. Since the cap is
	// max(maxGCBackoffInterval, baseInterval) = max(30s, 60s) = 60s, the
	// doubled interval (120s) gets capped back to 60s.
	ml.consecutiveIneffectiveGCs = 1
	assert.Equal(t, 1*time.Minute, ml.effectiveGCInterval(1*time.Minute)) // 60s * 2 = 120s, capped at max(30s,60s)=60s
	ml.consecutiveIneffectiveGCs = 20
	assert.Equal(t, 1*time.Minute, ml.effectiveGCInterval(1*time.Minute)) // always capped at 60s
}

func TestGCEffectivenessWhenPressureResolved(t *testing.T) {
	// A GC that frees less than 5% of memory but brings usage below the soft
	// limit should be treated as effective. Otherwise the next pressure event
	// is backoff-throttled unnecessarily.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MemoryLimitMiB:               100,
		MemorySpikeLimitMiB:          5,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	// Soft limit = 100 - 5 = 95 MiB. Hard limit = 100 MiB.
	// Simulate: before GC = 100 MiB, after GC = 94 MiB.
	// That's only 6% reclaimed, but it resolved the pressure (94 < 95).
	var currentAllocMiB uint64
	gcCalled := false
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		if gcCalled {
			ms.Alloc = 94 * mibBytes
			gcCalled = false
		} else {
			ms.Alloc = currentAllocMiB * mibBytes
		}
	}
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	ml.runGCFn = func() {
		gcCalled = true
	}

	// First pressure event: GC resolves it.
	currentAllocMiB = 100
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse(), "should not refuse after GC resolved pressure")
	assert.Equal(t, 0, ml.consecutiveIneffectiveGCs, "GC that resolved pressure should be effective")

	// Second pressure event: GC should fire without any backoff.
	currentAllocMiB = 100
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse())
	assert.Equal(t, 0, ml.consecutiveIneffectiveGCs, "no stale backoff from previous event")
}
