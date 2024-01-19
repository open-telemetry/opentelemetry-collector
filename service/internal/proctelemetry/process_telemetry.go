// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/common"
	"github.com/shirou/gopsutil/v3/process"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.uber.org/multierr"
)

const (
	scopeName      = "go.opentelemetry.io/collector/service/process_telemetry"
	processNameKey = "process_name"
)

// processMetrics is a struct that contains views related to process metrics (cpu, mem, etc)
type processMetrics struct {
	startTimeUnixNano int64
	ballastSizeBytes  uint64
	proc              *process.Process
	context           context.Context

	otelProcessUptime otelmetric.Float64ObservableCounter
	otelAllocMem      otelmetric.Int64ObservableGauge
	otelTotalAllocMem otelmetric.Int64ObservableCounter
	otelSysMem        otelmetric.Int64ObservableGauge
	otelCPUSeconds    otelmetric.Float64ObservableCounter
	otelRSSMemory     otelmetric.Int64ObservableGauge

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
func RegisterProcessMetrics(mp otelmetric.MeterProvider, ballastSizeBytes uint64, opts ...RegisterOption) error {
	set := registerOption{}
	for _, opt := range opts {
		opt.apply(&set)
	}
	var err error
	pm := &processMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ballastSizeBytes:  ballastSizeBytes,
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

	return pm.record(mp.Meter(scopeName))
}

func (pm *processMetrics) record(meter otelmetric.Meter) error {
	var errs, err error

	pm.otelProcessUptime, err = meter.Float64ObservableCounter(
		"process_uptime",
		otelmetric.WithDescription("Uptime of the process"),
		otelmetric.WithUnit("s"),
		otelmetric.WithFloat64Callback(func(_ context.Context, o otelmetric.Float64Observer) error {
			o.Observe(pm.updateProcessUptime())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelAllocMem, err = meter.Int64ObservableGauge(
		"process_runtime_heap_alloc_bytes",
		otelmetric.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')"),
		otelmetric.WithUnit("By"),
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(pm.updateAllocMem())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelTotalAllocMem, err = meter.Int64ObservableCounter(
		"process_runtime_total_alloc_bytes",
		otelmetric.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		otelmetric.WithUnit("By"),
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(pm.updateTotalAllocMem())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelSysMem, err = meter.Int64ObservableGauge(
		"process_runtime_total_sys_memory_bytes",
		otelmetric.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')"),
		otelmetric.WithUnit("By"),
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(pm.updateSysMem())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelCPUSeconds, err = meter.Float64ObservableCounter(
		"process_cpu_seconds",
		otelmetric.WithDescription("Total CPU user and system time in seconds"),
		otelmetric.WithUnit("s"),
		otelmetric.WithFloat64Callback(func(_ context.Context, o otelmetric.Float64Observer) error {
			o.Observe(pm.updateCPUSeconds())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelRSSMemory, err = meter.Int64ObservableGauge(
		"process_memory_rss",
		otelmetric.WithDescription("Total physical memory (resident set size)"),
		otelmetric.WithUnit("By"),
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(pm.updateRSSMemory())
			return nil
		}))
	errs = multierr.Append(errs, err)

	return errs
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
	if pm.ballastSizeBytes > 0 {
		pm.ms.Alloc -= pm.ballastSizeBytes
		pm.ms.HeapAlloc -= pm.ballastSizeBytes
		pm.ms.HeapSys -= pm.ballastSizeBytes
		pm.ms.HeapInuse -= pm.ballastSizeBytes
	}
}
