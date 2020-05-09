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
	"fmt"
	"time"

	"github.com/shirou/gopsutil/disk"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

// Scraper for FileSystem Metrics
type Scraper struct {
	config   *Config
	consumer consumer.MetricsConsumer
	cancel   context.CancelFunc
}

type deviceUsage struct {
	deviceName string
	usage      *disk.UsageStat
}

// NewFileSystemScraper creates a set of FileSystem related metrics
func NewFileSystemScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
	return &Scraper{config: cfg, consumer: consumer}, nil
}

// Start
func (s *Scraper) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)

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

// Shutdown
func (s *Scraper) Shutdown(ctx context.Context) error {
	s.cancel()
	return nil
}

func (s *Scraper) scrapeMetrics(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "filesystemscraper.scrapeMetrics")
	defer span.End()

	metricData := data.NewMetricData()
	metrics := internal.InitializeMetricSlice(metricData)

	err := s.scrapeAndAppendMetrics(metrics)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping filesystem metrics: %v", err)})
	}

	if metrics.Len() > 0 {
		err := s.consumer.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(metricData))
		if err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
			return
		}
	}
}

// scrapeAndAppendMetrics appends scraped metrics to the provided metric slice.
// If errors occur scraping some metrics, an error will be returned, but the
// successfully scraped metrics will still be appended to the provided slice
func (s *Scraper) scrapeAndAppendMetrics(metrics pdata.MetricSlice) error {
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
		metrics.Resize(1 + systemSpecificMetricsLen)

		initializeMetricFileSystemUsed(metrics.At(0), usages)
		appendSystemSpecificMetrics(metrics, 1, usages)
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
