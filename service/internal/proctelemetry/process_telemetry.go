// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"errors"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/common"
	"github.com/shirou/gopsutil/v4/process"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

// processMetrics is a struct that contains views related to process metrics (cpu, mem, etc)
type processMetrics struct {
	startTimeUnixNano int64
	proc              *process.Process
	context           context.Context

	// mu protects everything bellow.
	mu         sync.Mutex
	lastMsRead time.Time
	ms         *runtime.MemStats
}

type RegisterOption interface {
	apply(*registerOption)
}

type registerOption struct {
	hostProc string
}

type registerOptionFunc func(*registerOption)

func (fn registerOptionFunc) apply(set *registerOption) {
	fn(set)
}

// WithHostProc overrides the /proc folder on Linux used by process telemetry.
func WithHostProc(hostProc string) RegisterOption {
	return registerOptionFunc(func(uo *registerOption) {
		uo.hostProc = hostProc
	})
}

// RegisterProcessMetrics creates a new set of processMetrics (mem, cpu) that can be used to measure
// basic information about this process.
func RegisterProcessMetrics(cfg component.TelemetrySettings, opts ...RegisterOption) error {
	set := registerOption{}
	for _, opt := range opts {
		opt.apply(&set)
	}
	var err error
	pm := &processMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ms:                &runtime.MemStats{},
	}

	ctx := context.Background()
	if set.hostProc != "" {
		ctx = context.WithValue(ctx, common.EnvKey, common.EnvMap{common.HostProcEnvKey: set.hostProc})
	}
	pm.context = ctx
	//nolint:gosec
	pm.proc, err = process.NewProcessWithContext(pm.context, int32(os.Getpid()))
	if err != nil {
		return err
	}

	tb, err := metadata.NewTelemetryBuilder(cfg)
	if err != nil {
		return err
	}
	return errors.Join(
		tb.RegisterProcessUptimeCallback(pm.updateProcessUptime),
		tb.RegisterProcessRuntimeHeapAllocBytesCallback(pm.updateAllocMem),
		tb.RegisterProcessRuntimeTotalAllocBytesCallback(pm.updateTotalAllocMem),
		tb.RegisterProcessRuntimeTotalSysMemoryBytesCallback(pm.updateSysMem),
		tb.RegisterProcessCPUSecondsCallback(pm.updateCPUSeconds),
		tb.RegisterProcessMemoryRssCallback(pm.updateRSSMemory),
	)
}

func (pm *processMetrics) updateProcessUptime(_ context.Context, obs metric.Float64Observer) error {
	now := time.Now().UnixNano()
	obs.Observe(float64(now-pm.startTimeUnixNano) / 1e9)
	return nil
}

func (pm *processMetrics) updateAllocMem(_ context.Context, obs metric.Int64Observer) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	//nolint:gosec
	obs.Observe(int64(pm.ms.Alloc))
	return nil
}

func (pm *processMetrics) updateTotalAllocMem(_ context.Context, obs metric.Int64Observer) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	//nolint:gosec
	obs.Observe(int64(pm.ms.TotalAlloc))
	return nil
}

func (pm *processMetrics) updateSysMem(_ context.Context, obs metric.Int64Observer) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	//nolint:gosec
	obs.Observe(int64(pm.ms.Sys))
	return nil
}

func (pm *processMetrics) updateCPUSeconds(_ context.Context, obs metric.Float64Observer) error {
	times, err := pm.proc.TimesWithContext(pm.context)
	if err != nil {
		return err
	}

	obs.Observe(times.User + times.System + times.Idle + times.Nice +
		times.Iowait + times.Irq + times.Softirq + times.Steal)
	return nil
}

func (pm *processMetrics) updateRSSMemory(_ context.Context, obs metric.Int64Observer) error {
	mem, err := pm.proc.MemoryInfoWithContext(pm.context)
	if err != nil {
		return err
	}
	//nolint:gosec
	obs.Observe(int64(mem.RSS))
	return nil
}

func (pm *processMetrics) readMemStatsIfNeeded() {
	now := time.Now()
	// If last time we read was less than one second ago just reuse the values
	if now.Sub(pm.lastMsRead) < time.Second {
		return
	}
	pm.lastMsRead = now
	runtime.ReadMemStats(pm.ms)
}
