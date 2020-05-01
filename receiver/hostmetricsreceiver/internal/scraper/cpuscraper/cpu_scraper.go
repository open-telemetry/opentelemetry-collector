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
	config    *Config
	consumer  consumer.MetricsConsumer
	startTime pdata.TimestampUnixNano
	cancel    context.CancelFunc
}

// NewCPUScraper creates a set of CPU related metrics
func NewCPUScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
	return &Scraper{config: cfg, consumer: consumer}, nil
}

// Start
func (c *Scraper) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)

	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	c.startTime = pdata.TimestampUnixNano(bootTime)

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
	initializeCPUSecondsMetric(metric, c.startTime, cpuTimes)
	return nil
}

func initializeCPUSecondsMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, cpuTimes []cpu.TimesStat) {
	MetricCPUSecondsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(cpuTimes) * cpuStatesLen)
	for i, cpuTime := range cpuTimes {
		appendCPUStateTimes(idps, i*cpuStatesLen, startTime, cpuTime)
	}
}

const gopsCPUTotal string = "cpu-total"

func initializeCPUSecondsDataPoint(dataPoint pdata.Int64DataPoint, startTime pdata.TimestampUnixNano, cpuLabel string, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	// ignore cpu label if reporting "total" cpu usage
	if cpuLabel != gopsCPUTotal {
		labelsMap.Insert(CPULabel, cpuLabel)
	}
	labelsMap.Insert(StateLabel, stateLabel)

	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
