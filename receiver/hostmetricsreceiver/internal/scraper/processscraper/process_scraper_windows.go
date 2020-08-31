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

// +build windows

package processscraper

import (
	"path/filepath"
	"regexp"

	"github.com/shirou/gopsutil/cpu"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const cpuStatesLen = 2

func appendCPUTimeStateDataPoints(ddps pdata.DoubleDataPointSlice, startTime, now pdata.TimestampUnixNano, cpuTime *cpu.TimesStat) {
	initializeCPUTimeDataPoint(ddps.At(0), startTime, now, cpuTime.User, userStateLabelValue)
	initializeCPUTimeDataPoint(ddps.At(1), startTime, now, cpuTime.System, systemStateLabelValue)
}

func initializeCPUTimeDataPoint(dataPoint pdata.DoubleDataPoint, startTime, now pdata.TimestampUnixNano, value float64, stateLabel string) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func getProcessExecutable(proc processHandle) (*executableMetadata, error) {
	exe, err := proc.Exe()
	if err != nil {
		return nil, err
	}

	name := filepath.Base(exe)
	executable := &executableMetadata{name: name, path: exe}
	return executable, nil
}

// matches the first argument before an unquoted space or slash
var cmdRegex = regexp.MustCompile(`^((?:[^"]*?"[^"]*?")*?[^"]*?)(?:[ \/]|$)`)

func getProcessCommand(proc processHandle) (*commandMetadata, error) {
	cmdline, err := proc.Cmdline()
	if err != nil {
		return nil, err
	}

	cmd := cmdline
	match := cmdRegex.FindStringSubmatch(cmdline)
	if match != nil {
		cmd = match[1]
	}

	command := &commandMetadata{command: cmd, commandLine: cmdline}
	return command, nil
}
