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

// +build !linux,!darwin,!freebsd,!openbsd,!solaris

package filesystemscraper

import (
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const fileSystemStatesLen = 2

func appendFileSystemUsageStateDataPoints(idps pdata.IntDataPointSlice, now pdata.Timestamp, deviceUsage *deviceUsage) {
	initializeFileSystemUsageDataPoint(idps.AppendEmpty(), now, deviceUsage.partition, metadata.LabelFilesystemState.Used, int64(deviceUsage.usage.Used))
	initializeFileSystemUsageDataPoint(idps.AppendEmpty(), now, deviceUsage.partition, metadata.LabelFilesystemState.Free, int64(deviceUsage.usage.Free))
}

const systemSpecificMetricsLen = 0

func appendSystemSpecificMetrics(metrics pdata.MetricSlice, now pdata.Timestamp, deviceUsages []*deviceUsage) {
}
