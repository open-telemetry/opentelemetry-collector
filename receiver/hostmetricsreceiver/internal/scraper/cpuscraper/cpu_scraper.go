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

package cpuscraper

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

// Scraper for CPU Metrics
type Scraper struct {
	config         *Config
	consumer       consumer.MetricsConsumer
	startTime      pdata.TimestampUnixNano
	previousValues map[string]*cpuTotalUsedStat
	cancel         context.CancelFunc
}

type cpuTotalUsedStat struct {
	total float64
	used  float64
}

// NewCPUScraper creates a set of CPU related metrics
func NewCPUScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
	return &Scraper{config: cfg, consumer: consumer}, nil
}

// Start
func (c *Scraper) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)

	c.initializeStartTime()
	c.initializeCPUValues()

	go func() {
		ticker := time.NewTicker(c.config.CollectionInterval())
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.scrapeMetrics(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Close
func (c *Scraper) Close(ctx context.Context) error {
	c.cancel()
	return nil
}

func (c *Scraper) initializeStartTime() error {
	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	c.startTime = pdata.TimestampUnixNano(bootTime)
	return nil
}

func (c *Scraper) initializeCPUValues() error {
	cpuTimes, err := cpu.Times(c.config.ReportPerCPU)
	if err != nil {
		return err
	}

	c.setPreviousCPUValues(cpuTimes)
	return nil
}

func (c *Scraper) setPreviousCPUValues(cpuTimes []cpu.TimesStat) {
	c.previousValues = make(map[string]*cpuTotalUsedStat, len(cpuTimes))
	for _, cpuTime := range cpuTimes {
		c.previousValues[cpuTime.CPU] = convertToTotalUsedStat(cpuTime)
	}
}

func convertToTotalUsedStat(cpuTime cpu.TimesStat) *cpuTotalUsedStat {
	total := cpuTime.Total()
	used := total - cpuTime.Idle
	return &cpuTotalUsedStat{total, used}
}

func (c *Scraper) scrapeMetrics(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "cpuscraper.scrapeMetrics")
	defer span.End()

	metricData := data.NewMetricData()
	metrics := internal.InitializeMetricSlice(metricData)

	err := c.scrapeAndAppendMetrics(metrics)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping cpu metrics: %v", err)})
		return
	}

	if metrics.Len() > 0 {
		err := c.consumer.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(metricData))
		if err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
			return
		}
	}
}

func (c *Scraper) scrapeAndAppendMetrics(metrics pdata.MetricSlice) error {
	cpuTimes, err := cpu.Times(c.config.ReportPerCPU)
	if err != nil {
		return err
	}

	metric := internal.AddNewMetric(metrics)
	c.initializeCPUSecondsMetric(metric, cpuTimes)

	metric = internal.AddNewMetric(metrics)
	c.initializeCPUUtilizationMetric(metric, cpuTimes)
	return nil
}

func (c *Scraper) initializeCPUSecondsMetric(metric pdata.Metric, cpuTimes []cpu.TimesStat) {
	MetricCPUSecondsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(4 * len(cpuTimes))
	for i, cpuTime := range cpuTimes {
		c.initializeCPUSecondsDataPoint(idps.At(4*i+0), cpuTime.CPU, UserStateLabelValue, int64(cpuTime.User))
		c.initializeCPUSecondsDataPoint(idps.At(4*i+1), cpuTime.CPU, SystemStateLabelValue, int64(cpuTime.System))
		c.initializeCPUSecondsDataPoint(idps.At(4*i+2), cpuTime.CPU, IdleStateLabelValue, int64(cpuTime.Idle))
		c.initializeCPUSecondsDataPoint(idps.At(4*i+3), cpuTime.CPU, InterruptStateLabelValue, int64(cpuTime.Irq))
	}
}

const gopsCPUTotal string = "cpu-total"

func (c *Scraper) initializeCPUSecondsDataPoint(dataPoint pdata.Int64DataPoint, cpuLabel string, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	// ignore cpu label if reporting "total" cpu usage
	if cpuLabel != gopsCPUTotal {
		labelsMap.Insert(CPULabel, cpuLabel)
	}
	labelsMap.Insert(StateLabel, stateLabel)

	dataPoint.SetStartTime(c.startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}

func (c *Scraper) initializeCPUUtilizationMetric(metric pdata.Metric, cpuTimes []cpu.TimesStat) {
	MetricCPUUtilizationDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.DoubleDataPoints()
	idps.Resize(len(cpuTimes))
	for i, cpuTime := range cpuTimes {
		totalUsed := convertToTotalUsedStat(cpuTime)
		utilization := getUtilization(c.previousValues[cpuTime.CPU], totalUsed)
		c.initializeCPUUtilizationDataPoint(idps.At(i), cpuTime.CPU, utilization)
	}

	c.setPreviousCPUValues(cpuTimes)
}

func (c *Scraper) initializeCPUUtilizationDataPoint(dataPoint pdata.DoubleDataPoint, cpuLabel string, value float64) {
	// ignore cpu label if reporting "total" cpu usage
	if cpuLabel != gopsCPUTotal {
		dataPoint.LabelsMap().Insert(CPULabel, cpuLabel)
	}

	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}

func getUtilization(prev *cpuTotalUsedStat, current *cpuTotalUsedStat) float64 {
	if prev == nil {
		return 0
	}

	usedDiff := current.used - prev.used
	totalDiff := current.total - prev.total

	if totalDiff == 0 {
		return 0
	}

	return math.Min(100, math.Max(0, usedDiff/totalDiff*100))
}
