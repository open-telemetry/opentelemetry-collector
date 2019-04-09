// Copyright 2019, OpenCensus Authors
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

package vmmetricsreceiver

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

// Measure and view constants for VM and process metrics.
// TODO(songy23): remove all measures and views and use Metrics APIs instead.

var mRuntimeAllocMem = stats.Int64("process/memory_alloc", "Number of bytes currently allocated in use", "By")
var viewAllocMem = &view.View{
	Name:        mRuntimeAllocMem.Name(),
	Description: mRuntimeAllocMem.Description(),
	Measure:     mRuntimeAllocMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mRuntimeTotalAllocMem = stats.Int64("process/total_memory_alloc", "Number of allocations in total", "By")
var viewTotalAllocMem = &view.View{
	Name:        mRuntimeTotalAllocMem.Name(),
	Description: mRuntimeTotalAllocMem.Description(),
	Measure:     mRuntimeTotalAllocMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mRuntimeSysMem = stats.Int64("process/sys_memory_alloc", "Number of bytes given to the process to use in total", "By")
var viewSysMem = &view.View{
	Name:        mRuntimeSysMem.Name(),
	Description: mRuntimeSysMem.Description(),
	Measure:     mRuntimeSysMem,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mCPUSeconds = stats.Float64("process/cpu_seconds", "CPU seconds for this process", "s")
var viewCPUSeconds = &view.View{
	Name:        mCPUSeconds.Name(),
	Description: mCPUSeconds.Description(),
	Measure:     mCPUSeconds,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mUserCPUSeconds = stats.Float64("system/cpu_seconds/user", "Total kernel/system user CPU seconds", "s")
var viewUserCPUSeconds = &view.View{
	Name:        mUserCPUSeconds.Name(),
	Description: mUserCPUSeconds.Description(),
	Measure:     mUserCPUSeconds,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mNiceCPUSeconds = stats.Float64("system/cpu_seconds/nice", "Total kernel/system nice CPU seconds", "s")
var viewNiceCPUSeconds = &view.View{
	Name:        mNiceCPUSeconds.Name(),
	Description: mNiceCPUSeconds.Description(),
	Measure:     mNiceCPUSeconds,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mSystemCPUSeconds = stats.Float64("system/cpu_seconds/system", "Total kernel/system system CPU seconds", "s")
var viewSystemCPUSeconds = &view.View{
	Name:        mSystemCPUSeconds.Name(),
	Description: mSystemCPUSeconds.Description(),
	Measure:     mSystemCPUSeconds,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mIdleCPUSeconds = stats.Float64("system/cpu_seconds/idle", "Total kernel/system idle CPU seconds", "s")
var viewIdleCPUSeconds = &view.View{
	Name:        mIdleCPUSeconds.Name(),
	Description: mIdleCPUSeconds.Description(),
	Measure:     mIdleCPUSeconds,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mIowaitCPUSeconds = stats.Float64("system/cpu_seconds/iowait", "Total kernel/system Iowait CPU seconds", "s")
var viewIowaitCPUSeconds = &view.View{
	Name:        mIowaitCPUSeconds.Name(),
	Description: mIowaitCPUSeconds.Description(),
	Measure:     mIowaitCPUSeconds,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mProcessesCreated = stats.Int64("system/processes/created", "Total number of times a process was created", "1")
var viewProcessesCreated = &view.View{
	Name:        mProcessesCreated.Name(),
	Description: mProcessesCreated.Description(),
	Measure:     mProcessesCreated,
	Aggregation: view.Sum(),
	TagKeys:     nil,
}

var mProcessesRunning = stats.Int64("system/processes/running", "Total number of running processes", "1")
var viewProcessesRunning = &view.View{
	Name:        mProcessesRunning.Name(),
	Description: mProcessesRunning.Description(),
	Measure:     mProcessesRunning,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var mProcessesBlocked = stats.Int64("system/processes/blocked", "Total number of blocked processes", "1")
var viewProcessesBlocked = &view.View{
	Name:        mProcessesBlocked.Name(),
	Description: mProcessesBlocked.Description(),
	Measure:     mProcessesBlocked,
	Aggregation: view.LastValue(),
	TagKeys:     nil,
}

var vmViews = []*view.View{
	viewAllocMem,
	viewTotalAllocMem,
	viewSysMem,
	viewCPUSeconds,
	viewUserCPUSeconds,
	viewNiceCPUSeconds,
	viewSystemCPUSeconds,
	viewIdleCPUSeconds,
	viewIowaitCPUSeconds,
	viewProcessesCreated,
	viewProcessesRunning,
	viewProcessesBlocked}
