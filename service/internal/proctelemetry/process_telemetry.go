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
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/unit"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
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
	otelProcessUptime asyncfloat64.Counter
	otelAllocMem      asyncint64.Gauge
	otelTotalAllocMem asyncint64.Counter
	otelSysMem        asyncint64.Gauge
	otelCpuSeconds    asyncfloat64.Counter
	otelRSSMemory     asyncint64.Gauge

	meter             otelmetric.Meter
	useOtelForMetrics bool

	// mu protects everything bellow.
	mu         sync.Mutex
	lastMsRead time.Time
	ms         *runtime.MemStats
}

// RegisterProcessMetrics creates a new set of processMetrics (mem, cpu) that can be used to measure
// basic information about this process.
func RegisterProcessMetrics(ctx context.Context, ocRegistry *metric.Registry, mp otelmetric.MeterProvider, registry *featuregate.Registry, ballastSizeBytes uint64) error {
	var err error
	pm := &processMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ballastSizeBytes:  ballastSizeBytes,
		ms:                &runtime.MemStats{},
		useOtelForMetrics: registry.IsEnabled(obsreportconfig.UseOtelForInternalMetricsfeatureGateID),
	}

	if pm.useOtelForMetrics {
		pm.meter = mp.Meter(scopeName)
	}
	pm.proc, err = process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return err
	}

	if pm.useOtelForMetrics {
		return pm.recordWithOtel(ctx)
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
	if err = pm.rssMemory.UpsertEntry(pm.updateRSSMemory); err != nil {
		return err
	}

	return nil
}

func (pm *processMetrics) recordWithOtel(ctx context.Context) error {
	var err error

	pm.otelProcessUptime, err = pm.meter.AsyncFloat64().Counter(
		"process_uptime",
		instrument.WithDescription("Uptime of the process"),
		instrument.WithUnit(unit.Unit("s")))
	if err != nil {
		return err
	}
	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelProcessUptime}, func(ctx context.Context) {
		pm.otelProcessUptime.Observe(ctx, pm.updateProcessUptime())
	})
	if err != nil {
		return err
	}

	pm.otelAllocMem, err = pm.meter.AsyncInt64().Gauge(
		"process_runtime_heap_alloc_bytes",
		instrument.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')"),
		instrument.WithUnit(unit.Bytes))
	if err != nil {
		return err
	}
	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelAllocMem}, func(ctx context.Context) {
		pm.otelAllocMem.Observe(ctx, pm.updateAllocMem())
	})
	if err != nil {
		return err
	}

	pm.otelTotalAllocMem, err = pm.meter.AsyncInt64().Counter(
		"process_runtime_total_alloc_bytes",
		instrument.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		instrument.WithUnit(unit.Bytes))
	if err != nil {
		return err
	}
	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelTotalAllocMem}, func(ctx context.Context) {
		pm.otelTotalAllocMem.Observe(ctx, pm.updateTotalAllocMem())
	})
	if err != nil {
		return err
	}

	pm.otelSysMem, err = pm.meter.AsyncInt64().Gauge(
		"process_runtime_total_sys_memory_bytes",
		instrument.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')"),
		instrument.WithUnit(unit.Bytes))
	if err != nil {
		return err
	}
	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelSysMem}, func(ctx context.Context) {
		pm.otelSysMem.Observe(ctx, pm.updateSysMem())
	})
	if err != nil {
		return err
	}

	pm.otelCpuSeconds, err = pm.meter.AsyncFloat64().Counter(
		"process_cpu_seconds",
		instrument.WithDescription("Total CPU user and system time in seconds"),
		instrument.WithUnit(unit.Unit("s")))
	if err != nil {
		return err
	}
	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelCpuSeconds}, func(ctx context.Context) {
		pm.otelCpuSeconds.Observe(ctx, pm.updateCPUSeconds())
	})
	if err != nil {
		return err
	}

	pm.otelRSSMemory, err = pm.meter.AsyncInt64().Gauge(
		"process_memory_rss",
		instrument.WithDescription("Total physical memory (resident set size)"),
		instrument.WithUnit(unit.Bytes))
	if err != nil {
		return err
	}
	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelRSSMemory}, func(ctx context.Context) {
		pm.otelRSSMemory.Observe(ctx, pm.updateRSSMemory())
	})
	if err != nil {
		return err
	}

	return nil
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
