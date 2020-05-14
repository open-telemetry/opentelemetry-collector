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

package processscraper

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
)

type processHandle interface {
	GetPid() int32
	Name() (string, error)
	Exe() (string, error)
	Username() (string, error)
	Cmdline() (string, error)
	Times() (*cpu.TimesStat, error)
	MemoryInfo() (*process.MemoryInfoStat, error)
	IOCounters() (*process.IOCountersStat, error)
}

type gopsProcessHandle struct {
	*process.Process
}

func (p *gopsProcessHandle) GetPid() int32 {
	return p.Pid
}

func getProcessHandlesInternal() ([]processHandle, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	processAccessors := make([]processHandle, len(processes))
	for i, proc := range processes {
		processAccessors[i] = &gopsProcessHandle{proc}
	}
	return processAccessors, nil
}
