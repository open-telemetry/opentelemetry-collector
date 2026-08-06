// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/internal/memorylimiter/iruntime"
)

type mockHost struct {
	events []*componentstatus.Event
}

func (m *mockHost) GetExtensions() map[component.ID]component.Component {
	return nil
}

func (m *mockHost) Report(e *componentstatus.Event) {
	m.events = append(m.events, e)
}

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

	host := &mockHost{}
	ml.host = host

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.CheckMemLimits()
	assert.Len(t, host.events, 1)
	assert.Equal(t, componentstatus.StatusOK, host.events[len(host.events)-1].Status())
	assert.False(t, ml.MustRefuse())

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.CheckMemLimits()
	assert.Len(t, host.events, 2)
	assert.Equal(t, componentstatus.StatusRecoverableError, host.events[len(host.events)-1].Status())
	assert.True(t, ml.MustRefuse())

	// Check spike limit
	ml.usageChecker.memSpikeLimit = 512 * mibBytes

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.CheckMemLimits()
	assert.Len(t, host.events, 3)
	assert.Equal(t, componentstatus.StatusOK, host.events[len(host.events)-1].Status())
	assert.False(t, ml.MustRefuse())

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.CheckMemLimits()
	assert.Len(t, host.events, 4)
	assert.Equal(t, componentstatus.StatusRecoverableError, host.events[len(host.events)-1].Status())
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

func TestRefreshPercentageMemoryLimit(t *testing.T) {
	var totalMemory uint64 = 100 * mibBytes
	getMemory := func() (uint64, error) {
		return totalMemory, nil
	}
	originalGetMemory := GetMemoryFn
	GetMemoryFn = getMemory
	t.Cleanup(func() { GetMemoryFn = originalGetMemory })

	ml, err := NewMemoryLimiter(&Config{
		CheckInterval:         time.Minute,
		MemoryLimitPercentage: 80,
		MemorySpikePercentage: 20,
	}, zap.NewNop())
	require.NoError(t, err)
	assert.Equal(t, uint64(80*mibBytes), ml.usageChecker.memAllocLimit)
	assert.Equal(t, uint64(20*mibBytes), ml.usageChecker.memSpikeLimit)

	totalMemory = 200 * mibBytes
	ml.refreshPercentageMemoryLimit()
	assert.Equal(t, uint64(160*mibBytes), ml.usageChecker.memAllocLimit)
	assert.Equal(t, uint64(40*mibBytes), ml.usageChecker.memSpikeLimit)
}

