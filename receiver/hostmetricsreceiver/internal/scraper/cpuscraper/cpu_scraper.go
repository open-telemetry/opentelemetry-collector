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
	"time"

	"github.com/shirou/gopsutil/cpu"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// Scraper for CPU Metrics
type Scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
}

// NewCPUScraper creates a set of CPU related metrics
func NewCPUScraper(_ context.Context, cfg *Config) *Scraper {
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
	_, span := trace.StartSpan(ctx, "cpuscraper.ScrapeAndAppendMetrics")
	defer span.End()

	cpuTimes, err := cpu.Times(s.config.ReportPerCPU)
	if err != nil {
		return err
	}

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 1)
	initializeCPUSecondsMetric(metrics.At(startIdx), s.startTime, cpuTimes)
	return nil
}

func initializeCPUSecondsMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, cpuTimes []cpu.TimesStat) {
	metricCPUSecondsDescriptor.CopyTo(metric.MetricDescriptor())

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
		labelsMap.Insert(cpuLabelName, cpuLabel)
	}
	labelsMap.Insert(stateLabelName, stateLabel)

	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
