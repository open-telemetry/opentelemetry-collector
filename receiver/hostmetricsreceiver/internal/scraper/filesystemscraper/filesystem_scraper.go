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

package filesystemscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/disk"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// scraper for FileSystem Metrics
type scraper struct {
	config *Config
}

type deviceUsage struct {
	deviceName string
	usage      *disk.UsageStat
}

// newFileSystemScraper creates a FileSystem Scraper
func newFileSystemScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg}
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
func (s *scraper) ScrapeMetrics(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	// omit logical (virtual) filesystems (not relevant for windows)
	partitions, err := disk.Partitions( /*all=*/ false)
	if err != nil {
		return metrics, err
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
		metrics.Resize(1 + systemSpecificMetricsLen)

		initializeFileSystemUsageMetric(metrics.At(0), usages)
		appendSystemSpecificMetrics(metrics, 1, usages)
	}

	if len(errors) > 0 {
		return metrics, componenterror.CombineErrors(errors)
	}

	return metrics, nil
}

func initializeFileSystemUsageMetric(metric pdata.Metric, deviceUsages []*deviceUsage) {
	fileSystemUsageDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(fileSystemStatesLen * len(deviceUsages))
	for i, deviceUsage := range deviceUsages {
		appendFileSystemUsageStateDataPoints(idps, i*fileSystemStatesLen, deviceUsage)
	}
}

func initializeFileSystemUsageDataPoint(dataPoint pdata.Int64DataPoint, deviceLabel string, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
