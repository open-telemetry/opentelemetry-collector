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

// +build linux darwin freebsd openbsd

package processesscraper

import (
	"time"

	"github.com/shirou/gopsutil/load"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func appendSystemSpecificProcessesMetrics(metrics pdata.MetricSlice, startIndex int, miscFunc getMiscStats) error {
	misc, err := miscFunc()
	if err != nil {
		return err
	}

	metrics.Resize(startIndex + 2)
	initializeProcessesRunningMetric(metrics.At(startIndex+0), misc)
	initializeProcessesBlockedMetric(metrics.At(startIndex+1), misc)
	return nil
}

func initializeProcessesRunningMetric(metric pdata.Metric, misc *load.MiscStat) {
	processesRunningDescriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.Int64DataPoints()
	ddps.Resize(1)
	ddps.At(0).SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	ddps.At(0).SetValue(int64(misc.ProcsRunning))
}

func initializeProcessesBlockedMetric(metric pdata.Metric, misc *load.MiscStat) {
	processesBlockedDescriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.Int64DataPoints()
	ddps.Resize(1)
	ddps.At(0).SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	ddps.At(0).SetValue(int64(misc.ProcsBlocked))
}
