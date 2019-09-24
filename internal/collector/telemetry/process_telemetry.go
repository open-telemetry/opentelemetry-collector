// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/prometheus/procfs"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

// ProcessMetricsViews is a struct that contains views related to process metrics (cpu, mem, etc)
type ProcessMetricsViews struct {
	ballastSizeBytes uint64
	views            []*view.View
	done             chan struct{}
}

var mRuntimeAllocMem = stats.Int64("oc.io/process/memory_alloc", "Number of bytes currently allocated in use", "By")
var viewAllocMem = &view.View{
	Name:        mRuntimeAllocMem.Name(),
	Description: mRuntimeAllocMem.Description(),
	Measure:     mRuntimeAllocMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mRuntimeTotalAllocMem = stats.Int64("oc.io/process/total_memory_alloc", "Number of allocations in total", "By")
var viewTotalAllocMem = &view.View{
	Name:        mRuntimeTotalAllocMem.Name(),
	Description: mRuntimeTotalAllocMem.Description(),
	Measure:     mRuntimeTotalAllocMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mRuntimeSysMem = stats.Int64("oc.io/process/sys_memory_alloc", "Number of bytes given to the process to use in total", "By")
var viewSysMem = &view.View{
	Name:        mRuntimeSysMem.Name(),
	Description: mRuntimeSysMem.Description(),
	Measure:     mRuntimeSysMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mCPUSeconds = stats.Int64("oc.io/process/cpu_seconds", "CPU seconds for this process", "1")
var viewCPUSeconds = &view.View{
	Name:        mCPUSeconds.Name(),
	Description: mCPUSeconds.Description(),
	Measure:     mCPUSeconds,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

// NewProcessMetricsViews creates a new set of ProcessMetrics (mem, cpu) that can be used to measure
// basic information about this process.
func NewProcessMetricsViews(ballastSizeBytes uint64) *ProcessMetricsViews {
	return &ProcessMetricsViews{
		ballastSizeBytes: ballastSizeBytes,
		views:            []*view.View{viewAllocMem, viewTotalAllocMem, viewSysMem, viewCPUSeconds},
		done:             make(chan struct{}),
	}
}

// StartCollection starts a ticker'd goroutine that will update the PMV measurements every 5 seconds
func (pmv *ProcessMetricsViews) StartCollection() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
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

// Views returns the views internal to the PMV
func (pmv *ProcessMetricsViews) Views() []*view.View {
	return pmv.views
}

// StopCollection stops the collection of the process metric information
func (pmv *ProcessMetricsViews) StopCollection() {
	close(pmv.done)
}

func (pmv *ProcessMetricsViews) updateViews() {
	ms := &runtime.MemStats{}
	pmv.readMemStats(ms)
	stats.Record(context.Background(), mRuntimeAllocMem.M(int64(ms.Alloc)))
	stats.Record(context.Background(), mRuntimeTotalAllocMem.M(int64(ms.TotalAlloc)))
	stats.Record(context.Background(), mRuntimeSysMem.M(int64(ms.Sys)))

	pid := os.Getpid()
	proc, err := procfs.NewProc(pid)
	if err == nil {
		if procStat, err := proc.Stat(); err == nil {
			stats.Record(context.Background(), mCPUSeconds.M(int64(procStat.CPUTime())))
		}
	}
}

func (pmv *ProcessMetricsViews) readMemStats(ms *runtime.MemStats) {
	runtime.ReadMemStats(ms)
	ms.Alloc -= pmv.ballastSizeBytes
	ms.HeapAlloc -= pmv.ballastSizeBytes
	ms.HeapSys -= pmv.ballastSizeBytes
	ms.HeapInuse -= pmv.ballastSizeBytes
}
