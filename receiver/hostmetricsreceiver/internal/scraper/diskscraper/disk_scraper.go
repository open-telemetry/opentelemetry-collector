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
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// Scraper for Disk Metrics
type Scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
}

// NewDiskScraper creates a Disk Scraper
func NewDiskScraper(_ context.Context, cfg *Config) *Scraper {
	return &Scraper{config: cfg}
}

// Initialize
func (s *Scraper) Initialize(_ context.Context, startTime pdata.TimestampUnixNano) error {
	s.startTime = startTime
	return nil
}

// Close
func (s *Scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeAndAppendMetrics
func (s *Scraper) ScrapeAndAppendMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "diskscraper.ScrapeAndAppendMetrics")
	defer span.End()

	ioCounters, err := disk.IOCounters()
	if err != nil {
		return err
	}

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 3)
	initializeMetricDiskBytes(metrics.At(startIdx+0), ioCounters, s.startTime)
	initializeMetricDiskOps(metrics.At(startIdx+1), ioCounters, s.startTime)
	initializeMetricDiskTime(metrics.At(startIdx+2), ioCounters, s.startTime)
	return nil
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
