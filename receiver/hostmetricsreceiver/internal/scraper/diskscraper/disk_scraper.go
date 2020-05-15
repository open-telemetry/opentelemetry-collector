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

package diskscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// scraper for Disk Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
}

// newDiskScraper creates a Disk Scraper
func newDiskScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg}
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime)
	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(ctx context.Context) (pdata.MetricSlice, error) {
	_, span := trace.StartSpan(ctx, "diskscraper.ScrapeMetrics")
	defer span.End()

	metrics := pdata.NewMetricSlice()

	ioCounters, err := disk.IOCounters()
	if err != nil {
		return metrics, err
	}

	metrics.Resize(3)
	initializeMetricDiskBytes(metrics.At(0), ioCounters, s.startTime)
	initializeMetricDiskOps(metrics.At(1), ioCounters, s.startTime)
	initializeMetricDiskTime(metrics.At(2), ioCounters, s.startTime)
	return metrics, nil
}

func initializeMetricDiskBytes(metric pdata.Metric, ioCounters map[string]disk.IOCountersStat, startTime pdata.TimestampUnixNano) {
	metricDiskBytesDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDataPoint(idps.At(idx+0), startTime, device, readDirectionLabelValue, int64(ioCounter.ReadBytes))
		initializeDataPoint(idps.At(idx+1), startTime, device, writeDirectionLabelValue, int64(ioCounter.WriteBytes))
		idx += 2
	}
}

func initializeMetricDiskOps(metric pdata.Metric, ioCounters map[string]disk.IOCountersStat, startTime pdata.TimestampUnixNano) {
	metricDiskOpsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDataPoint(idps.At(idx+0), startTime, device, readDirectionLabelValue, int64(ioCounter.ReadCount))
		initializeDataPoint(idps.At(idx+1), startTime, device, writeDirectionLabelValue, int64(ioCounter.WriteCount))
		idx += 2
	}
}

func initializeMetricDiskTime(metric pdata.Metric, ioCounters map[string]disk.IOCountersStat, startTime pdata.TimestampUnixNano) {
	metricDiskTimeDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDataPoint(idps.At(idx+0), startTime, device, readDirectionLabelValue, int64(ioCounter.ReadTime))
		initializeDataPoint(idps.At(idx+1), startTime, device, writeDirectionLabelValue, int64(ioCounter.WriteTime))
		idx += 2
	}
}

func initializeDataPoint(dataPoint pdata.Int64DataPoint, startTime pdata.TimestampUnixNano, deviceLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
