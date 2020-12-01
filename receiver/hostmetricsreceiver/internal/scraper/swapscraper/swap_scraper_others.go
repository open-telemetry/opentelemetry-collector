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

package swapscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	swapUsageMetricsLen = 1
	pagingMetricsLen    = 2
)

// scraper for Swap Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano

	// for mocking
	bootTime      func() (uint64, error)
	virtualMemory func() (*mem.VirtualMemoryStat, error)
	swapMemory    func() (*mem.SwapMemoryStat, error)
}

// newSwapScraper creates a Swap Scraper
func newSwapScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg, bootTime: host.BootTime, virtualMemory: mem.VirtualMemory, swapMemory: mem.SwapMemory}
}

func (s *scraper) start(context.Context, component.Host) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime * 1e9)
	return nil
}

func (s *scraper) scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	var errors []error

	err := s.scrapeAndAppendSwapUsageMetric(metrics)
	if err != nil {
		errors = append(errors, err)
	}

	err = s.scrapeAndAppendPagingMetrics(metrics)
	if err != nil {
		errors = append(errors, err)
	}

	return metrics, scraperhelper.CombineScrapeErrors(errors)
}

func (s *scraper) scrapeAndAppendSwapUsageMetric(metrics pdata.MetricSlice) error {
	now := internal.TimeToUnixNano(time.Now())
	vmem, err := s.virtualMemory()
	if err != nil {
		return consumererror.NewPartialScrapeError(err, swapUsageMetricsLen)
	}

	idx := metrics.Len()
	metrics.Resize(idx + swapUsageMetricsLen)
	initializeSwapUsageMetric(metrics.At(idx), now, vmem)
	return nil
}

func initializeSwapUsageMetric(metric pdata.Metric, now pdata.TimestampUnixNano, vmem *mem.VirtualMemoryStat) {
	swapUsageDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(3)
	initializeSwapUsageDataPoint(idps.At(0), now, usedLabelValue, int64(vmem.SwapTotal-vmem.SwapFree-vmem.SwapCached))
	initializeSwapUsageDataPoint(idps.At(1), now, freeLabelValue, int64(vmem.SwapFree))
	initializeSwapUsageDataPoint(idps.At(2), now, cachedLabelValue, int64(vmem.SwapCached))
}

func initializeSwapUsageDataPoint(dataPoint pdata.IntDataPoint, now pdata.TimestampUnixNano, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func (s *scraper) scrapeAndAppendPagingMetrics(metrics pdata.MetricSlice) error {
	now := internal.TimeToUnixNano(time.Now())
	swap, err := s.swapMemory()
	if err != nil {
		return consumererror.NewPartialScrapeError(err, pagingMetricsLen)
	}

	idx := metrics.Len()
	metrics.Resize(idx + pagingMetricsLen)
	initializePagingMetric(metrics.At(idx+0), s.startTime, now, swap)
	initializePageFaultsMetric(metrics.At(idx+1), s.startTime, now, swap)
	return nil
}

func initializePagingMetric(metric pdata.Metric, startTime, now pdata.TimestampUnixNano, swap *mem.SwapMemoryStat) {
	swapPagingDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(4)
	initializePagingDataPoint(idps.At(0), startTime, now, majorTypeLabelValue, inDirectionLabelValue, int64(swap.Sin))
	initializePagingDataPoint(idps.At(1), startTime, now, majorTypeLabelValue, outDirectionLabelValue, int64(swap.Sout))
	initializePagingDataPoint(idps.At(2), startTime, now, minorTypeLabelValue, inDirectionLabelValue, int64(swap.PgIn))
	initializePagingDataPoint(idps.At(3), startTime, now, minorTypeLabelValue, outDirectionLabelValue, int64(swap.PgOut))
}

func initializePagingDataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.TimestampUnixNano, typeLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(typeLabelName, typeLabel)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializePageFaultsMetric(metric pdata.Metric, startTime, now pdata.TimestampUnixNano, swap *mem.SwapMemoryStat) {
	swapPageFaultsDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(1)
	initializePageFaultDataPoint(idps.At(0), startTime, now, minorTypeLabelValue, int64(swap.PgFault))
	// TODO add swap.PgMajFault once available in gopsutil
}

func initializePageFaultDataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.TimestampUnixNano, typeLabel string, value int64) {
	dataPoint.LabelsMap().Insert(typeLabelName, typeLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
