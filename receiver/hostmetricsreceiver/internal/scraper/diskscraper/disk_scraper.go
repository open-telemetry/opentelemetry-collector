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
	"fmt"
	"time"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

// Scraper for Disk Metrics
type Scraper struct {
	config    *Config
	consumer  consumer.MetricsConsumer
	startTime pdata.TimestampUnixNano
	cancel    context.CancelFunc
}

// NewDiskScraper creates a set of Disk related metrics
func NewDiskScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
	return &Scraper{config: cfg, consumer: consumer}, nil
}

// Start
func (s *Scraper) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)

	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime)

	go func() {
		ticker := time.NewTicker(s.config.CollectionInterval())
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.scrapeMetrics(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Close
func (s *Scraper) Close(ctx context.Context) error {
	s.cancel()
	return nil
}

func (s *Scraper) scrapeMetrics(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "diskscraper.scrapeMetrics")
	defer span.End()

	metricData := data.NewMetricData()
	metrics := internal.InitializeMetricSlice(metricData)

	err := s.scrapeAndAppendMetrics(metrics)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping disk metrics: %v", err)})
		return
	}

	if metrics.Len() > 0 {
		err := s.consumer.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(metricData))
		if err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
			return
		}
	}
}

func (s *Scraper) scrapeAndAppendMetrics(metrics pdata.MetricSlice) error {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return err
	}

	metrics.Resize(3)
	initializeMetricDiskBytes(metrics.At(0), ioCounters, s.startTime)
	initializeMetricDiskOps(metrics.At(1), ioCounters, s.startTime)
	initializeMetricDiskTime(metrics.At(2), ioCounters, s.startTime)
	return nil
}

func initializeMetricDiskBytes(metric pdata.Metric, ioCounters map[string]disk.IOCountersStat, startTime pdata.TimestampUnixNano) {
	metricDiskBytesDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDataPoint(idps.At(idx+0), startTime, device, receiveDirectionLabelValue, int64(ioCounter.ReadBytes))
		initializeDataPoint(idps.At(idx+1), startTime, device, transmitDirectionLabelValue, int64(ioCounter.WriteBytes))
		idx += 2
	}
}

func initializeMetricDiskOps(metric pdata.Metric, ioCounters map[string]disk.IOCountersStat, startTime pdata.TimestampUnixNano) {
	metricDiskOpsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDataPoint(idps.At(idx+0), startTime, device, receiveDirectionLabelValue, int64(ioCounter.ReadCount))
		initializeDataPoint(idps.At(idx+1), startTime, device, transmitDirectionLabelValue, int64(ioCounter.WriteCount))
		idx += 2
	}
}

func initializeMetricDiskTime(metric pdata.Metric, ioCounters map[string]disk.IOCountersStat, startTime pdata.TimestampUnixNano) {
	metricDiskTimeDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2 * len(ioCounters))

	idx := 0
	for device, ioCounter := range ioCounters {
		initializeDataPoint(idps.At(idx+0), startTime, device, receiveDirectionLabelValue, int64(ioCounter.ReadTime))
		initializeDataPoint(idps.At(idx+1), startTime, device, transmitDirectionLabelValue, int64(ioCounter.WriteTime))
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
