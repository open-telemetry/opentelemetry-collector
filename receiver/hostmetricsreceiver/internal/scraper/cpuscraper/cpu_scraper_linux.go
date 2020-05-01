// Copyright 2020, OpenTelemetry Authors
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

// +build linux

package cpuscraper

import (
	"github.com/shirou/gopsutil/cpu"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

const cpuStatesLen = 8

func appendCPUStateTimes(idps pdata.Int64DataPointSlice, startIdx int, startTime pdata.TimestampUnixNano, cpuTime cpu.TimesStat) {
	initializeCPUSecondsDataPoint(idps.At(startIdx+0), startTime, cpuTime.CPU, UserStateLabelValue, int64(cpuTime.User))
	initializeCPUSecondsDataPoint(idps.At(startIdx+1), startTime, cpuTime.CPU, SystemStateLabelValue, int64(cpuTime.System))
	initializeCPUSecondsDataPoint(idps.At(startIdx+2), startTime, cpuTime.CPU, IdleStateLabelValue, int64(cpuTime.Idle))
	initializeCPUSecondsDataPoint(idps.At(startIdx+3), startTime, cpuTime.CPU, InterruptStateLabelValue, int64(cpuTime.Irq))
	initializeCPUSecondsDataPoint(idps.At(startIdx+4), startTime, cpuTime.CPU, NiceStateLabelValue, int64(cpuTime.Nice))
	initializeCPUSecondsDataPoint(idps.At(startIdx+5), startTime, cpuTime.CPU, SoftIRQStateLabelValue, int64(cpuTime.Softirq))
	initializeCPUSecondsDataPoint(idps.At(startIdx+6), startTime, cpuTime.CPU, StealStateLabelValue, int64(cpuTime.Steal))
	initializeCPUSecondsDataPoint(idps.At(startIdx+7), startTime, cpuTime.CPU, WaitStateLabelValue, int64(cpuTime.Iowait))
}
