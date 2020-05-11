// Copyright 2020, OpenTelemetry Authors
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

package filesystemscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/disk"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// Scraper for FileSystem Metrics
type Scraper struct {
	config *Config
}

type deviceUsage struct {
	deviceName string
	usage      *disk.UsageStat
}

// NewFileSystemScraper creates a FileSystem Scraper
func NewFileSystemScraper(_ context.Context, cfg *Config) *Scraper {
	return &Scraper{config: cfg}
}

// Initialize
func (s *Scraper) Initialize(_ context.Context, startTime pdata.TimestampUnixNano) error {
	return nil
}

// Close
func (s *Scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeAndAppendMetrics
func (s *Scraper) ScrapeAndAppendMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "filesystemscraper.ScrapeAndAppendMetrics")
	defer span.End()

	// omit logical (virtual) filesystems (not relevant for windows)
	all := false

	partitions, err := disk.Partitions(all)
	if err != nil {
		return err
	}

	var errors []error
	usages := make([]*deviceUsage, 0, len(partitions))
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		usages = append(usages, &deviceUsage{partition.Device, usage})
	}

	if len(usages) > 0 {
		startIdx := metrics.Len()
		metrics.Resize(startIdx + 1 + systemSpecificMetricsLen)

		initializeMetricFileSystemUsed(metrics.At(startIdx+0), usages)
		appendSystemSpecificMetrics(metrics, startIdx+1, usages)
	}

	if len(errors) > 0 {
		return componenterror.CombineErrors(errors)
	}

	return nil
}

func initializeMetricFileSystemUsed(metric pdata.Metric, deviceUsages []*deviceUsage) {
	metricFilesystemUsedDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(fileSystemStatesLen * len(deviceUsages))
	for i, deviceUsage := range deviceUsages {
		appendFileSystemUsedStateDataPoints(idps, i*fileSystemStatesLen, deviceUsage)
	}
}

func initializeFileSystemUsedDataPoint(dataPoint pdata.Int64DataPoint, deviceLabel string, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
