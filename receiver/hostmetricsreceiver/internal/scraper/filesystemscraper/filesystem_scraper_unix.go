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

// +build linux darwin freebsd openbsd solaris

package filesystemscraper

import (
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

const fileSystemStatesLen = 3

func appendFileSystemUsedStateDataPoints(idps pdata.Int64DataPointSlice, startIdx int, deviceUsage *deviceUsage) {
	initializeFileSystemUsedDataPoint(idps.At(startIdx+0), deviceUsage.deviceName, usedLabelValue, int64(deviceUsage.usage.Used))
	initializeFileSystemUsedDataPoint(idps.At(startIdx+1), deviceUsage.deviceName, freeLabelValue, int64(deviceUsage.usage.Free))
	initializeFileSystemUsedDataPoint(idps.At(startIdx+2), deviceUsage.deviceName, reservedLabelValue, int64(deviceUsage.usage.Total-deviceUsage.usage.Used-deviceUsage.usage.Free))
}

const systemSpecificMetricsLen = 1

func appendSystemSpecificMetrics(metrics pdata.MetricSlice, startIdx int, deviceUsages []*deviceUsage) {
	initializeMetricFileSystemINodesUsed(metrics.At(startIdx), deviceUsages)
}

func initializeMetricFileSystemINodesUsed(metric pdata.Metric, deviceUsages []*deviceUsage) {
	metricFilesystemINodesUsedDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(deviceUsages))
	for idx, deviceUsage := range deviceUsages {
		startIndex := 2 * idx
		initializeFileSystemUsedDataPoint(idps.At(startIndex+0), deviceUsage.deviceName, usedLabelValue, int64(deviceUsage.usage.InodesUsed))
		initializeFileSystemUsedDataPoint(idps.At(startIndex+1), deviceUsage.deviceName, freeLabelValue, int64(deviceUsage.usage.InodesFree))
	}
}