func TestRefreshPercentageMemoryLimitRetainsPreviousLimitOnError(t *testing.T) {
	ml, err := NewMemoryLimiter(&Config{
		CheckInterval:         time.Minute,
		MemoryLimitPercentage: 80,
		MemorySpikePercentage: 20,
	}, zap.NewNop())
	require.NoError(t, err)
	previous := ml.usageChecker
	ml.getMemoryFn = func() (uint64, error) { return 0, errors.New("cgroup unavailable") }
	ml.refreshPercentageMemoryLimit()
	assert.Equal(t, previous, ml.usageChecker)
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

func TestCheckMemLimitsHealthEvents(t *testing.T) {
	// Config used across all cases: limit=50 MiB, spike=10 MiB → soft=40 MiB, hard=50 MiB.
	// GC is a no-op in all cases; GC-triggered memory recovery is covered by TestGCRecovery.
	tests := []struct {
		name           string
		memAllocMiB    []uint64
		expectedEvents []componentstatus.Status
		expectRefusing bool
	}{
		{
			name:           "below soft limit reports StatusOK",
			memAllocMiB:    []uint64{30},
			expectedEvents: []componentstatus.Status{componentstatus.StatusOK},
			expectRefusing: false,
		},
		{
			name:           "recovery from refusing reports StatusOK",
			memAllocMiB:    []uint64{45, 30},
			expectedEvents: []componentstatus.Status{componentstatus.StatusRecoverableError, componentstatus.StatusOK},
			expectRefusing: false,
		},
		{
			name:           "above soft limit transitions to StatusRecoverableError",
			memAllocMiB:    []uint64{45},
			expectedEvents: []componentstatus.Status{componentstatus.StatusRecoverableError},
			expectRefusing: true,
		},
		{
			name:           "already refusing emits no new event",
			memAllocMiB:    []uint64{45, 45},
			expectedEvents: []componentstatus.Status{componentstatus.StatusRecoverableError},
			expectRefusing: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml, err := NewMemoryLimiter(&Config{
				CheckInterval:       1 * time.Minute,
				MemoryLimitMiB:      50,
				MemorySpikeLimitMiB: 10,
			}, zap.NewNop())
			require.NoError(t, err)
			host := &mockHost{}
			ml.host = host
			ml.runGCFn = func() {}
			var currentMemMiB uint64
			ml.readMemStatsFn = func(ms *runtime.MemStats) { ms.Alloc = currentMemMiB * mibBytes }

			for _, memMiB := range tt.memAllocMiB {
				currentMemMiB = memMiB
				ml.CheckMemLimits()
			}

			assert.Len(t, host.events, len(tt.expectedEvents))
			for i, expectedStatus := range tt.expectedEvents {
				assert.Equal(t, expectedStatus, host.events[i].Status())
			}
			assert.Equal(t, tt.expectRefusing, ml.MustRefuse())
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
				MaxGCIntervalWhenSoftLimited: 30 * time.Second,
				MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
				MaxGCIntervalWhenSoftLimited: 30 * time.Second,
				MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
				MaxGCIntervalWhenSoftLimited: 30 * time.Second,
				MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
				MaxGCIntervalWhenSoftLimited: 30 * time.Second,
				MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
				MaxGCIntervalWhenSoftLimited: 30 * time.Second,
				MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
	assert.NotZero(t, ml.currentHardGCInterval, "backoff should be armed after first ineffective GC")
	firstInterval := ml.currentHardGCInterval

	// Subsequent checks with only 1ms between them: GC should NOT fire
	// because backoff has been applied due to ineffective GC.
	for range 5 {
		ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
		ml.CheckMemLimits()
	}
	assert.True(t, ml.MustRefuse())
	assert.Equal(t, 1, numGCs, "GC should not fire again due to backoff")
	assert.Equal(t, firstInterval, ml.currentHardGCInterval, "backoff interval should not grow without another forced GC")

	// But after enough time passes (exceeding the backed-off interval),
	// GC should fire again and grow the backoff.
	ml.lastGCDone = ml.lastGCDone.Add(-3 * time.Minute)
	ml.CheckMemLimits()
	assert.True(t, ml.MustRefuse())
	assert.Equal(t, 2, numGCs, "GC should fire after backoff interval expires")
	assert.Greater(t, ml.currentHardGCInterval, firstInterval, "backoff should double after second ineffective GC, capped at max")
}

func TestGCBackoffResetOnRecovery(t *testing.T) {
	// When memory drops below the soft limit (e.g., exporter queue drains),
	// both per-path currentGCIntervals should reset to zero so the next pressure
	// event starts with fresh GC behavior.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
	assert.NotZero(t, ml.currentHardGCInterval)

	// Simulate recovery: memory drops below soft limit.
	currentMemAllocMiB = 30
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse())
	assert.Zero(t, ml.currentHardGCInterval, "hard backoff should reset on recovery")
	assert.Zero(t, ml.currentSoftGCInterval, "soft backoff should reset on recovery")

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
	// true while the heap is reclaimable. The top-of-tick checkLimitAndBackoff
	// call detects this by comparing current Alloc against lastStats.Alloc;
	// a >=5% drop resets the per-path currentGCIntervals to zero so the next
	// forced GC fires immediately.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
	assert.NotZero(t, ml.currentHardGCInterval)
	assert.Equal(t, uint64(55*mibBytes), ml.lastStats.Alloc)

	// Simulate Go runtime GC running and freeing memory: Alloc drops from
	// 55 to 48 MiB (still above soft limit of 40, but dropped >5% from the
	// 55 MiB observed after the last forced GC).
	currentMemAllocMiB = 48
	// Only 1ms has passed — normally backoff would prevent forced GC.
	ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
	ml.CheckMemLimits()

	// Top-of-tick comparison reset both currentGCIntervals to 0 (48 <= 55*0.95).
	// The soft-limit branch (48 < hard limit 50) then ran a forced GC with
	// gateInterval = max(MinGCIntervalWhenSoftLimited=0, currentSoftGCInterval=0) = 0.
	assert.Equal(t, 2, numGCs, "GC should fire after early backoff reset")
	// The second forced GC sees 48 before and 48 after — ineffective — so both
	// per-path backoffs re-arm in lockstep with a fresh seed.
	assert.NotZero(t, ml.currentSoftGCInterval)
	assert.NotZero(t, ml.currentHardGCInterval)
}

func TestGCBackoffEarlyResetAboveHardLimit(t *testing.T) {
	// Same early-reset behavior as TestGCBackoffEarlyResetWhenMemoryBecomesReclaimable
	// but verifies the hard-limit code path: memory stays above the hard limit
	// while still dropping enough (>5%) versus the last observation
	// for the backoff to reset and a forced GC to fire on the next tick.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 30 * time.Second,
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

	// Build up backoff: memory stuck at 100 MiB (well above hard limit), GC ineffective.
	currentMemAllocMiB = 100
	ml.CheckMemLimits()
	assert.Equal(t, 1, numGCs)
	assert.NotZero(t, ml.currentHardGCInterval)
	assert.Equal(t, uint64(100*mibBytes), ml.lastStats.Alloc)

	// Memory drops to 60 MiB — still above hard limit (50), but dropped >5%
	// versus the 100 MiB observed after the last forced GC.
	currentMemAllocMiB = 60
	// Only 1ms has passed — normally backoff would prevent forced GC.
	ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
	ml.CheckMemLimits()

	// Top-of-tick reset → forced GC fired.
	assert.Equal(t, 2, numGCs, "GC should fire after early backoff reset on hard-limit path")
	assert.NotZero(t, ml.currentHardGCInterval)
}

func TestNextBackoffInterval(t *testing.T) {
	// Pure-logic exercise of the doubling helper, covering: (a) ineffective
	// growth from floor, (b) doubling, (c) cap at max(configMax, configMin),
	// (d) floor at max(configMin, checkInterval*0.95), (e) configMin >
	// configMax case, (f) configMax=0 pins to configMin (disabled-path).
	mlWith := func(checkInterval time.Duration) *MemoryLimiter {
		cfg := &Config{
			CheckInterval:                checkInterval,
			MinGCIntervalWhenHardLimited: 0,
			MaxGCIntervalWhenSoftLimited: 30 * time.Second,
			MaxGCIntervalWhenHardLimited: 30 * time.Second,
			MemoryLimitMiB:               100,
			MemorySpikeLimitMiB:          5,
		}
		ml, err := NewMemoryLimiter(cfg, zap.NewNop())
		require.NoError(t, err)
		return ml
	}

	t.Run("grows_from_floor_then_doubles", func(t *testing.T) {
		ml := mlWith(1 * time.Second)
		// First ineffective GC seeds from the floor max(0, 950ms) = 950ms,
		// doubled → 1.9s.
		got := ml.nextBackoffInterval(0, 0, 30*time.Second)
		assert.Equal(t, 1900*time.Millisecond, got)
		// Second doubles to 3.8s.
		got = ml.nextBackoffInterval(got, 0, 30*time.Second)
		assert.Equal(t, 3800*time.Millisecond, got)
	})

	t.Run("doubling_caps_at_configMax", func(t *testing.T) {
		ml := mlWith(1 * time.Second)
		got := ml.nextBackoffInterval(25*time.Second, 0, 30*time.Second)
		assert.Equal(t, 30*time.Second, got)
	})

	t.Run("configMin_acts_as_floor_above_check_interval", func(t *testing.T) {
		// CheckInterval 1s → check floor 950ms. configMin 60s wins.
		ml := mlWith(1 * time.Second)
		got := ml.nextBackoffInterval(0, 60*time.Second, 30*time.Second)
		// Floor 60s, doubled → 120s, capped at max(30s, 60s) = 60s.
		assert.Equal(t, 60*time.Second, got)
	})

	t.Run("configMin_caps_doubling_above_configMax", func(t *testing.T) {
		// configMin 60s > configMax 30s — cap respects configMin.
		ml := mlWith(1 * time.Second)
		got := ml.nextBackoffInterval(40*time.Second, 60*time.Second, 30*time.Second)
		assert.Equal(t, 60*time.Second, got, "cap is max(configMax, configMin)")
	})

	t.Run("configMax_zero_pins_to_configMin", func(t *testing.T) {
		// configMax=0 disables backoff: return configMin regardless of current.
		ml := mlWith(1 * time.Second)
		assert.Equal(t, time.Duration(0), ml.nextBackoffInterval(10*time.Second, 0, 0))
		assert.Equal(t, 5*time.Second, ml.nextBackoffInterval(20*time.Second, 5*time.Second, 0))
	})
}

func TestCheckLimitAndBackoff(t *testing.T) {
	// Exercises checkLimitAndBackoff's lockstep advancement of both per-path
	// intervals on ineffective forced GC, and the both-paths reset on
	// effective observation.
	newML := func() *MemoryLimiter {
		cfg := &Config{
			CheckInterval:                1 * time.Second,
			MinGCIntervalWhenHardLimited: 0,
			MaxGCIntervalWhenSoftLimited: 30 * time.Second,
			MaxGCIntervalWhenHardLimited: 30 * time.Second,
			MemoryLimitMiB:               100,
			MemorySpikeLimitMiB:          5,
		}
		ml, err := NewMemoryLimiter(cfg, zap.NewNop())
		require.NoError(t, err)
		return ml
	}

	t.Run("effective_by_soft_resets_both_intervals", func(t *testing.T) {
		ml := newML()
		ml.currentSoftGCInterval = 10 * time.Second
		ml.currentHardGCInterval = 15 * time.Second
		ml.lastStats = &runtime.MemStats{Alloc: 100 * mibBytes}
		ms := &runtime.MemStats{Alloc: 50 * mibBytes}
		above := ml.checkLimitAndBackoff(ms, true)
		assert.False(t, above)
		assert.Zero(t, ml.currentSoftGCInterval)
		assert.Zero(t, ml.currentHardGCInterval)
	})

	t.Run("effective_by_reclaim_resets_both_intervals", func(t *testing.T) {
		ml := newML()
		ml.currentSoftGCInterval = 10 * time.Second
		ml.currentHardGCInterval = 15 * time.Second
		ml.lastStats = &runtime.MemStats{Alloc: 110 * mibBytes}
		// 110 → 99 is a 10% drop, above the 5% threshold, but still above
		// soft limit (95 MiB) so it counts as effective-by-reclaim.
		ms := &runtime.MemStats{Alloc: 99 * mibBytes}
		above := ml.checkLimitAndBackoff(ms, true)
		assert.True(t, above, "still above soft limit (95 MiB)")
		assert.Zero(t, ml.currentSoftGCInterval)
		assert.Zero(t, ml.currentHardGCInterval)
	})

	t.Run("ineffective_grows_both_intervals_in_lockstep", func(t *testing.T) {
		ml := newML()
		ml.lastStats = &runtime.MemStats{Alloc: 100 * mibBytes}
		ms := &runtime.MemStats{Alloc: 99 * mibBytes}
		ml.checkLimitAndBackoff(ms, true)
		// Both advance from 0 to 1.9s (floor 950ms doubled), capped at 30s.
		assert.Equal(t, 1900*time.Millisecond, ml.currentSoftGCInterval)
		assert.Equal(t, 1900*time.Millisecond, ml.currentHardGCInterval)
	})

	t.Run("not_didGC_does_not_grow", func(t *testing.T) {
		// Top-of-tick call (didGC=false) on ineffective state must not grow
		// either interval (only forced GCs cause growth).
		ml := newML()
		ml.lastStats = &runtime.MemStats{Alloc: 100 * mibBytes}
		ml.currentSoftGCInterval = 5 * time.Second
		ml.currentHardGCInterval = 7 * time.Second
		ms := &runtime.MemStats{Alloc: 99 * mibBytes}
		ml.checkLimitAndBackoff(ms, false)
		assert.Equal(t, 5*time.Second, ml.currentSoftGCInterval)
		assert.Equal(t, 7*time.Second, ml.currentHardGCInterval)
	})

	t.Run("ineffective_with_one_path_disabled_only_grows_enabled_path", func(t *testing.T) {
		// max_hard=0 pins the hard path's interval to min_hard; the soft
		// path's interval grows normally.
		cfg := &Config{
			CheckInterval:                1 * time.Second,
			MinGCIntervalWhenSoftLimited: 10 * time.Second,
			MinGCIntervalWhenHardLimited: 0,
			MaxGCIntervalWhenSoftLimited: 30 * time.Second,
			MaxGCIntervalWhenHardLimited: 0,
			MemoryLimitMiB:               100,
			MemorySpikeLimitMiB:          5,
		}
		ml, err := NewMemoryLimiter(cfg, zap.NewNop())
		require.NoError(t, err)
		ml.lastStats = &runtime.MemStats{Alloc: 100 * mibBytes}
		ms := &runtime.MemStats{Alloc: 99 * mibBytes}
		ml.checkLimitAndBackoff(ms, true)
		assert.Equal(t, 20*time.Second, ml.currentSoftGCInterval, "soft grows: floor 10s doubled to 20s")
		assert.Zero(t, ml.currentHardGCInterval, "hard pinned to min_hard=0 (backoff disabled)")
	})
}

func TestBackoffDisabledOptOut(t *testing.T) {
	// When max_gc_interval_when_*_limited are both 0, the limiter must
	// restore the pre-fix behavior: forced GC fires on every check tick if the
	// configured min interval is zero, regardless of how ineffective each GC is.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MaxGCIntervalWhenSoftLimited: 0,
		MaxGCIntervalWhenHardLimited: 0,
		MemoryLimitMiB:               50,
		MemorySpikeLimitMiB:          10,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = 55 * mibBytes
	}
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	numGCs := 0
	ml.runGCFn = func() {
		numGCs++
	}

	for range 5 {
		ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
		ml.CheckMemLimits()
	}
	assert.Equal(t, 5, numGCs, "with backoff disabled, every tick should force a GC")
	assert.Zero(t, ml.currentSoftGCInterval, "currentSoftGCInterval must remain zero when backoff is disabled")
	assert.Zero(t, ml.currentHardGCInterval, "currentHardGCInterval must remain zero when backoff is disabled")
}

func TestPerPathBackoffIsIndependent(t *testing.T) {
	// Regression for the per-path disable promise: setting max_when_hard=0
	// must keep the hard path's currentHardGCInterval pinned to
	// min_when_hard, regardless of any backoff accumulated on the soft path.
	// With max_when_hard=0 + min_when_hard=0, the hard path GCs every tick
	// even while the soft path has its backoff armed up to 30s.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenSoftLimited: 10 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 0,
		MemoryLimitMiB:               50,
		MemorySpikeLimitMiB:          10,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	var currentMemAllocMiB uint64
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = currentMemAllocMiB * mibBytes
	}
	numGCs := 0
	ml.runGCFn = func() {
		numGCs++
	}

	// Phase 1: soft pressure (above soft 40, below hard 50). Run an
	// ineffective forced GC; soft backoff arms, hard stays pinned at 0.
	currentMemAllocMiB = 45
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	ml.CheckMemLimits()
	assert.Equal(t, 1, numGCs, "soft branch should fire forced GC")
	assert.NotZero(t, ml.currentSoftGCInterval, "soft backoff should be armed")
	assert.Zero(t, ml.currentHardGCInterval, "hard backoff is disabled, must stay pinned to min_hard=0")

	// Phase 2: hard pressure (above hard 50). max_hard=0 keeps the hard
	// gate at max(min_hard=0, currentHardGCInterval=0) = 0, so the hard
	// path fires every tick even with only 1ms between ticks.
	currentMemAllocMiB = 55
	for range 5 {
		ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
		ml.CheckMemLimits()
	}
	assert.Equal(t, 6, numGCs, "hard path must fire every tick when its backoff is disabled")
	assert.Zero(t, ml.currentHardGCInterval, "hard backoff must stay pinned (disabled)")
}

