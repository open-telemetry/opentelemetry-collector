// Copyright The OpenTelemetry Authors
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
	"go.opentelemetry.io/collector/consumer/pdata"
)

const fileSystemStatesLen = 3

func appendFileSystemUsageStateDataPoints(idps pdata.Int64DataPointSlice, startIdx int, deviceUsage *deviceUsage) {
	initializeFileSystemUsageDataPoint(idps.At(startIdx+0), deviceUsage.deviceName, usedLabelValue, int64(deviceUsage.usage.Used))
	initializeFileSystemUsageDataPoint(idps.At(startIdx+1), deviceUsage.deviceName, freeLabelValue, int64(deviceUsage.usage.Free))
	initializeFileSystemUsageDataPoint(idps.At(startIdx+2), deviceUsage.deviceName, reservedLabelValue, int64(deviceUsage.usage.Total-deviceUsage.usage.Used-deviceUsage.usage.Free))
}

func appendFileSystemUtilizationStateDataPoints(ddps pdata.DoubleDataPointSlice, startIdx int, deviceUsage *deviceUsage) {
	total := float64(deviceUsage.usage.Total)
	initializeFileSystemUtilizationDataPoint(ddps.At(startIdx+0), deviceUsage.deviceName, usedLabelValue, float64(deviceUsage.usage.Used)/total)
	initializeFileSystemUtilizationDataPoint(ddps.At(startIdx+1), deviceUsage.deviceName, freeLabelValue, float64(deviceUsage.usage.Free)/total)
	initializeFileSystemUtilizationDataPoint(ddps.At(startIdx+2), deviceUsage.deviceName, reservedLabelValue, float64(deviceUsage.usage.Total-deviceUsage.usage.Used-deviceUsage.usage.Free)/total)
}

const systemSpecificMetricsLen = 2

func appendSystemSpecificMetrics(metrics pdata.MetricSlice, startIdx int, deviceUsages []*deviceUsage) {
	initializeFileSystemINodesUsageMetric(metrics.At(startIdx+0), deviceUsages)
	initializeFileSystemINodesUtilizationMetric(metrics.At(startIdx+1), deviceUsages)
}

func initializeFileSystemINodesUsageMetric(metric pdata.Metric, deviceUsages []*deviceUsage) {
	fileSystemINodesUsageDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(deviceUsages))

	idx := 0
	for _, deviceUsage := range deviceUsages {
		initializeFileSystemUsageDataPoint(idps.At(idx+0), deviceUsage.deviceName, usedLabelValue, int64(deviceUsage.usage.InodesUsed))
		initializeFileSystemUsageDataPoint(idps.At(idx+1), deviceUsage.deviceName, freeLabelValue, int64(deviceUsage.usage.InodesFree))
		idx += 2
	}
}

func initializeFileSystemINodesUtilizationMetric(metric pdata.Metric, deviceUsages []*deviceUsage) {
	fileSystemINodesUtilizationDescriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.DoubleDataPoints()
	ddps.Resize(2 * len(deviceUsages))

	idx := 0
	for _, deviceUsage := range deviceUsages {
		total := float64(deviceUsage.usage.InodesTotal)
		initializeFileSystemUtilizationDataPoint(ddps.At(idx+0), deviceUsage.deviceName, usedLabelValue, float64(deviceUsage.usage.InodesUsed)/total*100)
		initializeFileSystemUtilizationDataPoint(ddps.At(idx+1), deviceUsage.deviceName, freeLabelValue, float64(deviceUsage.usage.InodesFree)/total*100)
		idx += 2
	}
}
