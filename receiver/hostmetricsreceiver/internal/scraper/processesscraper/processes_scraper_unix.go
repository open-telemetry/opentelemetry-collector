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

// +build linux darwin freebsd openbsd

package processesscraper

import (
	"time"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

const systemSpecificMetricsLen = 2

func appendSystemSpecificProcessesMetrics(metrics pdata.MetricSlice, startIndex int, miscFunc getMiscStats) error {
	now := internal.TimeToUnixNano(time.Now())
	misc, err := miscFunc()
	if err != nil {
		return consumererror.NewPartialScrapeError(err, systemSpecificMetricsLen)
	}

	metrics.Resize(startIndex + systemSpecificMetricsLen)
	initializeProcessesMetric(metrics.At(startIndex+0), processesRunningDescriptor, now, int64(misc.ProcsRunning))
	initializeProcessesMetric(metrics.At(startIndex+1), processesBlockedDescriptor, now, int64(misc.ProcsBlocked))
	return nil
}

func initializeProcessesMetric(metric pdata.Metric, descriptor pdata.Metric, now pdata.TimestampUnixNano, value int64) {
	descriptor.CopyTo(metric)

	ddps := metric.IntSum().DataPoints()
	ddps.Resize(1)
	ddps.At(0).SetTimestamp(now)
	ddps.At(0).SetValue(value)
}