func TestNewDefaultConfigEnablesBackoffCap(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.Equal(t, 30*time.Second, cfg.MaxGCIntervalWhenSoftLimited, "soft backoff cap should default to 30s")
	assert.Equal(t, 30*time.Second, cfg.MaxGCIntervalWhenHardLimited, "hard backoff cap should default to 30s")
}

func TestBackoffGrowsBothPathsInLockstep(t *testing.T) {
	// Backing off only the
	// active path lets the inactive path snap to full-speed GC the moment
	// pressure shifts (e.g., back off soft a few times, then suddenly GC at
	// full speed once hard is breached). Both per-path intervals must grow
	// together on each ineffective forced GC.
	//
	// Phase 1 saturates currentSoftGCInterval at max_soft=30s via ineffective
	// soft GCs. Phase 2 bumps to hard pressure with only 1ms between ticks.
	// Under lockstep, currentHardGCInterval is already 30s → gate=max(0,30s)=30s
	// → 1ms < 30s → NO hard GC fires. Under the old only-grow-active design,
	// currentHardGCInterval stays 0 → gate=0 → 1ms > 0 → hard GC fires
	// immediately. Discriminator: numGCs unchanged in Phase 2 (lockstep)
	// vs +1 (only-grow-active).
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenSoftLimited: 0,
		MinGCIntervalWhenHardLimited: 0,
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 30 * time.Second,
		MemoryLimitMiB:               50,
		MemorySpikeLimitMiB:          10,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)

	var currentMemAllocMiB uint64
	ml.readMemStatsFn = func(ms *runtime.MemStats) {
		ms.Alloc = currentMemAllocMiB * mibBytes
	}
	numGCs := 0
	ml.runGCFn = func() {
		numGCs++
	}

	// Phase 1: saturate soft backoff at max_soft=30s. Each ineffective
	// soft GC doubles both intervals; floor 0.95s, cap 30s → 5 rounds suffice.
	currentMemAllocMiB = 45
	for range 6 {
		ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
		ml.CheckMemLimits()
	}
	assert.Equal(t, 30*time.Second, ml.currentSoftGCInterval, "soft saturates at max_soft")
	assert.Equal(t, 30*time.Second, ml.currentHardGCInterval, "hard grew in lockstep")
	phase1GCs := numGCs

	// Phase 2: bump to hard pressure with only 1ms between ticks.
	currentMemAllocMiB = 55
	for range 5 {
		ml.lastGCDone = ml.lastGCDone.Add(-1 * time.Millisecond)
		ml.CheckMemLimits()
	}
	assert.Equal(t, phase1GCs, numGCs, "hard gate inherited 30s backoff from lockstep — no GC fires within 1ms")
}

