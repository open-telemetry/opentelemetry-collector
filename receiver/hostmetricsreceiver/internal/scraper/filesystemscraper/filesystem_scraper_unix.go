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

// +build linux darwin freebsd openbsd solaris

package filesystemscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

const fileSystemStatesLen = 3

func appendFileSystemUsageStateDataPoints(idps pdata.IntDataPointSlice, startIdx int, now pdata.TimestampUnixNano, deviceUsage *deviceUsage) {
	initializeFileSystemUsageDataPoint(idps.At(startIdx+0), now, deviceUsage.partition, usedLabelValue, int64(deviceUsage.usage.Used))
	initializeFileSystemUsageDataPoint(idps.At(startIdx+1), now, deviceUsage.partition, freeLabelValue, int64(deviceUsage.usage.Free))
	initializeFileSystemUsageDataPoint(idps.At(startIdx+2), now, deviceUsage.partition, reservedLabelValue, int64(deviceUsage.usage.Total-deviceUsage.usage.Used-deviceUsage.usage.Free))
}

const systemSpecificMetricsLen = 1

func appendSystemSpecificMetrics(metrics pdata.MetricSlice, startIdx int, now pdata.TimestampUnixNano, deviceUsages []*deviceUsage) {
	metric := metrics.At(startIdx)
	fileSystemINodesUsageDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(deviceUsages))
	for idx, deviceUsage := range deviceUsages {
		startIndex := 2 * idx
		initializeFileSystemUsageDataPoint(idps.At(startIndex+0), now, deviceUsage.partition, usedLabelValue, int64(deviceUsage.usage.InodesUsed))
		initializeFileSystemUsageDataPoint(idps.At(startIndex+1), now, deviceUsage.partition, freeLabelValue, int64(deviceUsage.usage.InodesFree))
	}
}
