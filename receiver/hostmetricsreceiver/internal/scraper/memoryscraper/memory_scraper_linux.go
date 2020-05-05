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

package memoryscraper

import (
	"github.com/shirou/gopsutil/mem"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

const memStatesLen = 6

func appendMemoryUsedStates(idps pdata.Int64DataPointSlice, memInfo *mem.VirtualMemoryStat) {
	initializeMemoryUsedDataPoint(idps.At(0), usedStateLabelValue, int64(memInfo.Used))
	initializeMemoryUsedDataPoint(idps.At(1), freeStateLabelValue, int64(memInfo.Free))
	initializeMemoryUsedDataPoint(idps.At(2), bufferedStateLabelValue, int64(memInfo.Buffers))
	initializeMemoryUsedDataPoint(idps.At(3), cachedStateLabelValue, int64(memInfo.Cached))
	initializeMemoryUsedDataPoint(idps.At(4), slabReclaimableStateLabelValue, int64(memInfo.SReclaimable))
	initializeMemoryUsedDataPoint(idps.At(5), slabUnreclaimableStateLabelValue, int64(memInfo.SUnreclaim))
}
