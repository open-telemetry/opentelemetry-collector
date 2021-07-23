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

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
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

func (s *scraper) Scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := pdata.TimestampFromTime(time.Now())

	// omit logical (virtual) filesystems (not relevant for windows)
	partitions, err := s.partitions( /*all=*/ false)
	if err != nil {
		return metrics, scrapererror.NewPartialScrapeError(err, metricsLen)
	}

	var errors scrapererror.ScrapeErrors
	usages := make([]*deviceUsage, 0, len(partitions))
	for _, partition := range partitions {
		if !s.fsFilter.includePartition(partition) {
			continue
		}
		usage, usageErr := s.usage(partition.Mountpoint)
		if usageErr != nil {
			errors.AddPartial(0, usageErr)
			continue
		}

		usages = append(usages, &deviceUsage{partition, usage})
	}

	if len(usages) > 0 {
		metrics.EnsureCapacity(metricsLen)
		initializeFileSystemUsageMetric(metrics.AppendEmpty(), now, usages)
		appendSystemSpecificMetrics(metrics, now, usages)
	}

	err = errors.Combine()
	if err != nil && len(usages) == 0 {
		err = scrapererror.NewPartialScrapeError(err, metricsLen)
	}

	return metrics, err
}

func initializeFileSystemUsageMetric(metric pdata.Metric, now pdata.Timestamp, deviceUsages []*deviceUsage) {
	metadata.Metrics.SystemFilesystemUsage.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.EnsureCapacity(fileSystemStatesLen * len(deviceUsages))
	for _, deviceUsage := range deviceUsages {
		appendFileSystemUsageStateDataPoints(idps, now, deviceUsage)
	}
}

func initializeFileSystemUsageDataPoint(dataPoint pdata.IntDataPoint, now pdata.Timestamp, partition disk.PartitionStat, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.FilesystemDevice, partition.Device)
	labelsMap.Insert(metadata.Labels.FilesystemType, partition.Fstype)
	labelsMap.Insert(metadata.Labels.FilesystemMode, getMountMode(partition.Opts))
	labelsMap.Insert(metadata.Labels.FilesystemMountpoint, partition.Mountpoint)
	labelsMap.Insert(metadata.Labels.FilesystemState, stateLabel)
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
