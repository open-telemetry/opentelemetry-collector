// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiter

import (
	"context"
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
			ml.readMemStatsFn = func(ms *runtime.MemStats) {
				ms.Alloc = memAllocMiB * mibBytes
			}
			// Mark last GC in the past so that even first call can trigger GC
			// Not updating the initialization code, since at the beginning of the collector no need to GC.
			ml.lastGCDone = ml.lastGCDone.Add(-time.Minute)
			numGCs := 0
			ml.runGCFn = func() {
				numGCs++
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

// Tests the recovery within the same tick
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
