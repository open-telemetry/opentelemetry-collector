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

package telemetry

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/process"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

// ProcessMetricsViews is a struct that contains views related to process metrics (cpu, mem, etc)
type ProcessMetricsViews struct {
	prevTimeUnixNano int64
	ballastSizeBytes uint64
	views            []*view.View
	done             chan struct{}
	proc             *process.Process
}

var mUptime = stats.Float64(
	"process/uptime",
	"Uptime of the process",
	stats.UnitSeconds)
var viewProcessUptime = &view.View{
	Name:        mUptime.Name(),
	Description: mUptime.Description(),
	Measure:     mUptime,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mRuntimeAllocMem = stats.Int64(
	"process/runtime/heap_alloc_bytes",
	"Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')",
	stats.UnitBytes)
var viewAllocMem = &view.View{
	Name:        mRuntimeAllocMem.Name(),
	Description: mRuntimeAllocMem.Description(),
	Measure:     mRuntimeAllocMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mRuntimeTotalAllocMem = stats.Int64(
	"process/runtime/total_alloc_bytes",
	"Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')",
	stats.UnitBytes)
var viewTotalAllocMem = &view.View{
	Name:        mRuntimeTotalAllocMem.Name(),
	Description: mRuntimeTotalAllocMem.Description(),
	Measure:     mRuntimeTotalAllocMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mRuntimeSysMem = stats.Int64(
	"process/runtime/total_sys_memory_bytes",
	"Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')",
	stats.UnitBytes)
var viewSysMem = &view.View{
	Name:        mRuntimeSysMem.Name(),
	Description: mRuntimeSysMem.Description(),
	Measure:     mRuntimeSysMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mCPUSeconds = stats.Float64(
	"process/cpu_seconds",
	"Total CPU user and system time in seconds",
	stats.UnitSeconds)
var viewCPUSeconds = &view.View{
	Name:        mCPUSeconds.Name(),
	Description: mCPUSeconds.Description(),
	Measure:     mCPUSeconds,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mRSSMemory = stats.Int64(
	"process/memory/rss",
	"Total physical memory (resident set size)",
	stats.UnitBytes)
var viewRSSMemory = &view.View{
	Name:        mRSSMemory.Name(),
	Description: mRSSMemory.Description(),
	Measure:     mRSSMemory,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

// NewProcessMetricsViews creates a new set of ProcessMetrics (mem, cpu) that can be used to measure
// basic information about this process.
func NewProcessMetricsViews(ballastSizeBytes uint64) (*ProcessMetricsViews, error) {
	pmv := &ProcessMetricsViews{
		prevTimeUnixNano: time.Now().UnixNano(),
		views:            []*view.View{viewProcessUptime, viewAllocMem, viewTotalAllocMem, viewSysMem, viewCPUSeconds, viewRSSMemory},
		ballastSizeBytes: ballastSizeBytes,
		done:             make(chan struct{}),
	}

	pid := os.Getpid()

	var err error
	pmv.proc, err = process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}

	return pmv, nil
}

// StartCollection starts a ticker'd goroutine that will update the PMV measurements every 5 seconds
func (pmv *ProcessMetricsViews) StartCollection() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pmv.updateViews()
			case <-pmv.done:
				return
			}
		}
	}()
}

// Views returns the views internal to the PMV.
func (pmv *ProcessMetricsViews) Views() []*view.View {
	return pmv.views
}

// StopCollection stops the collection of the process metric information.
func (pmv *ProcessMetricsViews) StopCollection() {
	close(pmv.done)
}

func (pmv *ProcessMetricsViews) updateViews() {
	now := time.Now().UnixNano()
	stats.Record(context.Background(), mUptime.M(float64(now-pmv.prevTimeUnixNano)/1e9))
	pmv.prevTimeUnixNano = now

	ms := &runtime.MemStats{}
	pmv.readMemStats(ms)
	stats.Record(context.Background(), mRuntimeAllocMem.M(int64(ms.Alloc)))
	stats.Record(context.Background(), mRuntimeTotalAllocMem.M(int64(ms.TotalAlloc)))
	stats.Record(context.Background(), mRuntimeSysMem.M(int64(ms.Sys)))

	if pmv.proc != nil {
		if times, err := pmv.proc.Times(); err == nil {
			stats.Record(context.Background(), mCPUSeconds.M(times.Total()))
		}
		if mem, err := pmv.proc.MemoryInfo(); err == nil {
			stats.Record(context.Background(), mRSSMemory.M(int64(mem.RSS)))
		}
	}
}

func (pmv *ProcessMetricsViews) readMemStats(ms *runtime.MemStats) {
	runtime.ReadMemStats(ms)
	if pmv.ballastSizeBytes > 0 {
		ms.Alloc -= pmv.ballastSizeBytes
		ms.HeapAlloc -= pmv.ballastSizeBytes
		ms.HeapSys -= pmv.ballastSizeBytes
		ms.HeapInuse -= pmv.ballastSizeBytes
	}
}
