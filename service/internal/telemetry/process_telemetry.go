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

package telemetry // import "go.opentelemetry.io/collector/service/internal/telemetry"

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"go.opencensus.io/metric"
	"go.opencensus.io/stats"
)

// ProcessMetrics is a struct that contains views related to process metrics (cpu, mem, etc)
type ProcessMetrics struct {
	startTimeUnixNano int64
	ballastSizeBytes  uint64

	processUptime *metric.Float64DerivedCumulative
	allocMem      *metric.Int64DerivedGauge
	totalAllocMem *metric.Int64DerivedCumulative
	sysMem        *metric.Int64DerivedGauge
	cpuSeconds    *metric.Float64DerivedCumulative
	rssMemory     *metric.Int64DerivedGauge

	proc *process.Process
}

// RegisterProcessMetrics creates a new set of ProcessMetrics (mem, cpu) that can be used to measure
// basic information about this process.
func RegisterProcessMetrics(registry *metric.Registry, ballastSizeBytes uint64) error {
	pmv := &ProcessMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ballastSizeBytes:  ballastSizeBytes,
	}

	var err error
	pmv.processUptime, err = registry.AddFloat64DerivedCumulative(
		"process/uptime",
		metric.WithDescription("Uptime of the process"),
		metric.WithUnit(stats.UnitSeconds))
	if err != nil {
		return err
	}
	if err = pmv.processUptime.UpsertEntry(pmv.updateProcessUptime); err != nil {
		return err
	}

	pmv.allocMem, err = registry.AddInt64DerivedGauge(
		"process/runtime/heap_alloc_bytes",
		metric.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	if err = pmv.allocMem.UpsertEntry(pmv.updateAllocMem); err != nil {
		return err
	}

	pmv.totalAllocMem, err = registry.AddInt64DerivedCumulative(
		"process/runtime/total_alloc_bytes",
		metric.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	if err = pmv.totalAllocMem.UpsertEntry(pmv.updateTotalAllocMem); err != nil {
		return err
	}

	pmv.sysMem, err = registry.AddInt64DerivedGauge(
		"process/runtime/total_sys_memory_bytes",
		metric.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	if err = pmv.sysMem.UpsertEntry(pmv.updateSysMem); err != nil {
		return err
	}

	pmv.cpuSeconds, err = registry.AddFloat64DerivedCumulative(
		"process/cpu_seconds",
		metric.WithDescription("Total CPU user and system time in seconds"),
		metric.WithUnit(stats.UnitSeconds))
	if err != nil {
		return err
	}
	if err = pmv.cpuSeconds.UpsertEntry(pmv.updateCPUSeconds); err != nil {
		return err
	}

	pmv.rssMemory, err = registry.AddInt64DerivedGauge(
		"process/memory/rss",
		metric.WithDescription("Total physical memory (resident set size)"),
		metric.WithUnit(stats.UnitBytes))
	if err != nil {
		return err
	}
	if err = pmv.rssMemory.UpsertEntry(pmv.updateRSSMemory); err != nil {
		return err
	}

	pmv.proc, err = process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return err
	}

	return nil
}

func (pmv *ProcessMetrics) updateProcessUptime() float64 {
	now := time.Now().UnixNano()
	return float64(now-pmv.startTimeUnixNano) / 1e9
}

func (pmv *ProcessMetrics) updateAllocMem() int64 {
	ms := runtime.MemStats{}
	pmv.readMemStats(&ms)
	return int64(ms.Alloc)
}

func (pmv *ProcessMetrics) updateTotalAllocMem() int64 {
	ms := runtime.MemStats{}
	pmv.readMemStats(&ms)
	return int64(ms.TotalAlloc)
}

func (pmv *ProcessMetrics) updateSysMem() int64 {
	ms := runtime.MemStats{}
	pmv.readMemStats(&ms)
	return int64(ms.Sys)
}

func (pmv *ProcessMetrics) updateCPUSeconds() float64 {
	times, err := pmv.proc.Times()
	if err != nil {
		return 0
	}

	return times.Total()
}

func (pmv *ProcessMetrics) updateRSSMemory() int64 {
	mem, err := pmv.proc.MemoryInfo()
	if err != nil {
		return 0
	}
	return int64(mem.RSS)
}

func (pmv *ProcessMetrics) readMemStats(ms *runtime.MemStats) {
	runtime.ReadMemStats(ms)
	if pmv.ballastSizeBytes > 0 {
		ms.Alloc -= pmv.ballastSizeBytes
		ms.HeapAlloc -= pmv.ballastSizeBytes
		ms.HeapSys -= pmv.ballastSizeBytes
		ms.HeapInuse -= pmv.ballastSizeBytes
	}
}
