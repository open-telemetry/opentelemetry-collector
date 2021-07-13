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

package diskscraper

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/host"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/perfcounters"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	metricsLen = 5

	logicalDisk = "LogicalDisk"

	readsPerSec  = "Disk Reads/sec"
	writesPerSec = "Disk Writes/sec"

	readBytesPerSec  = "Disk Read Bytes/sec"
	writeBytesPerSec = "Disk Write Bytes/sec"

	idleTime = "% Idle Time"

	avgDiskSecsPerRead  = "Avg. Disk sec/Read"
	avgDiskSecsPerWrite = "Avg. Disk sec/Write"

	queueLength = "Current Disk Queue Length"
)

// scraper for Disk Metrics
type scraper struct {
	config    *Config
	startTime pdata.Timestamp
	includeFS filterset.FilterSet
	excludeFS filterset.FilterSet

	perfCounterScraper perfcounters.PerfCounterScraper

	// for mocking
	bootTime func() (uint64, error)
}

// newDiskScraper creates a Disk Scraper
func newDiskScraper(_ context.Context, cfg *Config) (*scraper, error) {
	scraper := &scraper{config: cfg, perfCounterScraper: &perfcounters.PerfLibScraper{}, bootTime: host.BootTime}

	var err error

	if len(cfg.Include.Devices) > 0 {
		scraper.includeFS, err = filterset.CreateFilterSet(cfg.Include.Devices, &cfg.Include.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating device include filters: %w", err)
		}
	}

	if len(cfg.Exclude.Devices) > 0 {
		scraper.excludeFS, err = filterset.CreateFilterSet(cfg.Exclude.Devices, &cfg.Exclude.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating device exclude filters: %w", err)
		}
	}

	return scraper, nil
}

func (s *scraper) start(context.Context, component.Host) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.Timestamp(bootTime * 1e9)

	return s.perfCounterScraper.Initialize(logicalDisk)
}

func (s *scraper) scrape(ctx context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := pdata.TimestampFromTime(time.Now())

	counters, err := s.perfCounterScraper.Scrape()
	if err != nil {
		return metrics, scrapererror.NewPartialScrapeError(err, metricsLen)
	}

	logicalDiskObject, err := counters.GetObject(logicalDisk)
	if err != nil {
		return metrics, scrapererror.NewPartialScrapeError(err, metricsLen)
	}

	// filter devices by name
	logicalDiskObject.Filter(s.includeFS, s.excludeFS, false)

	logicalDiskCounterValues, err := logicalDiskObject.GetValues(readsPerSec, writesPerSec, readBytesPerSec, writeBytesPerSec, idleTime, avgDiskSecsPerRead, avgDiskSecsPerWrite, queueLength)
	if err != nil {
		return metrics, scrapererror.NewPartialScrapeError(err, metricsLen)
	}

	if len(logicalDiskCounterValues) > 0 {
		metrics.EnsureCapacity(metricsLen)
		initializeDiskIOMetric(metrics.AppendEmpty(), s.startTime, now, logicalDiskCounterValues)
		initializeDiskOperationsMetric(metrics.AppendEmpty(), s.startTime, now, logicalDiskCounterValues)
		initializeDiskIOTimeMetric(metrics.AppendEmpty(), s.startTime, now, logicalDiskCounterValues)
		initializeDiskOperationTimeMetric(metrics.AppendEmpty(), s.startTime, now, logicalDiskCounterValues)
		initializeDiskPendingOperationsMetric(metrics.AppendEmpty(), now, logicalDiskCounterValues)
	}

	return metrics, nil
}

func initializeDiskIOMetric(metric pdata.Metric, startTime, now pdata.Timestamp, logicalDiskCounterValues []*perfcounters.CounterValues) {
	metadata.Metrics.SystemDiskIo.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.EnsureCapacity(2 * len(logicalDiskCounterValues))
	for _, logicalDiskCounter := range logicalDiskCounterValues {
		initializeInt64DataPoint(idps.AppendEmpty(), startTime, now, logicalDiskCounter.InstanceName, metadata.LabelDiskDirection.Read, logicalDiskCounter.Values[readBytesPerSec])
		initializeInt64DataPoint(idps.AppendEmpty(), startTime, now, logicalDiskCounter.InstanceName, metadata.LabelDiskDirection.Write, logicalDiskCounter.Values[writeBytesPerSec])
	}
}

