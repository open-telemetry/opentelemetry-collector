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

package pagingscraper

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/host"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/perfcounters"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/pagingscraper/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	pagingUsageMetricsLen = 1
	pagingMetricsLen      = 1

	memory = "Memory"

	pageReadsPerSec  = "Page Reads/sec"
	pageWritesPerSec = "Page Writes/sec"
)

// scraper for Paging Metrics
type scraper struct {
	config    *Config
	startTime pdata.Timestamp

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

// newPagingScraper creates a Paging Scraper
func newPagingScraper(_ context.Context, cfg *Config) *scraper {
	once.Do(func() { pageSize = getPageSize() })

	return &scraper{config: cfg, pageSize: pageSize, perfCounterScraper: &perfcounters.PerfLibScraper{}, bootTime: host.BootTime, pageFileStats: getPageFileStats}
}

func (s *scraper) start(context.Context, component.Host) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.Timestamp(bootTime * 1e9)

	return s.perfCounterScraper.Initialize(memory)
}

func (s *scraper) scrape(context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	var errors scrapererror.ScrapeErrors

	err := s.scrapeAndAppendPagingUsageMetric(metrics)
	if err != nil {
		errors.AddPartial(pagingUsageMetricsLen, err)
	}

	err = s.scrapeAndAppendPagingOperationsMetric(metrics)
	if err != nil {
		errors.AddPartial(pagingMetricsLen, err)
	}

	return metrics, errors.Combine()
}

func (s *scraper) scrapeAndAppendPagingUsageMetric(metrics pdata.MetricSlice) error {
	now := pdata.NewTimestampFromTime(time.Now())
	pageFiles, err := s.pageFileStats()
	if err != nil {
		return err
	}

	idx := metrics.Len()
	metrics.EnsureCapacity(idx + pagingUsageMetricsLen)
	s.initializePagingUsageMetric(metrics.AppendEmpty(), now, pageFiles)
	return nil
}

func (s *scraper) initializePagingUsageMetric(metric pdata.Metric, now pdata.Timestamp, pageFiles []*pageFileData) {
	metadata.Metrics.SystemPagingUsage.Init(metric)

	idps := metric.Sum().DataPoints()
	idps.EnsureCapacity(2 * len(pageFiles))

	for _, pageFile := range pageFiles {
		initializePagingUsageDataPoint(idps.AppendEmpty(), now, pageFile.name, metadata.LabelState.Used, int64(pageFile.usedPages*s.pageSize))
		initializePagingUsageDataPoint(idps.AppendEmpty(), now, pageFile.name, metadata.LabelState.Free, int64((pageFile.totalPages-pageFile.usedPages)*s.pageSize))
	}
}

func initializePagingUsageDataPoint(dataPoint pdata.NumberDataPoint, now pdata.Timestamp, deviceLabel string, stateLabel string, value int64) {
	attributes := dataPoint.Attributes()
	attributes.InsertString(metadata.Labels.Device, deviceLabel)
	attributes.InsertString(metadata.Labels.State, stateLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetIntVal(value)
}

func (s *scraper) scrapeAndAppendPagingOperationsMetric(metrics pdata.MetricSlice) error {
	now := pdata.NewTimestampFromTime(time.Now())

	counters, err := s.perfCounterScraper.Scrape()
	if err != nil {
		return err
	}

	memoryObject, err := counters.GetObject(memory)
	if err != nil {
		return err
	}

	memoryCounterValues, err := memoryObject.GetValues(pageReadsPerSec, pageWritesPerSec)
	if err != nil {
		return err
	}

	if len(memoryCounterValues) > 0 {
		idx := metrics.Len()
		metrics.EnsureCapacity(idx + pagingMetricsLen)
		initializePagingOperationsMetric(metrics.AppendEmpty(), s.startTime, now, memoryCounterValues[0])
	}

	return nil
}

func initializePagingOperationsMetric(metric pdata.Metric, startTime, now pdata.Timestamp, memoryCounterValues *perfcounters.CounterValues) {
	metadata.Metrics.SystemPagingOperations.Init(metric)

	idps := metric.Sum().DataPoints()
	idps.EnsureCapacity(2)
	initializePagingOperationsDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelDirection.PageIn, memoryCounterValues.Values[pageReadsPerSec])
	initializePagingOperationsDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelDirection.PageOut, memoryCounterValues.Values[pageWritesPerSec])
}

func initializePagingOperationsDataPoint(dataPoint pdata.NumberDataPoint, startTime, now pdata.Timestamp, directionLabel string, value int64) {
	attributes := dataPoint.Attributes()
	attributes.InsertString(metadata.Labels.Type, metadata.LabelType.Major)
	attributes.InsertString(metadata.Labels.Direction, directionLabel)
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetIntVal(value)
}
