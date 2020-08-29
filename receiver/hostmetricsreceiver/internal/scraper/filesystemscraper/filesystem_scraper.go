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

package filesystemscraper

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/disk"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

// scraper for FileSystem Metrics
type scraper struct {
	config    *Config
	includeFS filterset.FilterSet
	excludeFS filterset.FilterSet

	// for mocking gopsutil disk.Partitions & disk.Usage
	partitions func(bool) ([]disk.PartitionStat, error)
	usage      func(string) (*disk.UsageStat, error)
}

type deviceUsage struct {
	deviceName string
	usage      *disk.UsageStat
}

// newFileSystemScraper creates a FileSystem Scraper
func newFileSystemScraper(_ context.Context, cfg *Config) (*scraper, error) {
	scraper := &scraper{config: cfg, partitions: disk.Partitions, usage: disk.Usage}

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

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(_ context.Context) (dataold.MetricSlice, error) {
	metrics := dataold.NewMetricSlice()

	now := internal.TimeToUnixNano(time.Now())

	// omit logical (virtual) filesystems (not relevant for windows)
	partitions, err := s.partitions( /*all=*/ false)
	if err != nil {
		return metrics, err
	}

	var errors []error
	usages := make([]*deviceUsage, 0, len(partitions))
	for _, partition := range partitions {
		// Skip partition stats having more than one mount point for the same device
		if deviceUsageAlreadySet(partition.Device, usages) {
			continue
		}

		usage, err := s.usage(partition.Mountpoint)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		usages = append(usages, &deviceUsage{partition.Device, usage})
	}

	// filter devices by name
	usages = s.filterByDevice(usages)

	if len(usages) > 0 {
		metrics.Resize(1 + systemSpecificMetricsLen)

		initializeFileSystemUsageMetric(metrics.At(0), now, usages)
		appendSystemSpecificMetrics(metrics, 1, now, usages)
	}

	return metrics, componenterror.CombineErrors(errors)
}

func deviceUsageAlreadySet(device string, usages []*deviceUsage) bool {
	for _, usage := range usages {
		if device == usage.deviceName {
			return true
		}
	}
	return false
}

func initializeFileSystemUsageMetric(metric dataold.Metric, now pdata.TimestampUnixNano, deviceUsages []*deviceUsage) {
	fileSystemUsageDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(fileSystemStatesLen * len(deviceUsages))
	for i, deviceUsage := range deviceUsages {
		appendFileSystemUsageStateDataPoints(idps, i*fileSystemStatesLen, now, deviceUsage)
	}
}

func initializeFileSystemUsageDataPoint(dataPoint dataold.Int64DataPoint, now pdata.TimestampUnixNano, deviceLabel string, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func (s *scraper) filterByDevice(usages []*deviceUsage) []*deviceUsage {
	if s.includeFS == nil && s.excludeFS == nil {
		return usages
	}

	filteredUsages := make([]*deviceUsage, 0, len(usages))
	for _, usage := range usages {
		if s.includeDevice(usage.deviceName) {
			filteredUsages = append(filteredUsages, usage)
		}
	}
	return filteredUsages
}

func (s *scraper) includeDevice(deviceName string) bool {
	return (s.includeFS == nil || s.includeFS.Matches(deviceName)) &&
		(s.excludeFS == nil || !s.excludeFS.Matches(deviceName))
}
