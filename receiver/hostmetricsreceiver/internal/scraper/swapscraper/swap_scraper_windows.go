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

// +build windows

package swapscraper

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/host"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/perfcounters"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	swapUsageMetricsLen = 1
	pagingMetricsLen    = 1

	memory = "Memory"

	pageReadsPerSec  = "Page Reads/sec"
	pageWritesPerSec = "Page Writes/sec"
)

// scraper for Swap Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano

	pageSize uint64

	perfCounterScraper perfcounters.PerfCounterScraper

	// for mocking
	bootTime      func() (uint64, error)
	pageFileStats func() ([]*pageFileData, error)
}

var (
	once     sync.Once
	pageSize uint64
)

// newSwapScraper creates a Swap Scraper
func newSwapScraper(_ context.Context, cfg *Config) *scraper {
	once.Do(func() { pageSize = getPageSize() })

	return &scraper{config: cfg, pageSize: pageSize, perfCounterScraper: &perfcounters.PerfLibScraper{}, bootTime: host.BootTime, pageFileStats: getPageFileStats}
}

func (s *scraper) start(context.Context, component.Host) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime * 1e9)

	return s.perfCounterScraper.Initialize(memory)
}

func (s *scraper) scrape(context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	var errors []error

	err := s.scrapeAndAppendSwapUsageMetric(metrics)
	if err != nil {
		errors = append(errors, err)
	}

	err = s.scrapeAndAppendPagingMetric(metrics)
	if err != nil {
		errors = append(errors, err)
	}

	return metrics, scraperhelper.CombineScrapeErrors(errors)
}

func (s *scraper) scrapeAndAppendSwapUsageMetric(metrics pdata.MetricSlice) error {
	now := internal.TimeToUnixNano(time.Now())
	pageFiles, err := s.pageFileStats()
	if err != nil {
		return consumererror.NewPartialScrapeError(err, swapUsageMetricsLen)
	}

	idx := metrics.Len()
	metrics.Resize(idx + swapUsageMetricsLen)
	s.initializeSwapUsageMetric(metrics.At(idx), now, pageFiles)
	return nil
}

func (s *scraper) initializeSwapUsageMetric(metric pdata.Metric, now pdata.TimestampUnixNano, pageFiles []*pageFileData) {
	swapUsageDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(pageFiles))

	idx := 0
	for _, pageFile := range pageFiles {
		initializeSwapUsageDataPoint(idps.At(idx+0), now, pageFile.name, usedLabelValue, int64(pageFile.usedPages*s.pageSize))
		initializeSwapUsageDataPoint(idps.At(idx+1), now, pageFile.name, freeLabelValue, int64((pageFile.totalPages-pageFile.usedPages)*s.pageSize))
		idx += 2
	}
}

func initializeSwapUsageDataPoint(dataPoint pdata.IntDataPoint, now pdata.TimestampUnixNano, deviceLabel string, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func (s *scraper) scrapeAndAppendPagingMetric(metrics pdata.MetricSlice) error {
	now := internal.TimeToUnixNano(time.Now())

	counters, err := s.perfCounterScraper.Scrape()
	if err != nil {
		return consumererror.NewPartialScrapeError(err, pagingMetricsLen)
	}

	memoryObject, err := counters.GetObject(memory)
	if err != nil {
		return consumererror.NewPartialScrapeError(err, pagingMetricsLen)
	}

	memoryCounterValues, err := memoryObject.GetValues(pageReadsPerSec, pageWritesPerSec)
	if err != nil {
		return consumererror.NewPartialScrapeError(err, pagingMetricsLen)
	}

	if len(memoryCounterValues) > 0 {
		idx := metrics.Len()
		metrics.Resize(idx + pagingMetricsLen)
		initializePagingMetric(metrics.At(idx), s.startTime, now, memoryCounterValues[0])
	}

	return nil
}

func initializePagingMetric(metric pdata.Metric, startTime, now pdata.TimestampUnixNano, memoryCounterValues *perfcounters.CounterValues) {
	swapPagingDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2)
	initializePagingDataPoint(idps.At(0), startTime, now, inDirectionLabelValue, memoryCounterValues.Values[pageReadsPerSec])
	initializePagingDataPoint(idps.At(1), startTime, now, outDirectionLabelValue, memoryCounterValues.Values[pageWritesPerSec])
}

func initializePagingDataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.TimestampUnixNano, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(typeLabelName, majorTypeLabelValue)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
