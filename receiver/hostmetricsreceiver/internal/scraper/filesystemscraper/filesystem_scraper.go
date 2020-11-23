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
	"strings"
	"time"

	"github.com/shirou/gopsutil/disk"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	standardMetricsLen = 1
	metricsLen         = standardMetricsLen + systemSpecificMetricsLen
)

// scraper for FileSystem Metrics
type scraper struct {
	config   *Config
	fsFilter fsFilter

	// for mocking gopsutil disk.Partitions & disk.Usage
	partitions func(bool) ([]disk.PartitionStat, error)
	usage      func(string) (*disk.UsageStat, error)
}

type deviceUsage struct {
	partition disk.PartitionStat
	usage     *disk.UsageStat
}

// newFileSystemScraper creates a FileSystem Scraper
func newFileSystemScraper(_ context.Context, cfg *Config) (*scraper, error) {
	fsFilter, err := cfg.createFilter()
	if err != nil {
		return nil, err
	}

	scraper := &scraper{config: cfg, partitions: disk.Partitions, usage: disk.Usage, fsFilter: *fsFilter}
	return scraper, nil
}

// Scrape
func (s *scraper) Scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := internal.TimeToUnixNano(time.Now())

	// omit logical (virtual) filesystems (not relevant for windows)
	partitions, err := s.partitions( /*all=*/ false)
	if err != nil {
		return metrics, consumererror.NewPartialScrapeError(err, metricsLen)
	}

	var errors []error
	usages := make([]*deviceUsage, 0, len(partitions))
	for _, partition := range partitions {
		if !s.fsFilter.includePartition(partition) {
			continue
		}
		usage, usageErr := s.usage(partition.Mountpoint)
		if usageErr != nil {
			errors = append(errors, consumererror.NewPartialScrapeError(usageErr, 0))
			continue
		}

		usages = append(usages, &deviceUsage{partition, usage})
	}

	if len(usages) > 0 {
		metrics.Resize(metricsLen)
		initializeFileSystemUsageMetric(metrics.At(0), now, usages)
		appendSystemSpecificMetrics(metrics, 1, now, usages)
	}

	err = scraperhelper.CombineScrapeErrors(errors)
	if err != nil && len(usages) == 0 {
		partialErr := err.(consumererror.PartialScrapeError)
		partialErr.Failed = metricsLen
		err = partialErr
	}

	return metrics, err
}

func initializeFileSystemUsageMetric(metric pdata.Metric, now pdata.TimestampUnixNano, deviceUsages []*deviceUsage) {
	fileSystemUsageDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(fileSystemStatesLen * len(deviceUsages))
	for i, deviceUsage := range deviceUsages {
		appendFileSystemUsageStateDataPoints(idps, i*fileSystemStatesLen, now, deviceUsage)
	}
}

func initializeFileSystemUsageDataPoint(dataPoint pdata.IntDataPoint, now pdata.TimestampUnixNano, partition disk.PartitionStat, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, partition.Device)
	labelsMap.Insert(typeLabelName, partition.Fstype)
	labelsMap.Insert(mountModeLabelName, getMountMode(partition.Opts))
	labelsMap.Insert(mountPointLabelName, partition.Mountpoint)
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func getMountMode(opts string) string {
	splitOptions := strings.Split(opts, ",")
	if exists(splitOptions, "rw") {
		return "rw"
	} else if exists(splitOptions, "ro") {
		return "ro"
	}
	return "unknown"
}

func exists(options []string, opt string) bool {
	for _, o := range options {
		if o == opt {
			return true
		}
	}
	return false
}

func (f *fsFilter) includePartition(partition disk.PartitionStat) bool {
	// If filters do not exist, return early.
	if !f.filtersExist || (f.includeDevice(partition.Device) &&
		f.includeFSType(partition.Fstype) &&
		f.includeMountPoint(partition.Mountpoint)) {
		return true
	}
	return false
}

func (f *fsFilter) includeDevice(deviceName string) bool {
	return (f.includeDeviceFilter == nil || f.includeDeviceFilter.Matches(deviceName)) &&
		(f.excludeDeviceFilter == nil || !f.excludeDeviceFilter.Matches(deviceName))
}

func (f *fsFilter) includeFSType(fsType string) bool {
	return (f.includeFSTypeFilter == nil || f.includeFSTypeFilter.Matches(fsType)) &&
		(f.excludeFSTypeFilter == nil || !f.excludeFSTypeFilter.Matches(fsType))
}

func (f *fsFilter) includeMountPoint(mountPoint string) bool {
	return (f.includeMountPointFilter == nil || f.includeMountPointFilter.Matches(mountPoint)) &&
		(f.excludeMountPointFilter == nil || !f.excludeMountPointFilter.Matches(mountPoint))
}
