// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/common"
	"github.com/shirou/gopsutil/v4/process"

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
	pm.proc, err = process.NewProcessWithContext(pm.context, int32(os.Getpid()))
	if err != nil {
		return err
	}

	_, err = metadata.NewTelemetryBuilder(cfg,
		metadata.WithProcessUptimeCallback(pm.updateProcessUptime),
		metadata.WithProcessRuntimeHeapAllocBytesCallback(pm.updateAllocMem),
		metadata.WithProcessRuntimeTotalAllocBytesCallback(pm.updateTotalAllocMem),
		metadata.WithProcessRuntimeTotalSysMemoryBytesCallback(pm.updateSysMem),
		metadata.WithProcessCPUSecondsCallback(pm.updateCPUSeconds),
		metadata.WithProcessMemoryRssCallback(pm.updateRSSMemory),
	)
	return err
}

func (pm *processMetrics) updateProcessUptime() float64 {
	now := time.Now().UnixNano()
	return float64(now-pm.startTimeUnixNano) / 1e9
}

func (pm *processMetrics) updateAllocMem() int64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	return int64(pm.ms.Alloc)
}

func (pm *processMetrics) updateTotalAllocMem() int64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	return int64(pm.ms.TotalAlloc)
}

func (pm *processMetrics) updateSysMem() int64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	return int64(pm.ms.Sys)
}

func (pm *processMetrics) updateCPUSeconds() float64 {
	times, err := pm.proc.TimesWithContext(pm.context)
	if err != nil {
		return 0
	}

	return times.User + times.System + times.Idle + times.Nice +
		times.Iowait + times.Irq + times.Softirq + times.Steal
}

func (pm *processMetrics) updateRSSMemory() int64 {
	mem, err := pm.proc.MemoryInfoWithContext(pm.context)
	if err != nil {
		return 0
	}
	return int64(mem.RSS)
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
