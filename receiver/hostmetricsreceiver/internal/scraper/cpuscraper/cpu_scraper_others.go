// Copyright The OpenTelemetry Authors
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

// +build !linux

package cpuscraper

import (
	"github.com/shirou/gopsutil/cpu"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const cpuStatesLen = 4

func appendCPUTimeStateDataPoints(ddps pdata.DoubleDataPointSlice, startIdx int, startTime pdata.TimestampUnixNano, cpuTime cpu.TimesStat) {
	initializeCPUDataPoint(ddps.At(startIdx+0), startTime, cpuTime.CPU, userStateLabelValue, cpuTime.User)
	initializeCPUDataPoint(ddps.At(startIdx+1), startTime, cpuTime.CPU, systemStateLabelValue, cpuTime.System)
	initializeCPUDataPoint(ddps.At(startIdx+2), startTime, cpuTime.CPU, idleStateLabelValue, cpuTime.Idle)
	initializeCPUDataPoint(ddps.At(startIdx+3), startTime, cpuTime.CPU, interruptStateLabelValue, cpuTime.Irq)
}

func appendCPUUtilizationStateDataPoints(ddps pdata.DoubleDataPointSlice, startIdx int, startTime pdata.TimestampUnixNano, cpuTime cpu.TimesStat, prevCPUTime cpu.TimesStat) {
	total := cpuTime.Total() - prevCPUTime.Total()
	initializeCPUDataPoint(ddps.At(startIdx+0), startTime, cpuTime.CPU, userStateLabelValue, (cpuTime.User-prevCPUTime.User)/total*100)
	initializeCPUDataPoint(ddps.At(startIdx+1), startTime, cpuTime.CPU, systemStateLabelValue, (cpuTime.System-prevCPUTime.System)/total*100)
	initializeCPUDataPoint(ddps.At(startIdx+2), startTime, cpuTime.CPU, idleStateLabelValue, (cpuTime.Idle-prevCPUTime.Idle)/total*100)
	initializeCPUDataPoint(ddps.At(startIdx+3), startTime, cpuTime.CPU, interruptStateLabelValue, (cpuTime.Irq-prevCPUTime.Irq)/total*100)
}
