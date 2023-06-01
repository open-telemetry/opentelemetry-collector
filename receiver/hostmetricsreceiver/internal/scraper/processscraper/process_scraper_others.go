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

//go:build !linux && !windows
// +build !linux,!windows

package processscraper

import (
	"github.com/shirou/gopsutil/cpu"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const cpuStatesLen = 1

func appendCPUTimeStateDataPoints(ddps pdata.DoubleDataPointSlice, startTime, now pdata.Timestamp, cpuTime *cpu.TimesStat, processName string) {
}

func appendCPUPercentDataPoints(ddps pdata.DoubleDataPointSlice, startTime, now pdata.Timestamp, value float64, processName string) {
	dataPoint := ddps.At(0)
	labelsMap := dataPoint.LabelsMap()
	if len(processName) > 0 {
		labelsMap.Insert(metadata.Labels.ProcessName, processName)
	}
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func getProcessExecutable(processHandle) (*executableMetadata, error) {
	return nil, nil
}

func getProcessCommand(processHandle) (*commandMetadata, error) {
	return nil, nil
}