func TestGCEffectivenessWhenPressureResolved(t *testing.T) {
	// A GC that frees less than 5% of memory but brings usage below the soft
	// limit should be treated as effective. Otherwise the next pressure event
	// is backoff-throttled unnecessarily.
	cfg := &Config{
		CheckInterval:                1 * time.Second,
		MinGCIntervalWhenHardLimited: 0,
		MaxGCIntervalWhenSoftLimited: 30 * time.Second,
		MaxGCIntervalWhenHardLimited: 30 * time.Second,
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
	assert.Zero(t, ml.currentHardGCInterval, "GC that resolved pressure should be effective")
	assert.Zero(t, ml.currentSoftGCInterval, "GC that resolved pressure should be effective")

	// Second pressure event: GC should fire without any backoff.
	currentAllocMiB = 100
	ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
	ml.CheckMemLimits()
	assert.False(t, ml.MustRefuse())
	assert.Zero(t, ml.currentHardGCInterval, "no stale backoff from previous event")
	assert.Zero(t, ml.currentSoftGCInterval, "no stale backoff from previous event")
}

func TestGCRecovery(t *testing.T) {
	tests := []struct {
		name          string
		initialMemMiB uint64 // determines which limit branch (soft vs hard) is taken
	}{
		{
			name:          "soft limit breach recovered by GC",
			initialMemMiB: 45, // above soft (40 MiB), below hard (50 MiB)
		},
		{
			name:          "hard limit breach recovered by GC",
			initialMemMiB: 55, // above hard (50 MiB)
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml, err := NewMemoryLimiter(&Config{
				CheckInterval:       1 * time.Minute,
				MemoryLimitMiB:      50,
				MemorySpikeLimitMiB: 10,
			}, zap.NewNop())
			require.NoError(t, err)

			host := &mockHost{}
			ml.host = host

			// Ensure the GC-interval guard passes on the first check.
			ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)

			var currentMemMiB uint64
			currentMemMiB = tt.initialMemMiB
			// runGCFn simulates GC reclaiming memory so that the subsequent
			// readMemStatsFn call returns a value below the soft limit (40 MiB).
			ml.runGCFn = func() { currentMemMiB = 30 }
			ml.readMemStatsFn = func(ms *runtime.MemStats) { ms.Alloc = currentMemMiB * mibBytes }

			ml.CheckMemLimits()

			// GC recovered memory within the same tick: only StatusOK, no RecoverableError.
			require.Len(t, host.events, 1)
			assert.Equal(t, componentstatus.StatusOK, host.events[0].Status())
			assert.False(t, ml.MustRefuse())
		})
	}
}

func TestStart(t *testing.T) {
	host := componenttest.NewNopHost()
	cfg := &Config{
		CheckInterval:                1 * time.Minute,
		MinGCIntervalWhenSoftLimited: 10 * time.Second,
		MemoryLimitMiB:               50,
		MemorySpikeLimitMiB:          10,
	}
	ml, err := NewMemoryLimiter(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, ml.Start(context.Background(), host))
	assert.Equal(t, host, ml.host)
	require.NoError(t, ml.Shutdown(context.Background()))
}
