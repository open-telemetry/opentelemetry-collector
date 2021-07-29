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

// +build !windows

package processesscraper

import (
	"runtime"

	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/processesscraper/internal/metadata"
)

const enableProcessesCount = true
const enableProcessesCreated = runtime.GOOS == "openbsd" || runtime.GOOS == "linux"

func (s *scraper) getProcessesMetadata() (processesMetadata, error) {
	processes, err := s.getProcesses()
	if err != nil {
		return processesMetadata{}, err
	}

	countByStatus := map[string]int64{}
	for _, process := range processes {
		var status string
		status, err = process.Status()
		if err != nil {
			// We expect an error in the case that a process has
			// been terminated as we run this code.
			continue
		}
		state, ok := charToState[status]
		if !ok {
			countByStatus[metadata.LabelStatus.Unknown]++
			continue
		}
		countByStatus[state]++
	}

	// Processes are actively changing as we run this code, so this reason
	// the above loop will tend to underestimate process counts.
	// getMiscStats is a single read/syscall so it should be more accurate.
	miscStat, err := s.getMiscStats()
	if err != nil {
		return processesMetadata{}, err
	}

	var procsCreated *int64
	if enableProcessesCreated {
		v := int64(miscStat.ProcsCreated)
		procsCreated = &v
	}

	countByStatus[metadata.LabelStatus.Blocked] = int64(miscStat.ProcsBlocked)
	countByStatus[metadata.LabelStatus.Running] = int64(miscStat.ProcsRunning)

	totalKnown := int64(0)
	for _, count := range countByStatus {
		totalKnown += count
	}
	if int64(miscStat.ProcsTotal) > totalKnown {
		countByStatus[metadata.LabelStatus.Unknown] = int64(miscStat.ProcsTotal) - totalKnown
	}

	return processesMetadata{
		countByStatus:    countByStatus,
		processesCreated: procsCreated,
	}, nil
}

var charToState = map[string]string{
	"A": metadata.LabelStatus.Daemon,
	"D": metadata.LabelStatus.Blocked,
	"E": metadata.LabelStatus.Detached,
	"O": metadata.LabelStatus.Orphan,
	"R": metadata.LabelStatus.Running,
	"S": metadata.LabelStatus.Sleeping,
	"T": metadata.LabelStatus.Stopped,
	"W": metadata.LabelStatus.Paging,
	"Y": metadata.LabelStatus.System,
	"Z": metadata.LabelStatus.Zombies,
}
