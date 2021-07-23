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

// +build !windows

package pagingscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	pagingUsageMetricsLen = 1
	pagingMetricsLen      = 2
)

// scraper for Paging Metrics
type scraper struct {
	config    *Config
	startTime pdata.Timestamp

	// for mocking
	bootTime      func() (uint64, error)
	virtualMemory func() (*mem.VirtualMemoryStat, error)
	swapMemory    func() (*mem.SwapMemoryStat, error)
}

// newPagingScraper creates a Paging Scraper
func newPagingScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg, bootTime: host.BootTime, virtualMemory: mem.VirtualMemory, swapMemory: mem.SwapMemory}
}

func (s *scraper) start(context.Context, component.Host) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.Timestamp(bootTime * 1e9)
	return nil
}

func (s *scraper) scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	var errors scrapererror.ScrapeErrors

	err := s.scrapeAndAppendPagingUsageMetric(metrics)
	if err != nil {
		errors.AddPartial(pagingUsageMetricsLen, err)
	}

	err = s.scrapeAndAppendPagingMetrics(metrics)
	if err != nil {
		errors.AddPartial(pagingMetricsLen, err)
	}

	return metrics, errors.Combine()
}

func (s *scraper) scrapeAndAppendPagingUsageMetric(metrics pdata.MetricSlice) error {
	now := pdata.TimestampFromTime(time.Now())
	vmem, err := s.virtualMemory()
	if err != nil {
		return err
	}

	idx := metrics.Len()
	metrics.EnsureCapacity(idx + pagingUsageMetricsLen)
	initializePagingUsageMetric(metrics.AppendEmpty(), now, vmem)
	return nil
}

func initializePagingUsageMetric(metric pdata.Metric, now pdata.Timestamp, vmem *mem.VirtualMemoryStat) {
	metadata.Metrics.SystemPagingUsage.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.EnsureCapacity(3)
	initializePagingUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelPagingState.Used, int64(vmem.SwapTotal-vmem.SwapFree-vmem.SwapCached))
	initializePagingUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelPagingState.Free, int64(vmem.SwapFree))
	initializePagingUsageDataPoint(idps.AppendEmpty(), now, metadata.LabelPagingState.Cached, int64(vmem.SwapCached))
}

func initializePagingUsageDataPoint(dataPoint pdata.IntDataPoint, now pdata.Timestamp, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.PagingState, stateLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func (s *scraper) scrapeAndAppendPagingMetrics(metrics pdata.MetricSlice) error {
	now := pdata.TimestampFromTime(time.Now())
	swap, err := s.swapMemory()
	if err != nil {
		return err
	}

	idx := metrics.Len()
	metrics.EnsureCapacity(idx + pagingMetricsLen)
	initializePagingOperationsMetric(metrics.AppendEmpty(), s.startTime, now, swap)
	initializePageFaultsMetric(metrics.AppendEmpty(), s.startTime, now, swap)
	return nil
}

func initializePagingOperationsMetric(metric pdata.Metric, startTime, now pdata.Timestamp, swap *mem.SwapMemoryStat) {
	metadata.Metrics.SystemPagingOperations.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.EnsureCapacity(4)
	initializePagingOperationsDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelPagingType.Major, metadata.LabelPagingDirection.PageIn, int64(swap.Sin))
	initializePagingOperationsDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelPagingType.Major, metadata.LabelPagingDirection.PageOut, int64(swap.Sout))
	initializePagingOperationsDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelPagingType.Minor, metadata.LabelPagingDirection.PageIn, int64(swap.PgIn))
	initializePagingOperationsDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelPagingType.Minor, metadata.LabelPagingDirection.PageOut, int64(swap.PgOut))
}

func initializePagingOperationsDataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.Timestamp, typeLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.PagingType, typeLabel)
	labelsMap.Insert(metadata.Labels.PagingDirection, directionLabel)
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializePageFaultsMetric(metric pdata.Metric, startTime, now pdata.Timestamp, swap *mem.SwapMemoryStat) {
	metadata.Metrics.SystemPagingFaults.Init(metric)

	idps := metric.IntSum().DataPoints()
	idps.EnsureCapacity(2)
	initializePageFaultDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelPagingType.Major, int64(swap.PgMajFault))
	initializePageFaultDataPoint(idps.AppendEmpty(), startTime, now, metadata.LabelPagingType.Minor, int64(swap.PgFault-swap.PgMajFault))
}

func initializePageFaultDataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.Timestamp, typeLabel string, value int64) {
	dataPoint.LabelsMap().Insert(metadata.Labels.PagingType, typeLabel)
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
