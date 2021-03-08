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

// +build !windows

package diskscraper

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	standardMetricsLen = 5
	metricsLen         = standardMetricsLen + systemSpecificMetricsLen
)

// scraper for Disk Metrics
type scraper struct {
	config    *Config
	startTime pdata.Timestamp
	includeFS filterset.FilterSet
	excludeFS filterset.FilterSet

	// for mocking
	bootTime   func() (uint64, error)
	ioCounters func(names ...string) (map[string]disk.IOCountersStat, error)
}

// newDiskScraper creates a Disk Scraper
func newDiskScraper(_ context.Context, cfg *Config) (*scraper, error) {
	scraper := &scraper{config: cfg, bootTime: host.BootTime, ioCounters: disk.IOCounters}

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
	return nil
}

func (s *scraper) scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := pdata.TimestampFromTime(time.Now())
	ioCounters, err := s.ioCounters()
	if err != nil {
		return metrics, scrapererror.NewPartialScrapeError(err, metricsLen)
	}

	// filter devices by name
	ioCounters = s.filterByDevice(ioCounters)

	if len(ioCounters) > 0 {
		metrics.Resize(metricsLen)
		initializeDiskIOMetric(metrics.At(0), s.startTime, now, ioCounters)
		initializeDiskOperationsMetric(metrics.At(1), s.startTime, now, ioCounters)
		initializeDiskIOTimeMetric(metrics.At(2), s.startTime, now, ioCounters)
		initializeDiskOperationTimeMetric(metrics.At(3), s.startTime, now, ioCounters)
		initializeDiskPendingOperationsMetric(metrics.At(4), now, ioCounters)
		appendSystemSpecificMetrics(metrics, 5, s.startTime, now, ioCounters)
	}

	return metrics, nil
}

func initializeDiskIOMetric(metric pdata.Metric, startTime, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	metadata.Metrics.SystemDiskIo.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeInt64DataPoint(idps.At(idx+0), startTime, now, device, metadata.LabelDiskDirection.Read, int64(ioCounter.ReadBytes))
		initializeInt64DataPoint(idps.At(idx+1), startTime, now, device, metadata.LabelDiskDirection.Write, int64(ioCounter.WriteBytes))
		idx += 2
	}
}

func initializeDiskOperationsMetric(metric pdata.Metric, startTime, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	metadata.Metrics.SystemDiskOperations.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeInt64DataPoint(idps.At(idx+0), startTime, now, device, metadata.LabelDiskDirection.Read, int64(ioCounter.ReadCount))
		initializeInt64DataPoint(idps.At(idx+1), startTime, now, device, metadata.LabelDiskDirection.Write, int64(ioCounter.WriteCount))
		idx += 2
	}
}

func initializeDiskIOTimeMetric(metric pdata.Metric, startTime, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	metadata.Metrics.SystemDiskIoTime.Init(metric)

	ddps := metric.DoubleSum().DataPoints()
	ddps.Resize(len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDoubleDataPoint(ddps.At(idx+0), startTime, now, device, "", float64(ioCounter.IoTime)/1e3)
		idx++
	}
}

func initializeDiskOperationTimeMetric(metric pdata.Metric, startTime, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	metadata.Metrics.SystemDiskOperationTime.Init(metric)

	ddps := metric.DoubleSum().DataPoints()
	ddps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDoubleDataPoint(ddps.At(idx+0), startTime, now, device, metadata.LabelDiskDirection.Read, float64(ioCounter.ReadTime)/1e3)
		initializeDoubleDataPoint(ddps.At(idx+1), startTime, now, device, metadata.LabelDiskDirection.Write, float64(ioCounter.WriteTime)/1e3)
		idx += 2
	}
}

func initializeDiskPendingOperationsMetric(metric pdata.Metric, now pdata.Timestamp, ioCounters map[string]disk.IOCountersStat) {
	metadata.Metrics.SystemDiskPendingOperations.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDiskPendingDataPoint(idps.At(idx), now, device, int64(ioCounter.IopsInProgress))
		idx++
	}
}

func initializeInt64DataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.Timestamp, deviceLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	if directionLabel != "" {
		labelsMap.Insert(metadata.Labels.DiskDirection, directionLabel)
	}
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDoubleDataPoint(dataPoint pdata.DoubleDataPoint, startTime, now pdata.Timestamp, deviceLabel string, directionLabel string, value float64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	if directionLabel != "" {
		labelsMap.Insert(metadata.Labels.DiskDirection, directionLabel)
	}
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDiskPendingDataPoint(dataPoint pdata.IntDataPoint, now pdata.Timestamp, deviceLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func (s *scraper) filterByDevice(ioCounters map[string]disk.IOCountersStat) map[string]disk.IOCountersStat {
	if s.includeFS == nil && s.excludeFS == nil {
		return ioCounters
	}

	for device := range ioCounters {
		if !s.includeDevice(device) {
			delete(ioCounters, device)
		}
	}
	return ioCounters
}

func (s *scraper) includeDevice(deviceName string) bool {
	return (s.includeFS == nil || s.includeFS.Matches(deviceName)) &&
		(s.excludeFS == nil || !s.excludeFS.Matches(deviceName))
}
