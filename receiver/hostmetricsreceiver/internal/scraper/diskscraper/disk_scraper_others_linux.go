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

package diskscraper

import (
	"github.com/shirou/gopsutil/disk"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const systemSpecificMetricsLen = 1

func appendSystemSpecificMetrics(metrics pdata.MetricSlice, startIdx int, startTime, now pdata.TimestampUnixNano, ioCounters map[string]disk.IOCountersStat) {
	metric := metrics.At(startIdx)
	diskMergedDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeInt64DataPoint(idps.At(idx+0), startTime, now, device, readDirectionLabelValue, int64(ioCounter.MergedReadCount))
		initializeInt64DataPoint(idps.At(idx+1), startTime, now, device, writeDirectionLabelValue, int64(ioCounter.MergedWriteCount))
		idx += 2
	}
}
