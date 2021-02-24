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
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const systemSpecificMetricsLen = 2

func appendSystemSpecificMetrics(metrics pdata.MetricSlice, startIdx int, startTime, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	initializeDiskWeightedIOTimeMetric(metrics.At(startIdx+0), startTime, now, ioCounters)
	initializeDiskMergedMetric(metrics.At(startIdx+1), startTime, now, ioCounters)
}

func initializeDiskWeightedIOTimeMetric(metric pdata.Metric, startTime, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	metadata.Metrics.SystemDiskWeightedIoTime.Init(metric)

	ddps := metric.DoubleSum().DataPoints()
	ddps.Resize(len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDoubleDataPoint(ddps.At(idx+0), startTime, now, device, "", float64(ioCounter.WeightedIO)/1e3)
		idx++
	}
}

func initializeDiskMergedMetric(metric pdata.Metric, startTime, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	metadata.Metrics.SystemDiskMerged.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeInt64DataPoint(idps.At(idx+0), startTime, now, device, metadata.LabelDiskDirection.Read, int64(ioCounter.MergedReadCount))
		initializeInt64DataPoint(idps.At(idx+1), startTime, now, device, metadata.LabelDiskDirection.Write, int64(ioCounter.MergedWriteCount))
		idx += 2
	}
}