func initializeDiskOperationsMetric(metric pdata.Metric, startTime, now pdata.Timestamp, logicalDiskCounterValues []*perfcounters.CounterValues) {
	metadata.Metrics.SystemDiskOperations.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.EnsureCapacity(2 * len(logicalDiskCounterValues))
	for _, logicalDiskCounter := range logicalDiskCounterValues {
		initializeInt64DataPoint(idps.AppendEmpty(), startTime, now, logicalDiskCounter.InstanceName, metadata.LabelDiskDirection.Read, logicalDiskCounter.Values[readsPerSec])
		initializeInt64DataPoint(idps.AppendEmpty(), startTime, now, logicalDiskCounter.InstanceName, metadata.LabelDiskDirection.Write, logicalDiskCounter.Values[writesPerSec])
	}
}

func initializeDiskIOTimeMetric(metric pdata.Metric, startTime, now pdata.Timestamp, logicalDiskCounterValues []*perfcounters.CounterValues) {
	metadata.Metrics.SystemDiskIoTime.Init(metric)

	ddps := metric.Sum().DataPoints()
	ddps.EnsureCapacity(len(logicalDiskCounterValues))
	for _, logicalDiskCounter := range logicalDiskCounterValues {
		// disk active time = system boot time - disk idle time
		initializeDoubleDataPoint(ddps.AppendEmpty(), startTime, now, logicalDiskCounter.InstanceName, "", float64(now-startTime)/1e9-float64(logicalDiskCounter.Values[idleTime])/1e7)
	}
}

func initializeDiskOperationTimeMetric(metric pdata.Metric, startTime, now pdata.Timestamp, logicalDiskCounterValues []*perfcounters.CounterValues) {
	metadata.Metrics.SystemDiskOperationTime.Init(metric)

	ddps := metric.Sum().DataPoints()
	ddps.EnsureCapacity(2 * len(logicalDiskCounterValues))
	for _, logicalDiskCounter := range logicalDiskCounterValues {
		initializeDoubleDataPoint(ddps.AppendEmpty(), startTime, now, logicalDiskCounter.InstanceName, metadata.LabelDiskDirection.Read, float64(logicalDiskCounter.Values[avgDiskSecsPerRead])/1e7)
		initializeDoubleDataPoint(ddps.AppendEmpty(), startTime, now, logicalDiskCounter.InstanceName, metadata.LabelDiskDirection.Write, float64(logicalDiskCounter.Values[avgDiskSecsPerWrite])/1e7)
	}
}

func initializeDiskPendingOperationsMetric(metric pdata.Metric, now pdata.Timestamp, logicalDiskCounterValues []*perfcounters.CounterValues) {
	metadata.Metrics.SystemDiskPendingOperations.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.EnsureCapacity(len(logicalDiskCounterValues))
	for _, logicalDiskCounter := range logicalDiskCounterValues {
		initializeDiskPendingDataPoint(idps.AppendEmpty(), now, logicalDiskCounter.InstanceName, logicalDiskCounter.Values[queueLength])
	}
}

func initializeInt64DataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.Timestamp, deviceLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	if directionLabel != "" {
		labelsMap.Insert(metadata.Labels.DiskDirection, directionLabel)
	}
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDoubleDataPoint(dataPoint pdata.DoubleDataPoint, startTime, now pdata.Timestamp, deviceLabel string, directionLabel string, value float64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	if directionLabel != "" {
		labelsMap.Insert(metadata.Labels.DiskDirection, directionLabel)
	}
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDiskPendingDataPoint(dataPoint pdata.IntDataPoint, now pdata.Timestamp, deviceLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
