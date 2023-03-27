// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"go.opencensus.io/metric"
	"go.opencensus.io/stats"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
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

	processUptime *metric.Float64DerivedCumulative
	allocMem      *metric.Int64DerivedGauge
	totalAllocMem *metric.Int64DerivedCumulative
	sysMem        *metric.Int64DerivedGauge
	cpuSeconds    *metric.Float64DerivedCumulative
	rssMemory     *metric.Int64DerivedGauge

	// otel metrics
	otelProcessUptime instrument.Float64ObservableCounter
	otelAllocMem      instrument.Int64ObservableGauge
	otelTotalAllocMem instrument.Int64ObservableCounter
	otelSysMem        instrument.Int64ObservableGauge
	otelCPUSeconds    instrument.Float64ObservableCounter
	otelRSSMemory     instrument.Int64ObservableGauge

	// mu protects everything bellow.
	mu         sync.Mutex
	lastMsRead time.Time
	ms         *runtime.MemStats
}

// RegisterProcessMetrics creates a new set of processMetrics (mem, cpu) that can be used to measure
// basic information about this process.
func RegisterProcessMetrics(ocRegistry *metric.Registry, mp otelmetric.MeterProvider, useOtel bool, ballastSizeBytes uint64) error {
	var err error
	pm := &processMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ballastSizeBytes:  ballastSizeBytes,
		ms:                &runtime.MemStats{},
	}

	pm.proc, err = process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return err
	}

	if useOtel {
		return pm.recordWithOtel(mp.Meter(scopeName))
	}
	return pm.recordWithOC(ocRegistry)
}

func (pm *processMetrics) recordWithOC(ocRegistry *metric.Registry) error {
	var err error

	pm.processUptime, err = ocRegistry.AddFloat64DerivedCumulative(
		"process/uptime",
		metric.WithDescription("Uptime of the process"),
		metric.WithUnit(stats.UnitSeconds))
	if err != nil {
		return err
	}
	if err = pm.processUptime.UpsertEntry(pm.updateProcessUptime); err != nil {
		return err
	}

	pm.allocMem, err = ocRegistry.AddInt64DerivedGauge(
		"process/runtime/heap_alloc_bytes",
		metric.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	if err = pm.allocMem.UpsertEntry(pm.updateAllocMem); err != nil {
		return err
	}

	pm.totalAllocMem, err = ocRegistry.AddInt64DerivedCumulative(
		"process/runtime/total_alloc_bytes",
		metric.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	if err = pm.totalAllocMem.UpsertEntry(pm.updateTotalAllocMem); err != nil {
		return err
	}

	pm.sysMem, err = ocRegistry.AddInt64DerivedGauge(
		"process/runtime/total_sys_memory_bytes",
		metric.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	if err = pm.sysMem.UpsertEntry(pm.updateSysMem); err != nil {
		return err
	}

	pm.cpuSeconds, err = ocRegistry.AddFloat64DerivedCumulative(
		"process/cpu_seconds",
		metric.WithDescription("Total CPU user and system time in seconds"),
		metric.WithUnit(stats.UnitSeconds))
	if err != nil {
		return err
	}
	if err = pm.cpuSeconds.UpsertEntry(pm.updateCPUSeconds); err != nil {
		return err
	}

	pm.rssMemory, err = ocRegistry.AddInt64DerivedGauge(
		"process/memory/rss",
		metric.WithDescription("Total physical memory (resident set size)"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	return pm.rssMemory.UpsertEntry(pm.updateRSSMemory)
}

func (pm *processMetrics) recordWithOtel(meter otelmetric.Meter) error {
	var errs, err error

	pm.otelProcessUptime, err = meter.Float64ObservableCounter(
		"process_uptime",
		instrument.WithDescription("Uptime of the process"),
		instrument.WithUnit("s"),
		instrument.WithFloat64Callback(func(_ context.Context, o instrument.Float64Observer) error {
			o.Observe(pm.updateProcessUptime())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelAllocMem, err = meter.Int64ObservableGauge(
		"process_runtime_heap_alloc_bytes",
		instrument.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')"),
		instrument.WithUnit("By"),
		instrument.WithInt64Callback(func(_ context.Context, o instrument.Int64Observer) error {
			o.Observe(pm.updateAllocMem())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelTotalAllocMem, err = meter.Int64ObservableCounter(
		"process_runtime_total_alloc_bytes",
		instrument.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		instrument.WithUnit("By"),
		instrument.WithInt64Callback(func(_ context.Context, o instrument.Int64Observer) error {
			o.Observe(pm.updateTotalAllocMem())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelSysMem, err = meter.Int64ObservableGauge(
		"process_runtime_total_sys_memory_bytes",
		instrument.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')"),
		instrument.WithUnit("By"),
		instrument.WithInt64Callback(func(_ context.Context, o instrument.Int64Observer) error {
			o.Observe(pm.updateSysMem())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelCPUSeconds, err = meter.Float64ObservableCounter(
		"process_cpu_seconds",
		instrument.WithDescription("Total CPU user and system time in seconds"),
		instrument.WithUnit("s"),
		instrument.WithFloat64Callback(func(_ context.Context, o instrument.Float64Observer) error {
			o.Observe(pm.updateCPUSeconds())
			return nil
		}))
	errs = multierr.Append(errs, err)

	pm.otelRSSMemory, err = meter.Int64ObservableGauge(
		"process_memory_rss",
		instrument.WithDescription("Total physical memory (resident set size)"),
		instrument.WithUnit("By"),
		instrument.WithInt64Callback(func(_ context.Context, o instrument.Int64Observer) error {
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
	times, err := pm.proc.Times()
	if err != nil {
		return 0
	}

	return times.User + times.System + times.Idle + times.Nice +
		times.Iowait + times.Irq + times.Softirq + times.Steal
}

func (pm *processMetrics) updateRSSMemory() int64 {
	mem, err := pm.proc.MemoryInfo()
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
