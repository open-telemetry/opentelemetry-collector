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

	"github.com/shirou/gopsutil/load"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	standardUnixMetricsLen = 1
	unixMetricsLen         = standardUnixMetricsLen + unixSystemSpecificMetricsLen
)

func appendSystemSpecificProcessesMetrics(metrics pdata.MetricSlice, startTime pdata.Timestamp, miscFunc getMiscStats) error {
	now := pdata.TimestampFromTime(time.Now())
	misc, err := miscFunc()
	if err != nil {
		return scrapererror.NewPartialScrapeError(err, unixMetricsLen)
	}

	metrics.EnsureCapacity(unixMetricsLen)
	initializeProcessesCountMetric(metrics.AppendEmpty(), startTime, now, misc)
	return appendUnixSystemSpecificProcessesMetrics(metrics, startTime, now, misc)
}

func initializeProcessesCountMetric(metric pdata.Metric, startTime pdata.Timestamp, now pdata.Timestamp, misc *load.MiscStat) {
	metadata.Metrics.SystemProcessesCount.Init(metric)

	ddps := metric.IntSum().DataPoints()
	ddps.EnsureCapacity(2)
	initializeProcessesCountDataPoint(ddps.AppendEmpty(), startTime, now, metadata.LabelProcessesStatus.Running, int64(misc.ProcsRunning))
	initializeProcessesCountDataPoint(ddps.AppendEmpty(), startTime, now, metadata.LabelProcessesStatus.Blocked, int64(misc.ProcsBlocked))
}

func initializeProcessesCountDataPoint(dataPoint pdata.IntDataPoint, startTime pdata.Timestamp, now pdata.Timestamp, statusLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.ProcessesStatus, statusLabel)
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
