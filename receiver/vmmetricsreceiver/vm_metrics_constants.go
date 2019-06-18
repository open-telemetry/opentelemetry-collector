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

package vmmetricsreceiver

import (
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
)

// VM and process metric constants.

var metricAllocMem = &metricspb.MetricDescriptor{
	Name:        "process/memory_alloc",
	Description: "Number of bytes currently allocated in use",
	Unit:        "By",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
	LabelKeys:   nil,
}

var metricTotalAllocMem = &metricspb.MetricDescriptor{
	Name:        "process/total_memory_alloc",
	Description: "Number of allocations in total",
	Unit:        "By",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
	LabelKeys:   nil,
}

var metricSysMem = &metricspb.MetricDescriptor{
	Name:        "process/sys_memory_alloc",
	Description: "Number of bytes given to the process to use in total",
	Unit:        "By",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
	LabelKeys:   nil,
}

var metricProcessCPUSeconds = &metricspb.MetricDescriptor{
	Name:        "process/cpu_seconds",
	Description: "CPU seconds for this process",
	Unit:        "s",
	Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
	LabelKeys:   nil,
}

var metricCPUSeconds = &metricspb.MetricDescriptor{
	Name:        "system/cpu_seconds",
	Description: "Total kernel/system CPU seconds broken down by different states",
	Unit:        "s",
	Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
	LabelKeys:   []*metricspb.LabelKey{{Key: "state", Description: "State of CPU time, e.g user/system/idle"}},
}

var metricProcessesCreated = &metricspb.MetricDescriptor{
	Name:        "system/processes/created",
	Description: "Total number of times a process was created",
	Unit:        "1",
	Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
	LabelKeys:   nil,
}

var metricProcessesRunning = &metricspb.MetricDescriptor{
	Name:        "system/processes/running",
	Description: "Total number of running processes",
	Unit:        "1",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
	LabelKeys:   nil,
}

var metricProcessesBlocked = &metricspb.MetricDescriptor{
	Name:        "system/processes/blocked",
	Description: "Total number of blocked processes",
	Unit:        "1",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
	LabelKeys:   nil,
}

var (
	labelValueCPUUser   = &metricspb.LabelValue{Value: "user", HasValue: true}
	labelValueCPUSystem = &metricspb.LabelValue{Value: "system", HasValue: true}
	labelValueCPUIdle   = &metricspb.LabelValue{Value: "idle", HasValue: true}
	labelValueCPUNice   = &metricspb.LabelValue{Value: "nice", HasValue: true}
	labelValueCPUIOWait = &metricspb.LabelValue{Value: "iowait", HasValue: true}
)

var vmMetricDescriptors = []*metricspb.MetricDescriptor{
	metricAllocMem,
	metricTotalAllocMem,
	metricSysMem,
	metricProcessCPUSeconds,
	metricCPUSeconds,
	metricProcessesCreated,
	metricProcessesRunning,
	metricProcessesBlocked,
}
