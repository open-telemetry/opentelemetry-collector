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

// +build linux

package memoryscraper

import (
	"github.com/shirou/gopsutil/mem"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const memStatesLen = 6

func appendMemoryUsageStateDataPoints(idps pdata.IntDataPointSlice, now pdata.Timestamp, memInfo *mem.VirtualMemoryStat) {
	initializeMemoryUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelMemState.Used, int64(memInfo.Used))
	initializeMemoryUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelMemState.Free, int64(memInfo.Free))
	initializeMemoryUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelMemState.Buffered, int64(memInfo.Buffers))
	initializeMemoryUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelMemState.Cached, int64(memInfo.Cached))
	initializeMemoryUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelMemState.SlabReclaimable, int64(memInfo.SReclaimable))
	initializeMemoryUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelMemState.SlabUnreclaimable, int64(memInfo.SUnreclaim))
}
