// Copyright The OpenTelemetry Authors
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
	"github.com/shirou/gopsutil/host"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// scraper for CPU Metrics
type scraper struct {
	config       *Config
	startTime    pdata.TimestampUnixNano
	prevCPUTimes map[string]cpu.TimesStat

	// for mocking gopsutil cpu.Times
	times func(bool) ([]cpu.TimesStat, error)
}

// newCPUScraper creates a set of CPU related metrics
func newCPUScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg, times: cpu.Times}
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime)
	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	cpuTimes, err := s.times(s.config.ReportPerCPU)
	if err != nil {
		return metrics, err
	}

	noMetrics := 1
	if s.prevCPUTimes != nil {
		noMetrics = 2
	}

	metrics.Resize(noMetrics)
	initializeCPUTimeMetric(metrics.At(0), s.startTime, cpuTimes)

	if s.prevCPUTimes != nil {
		initializeCPUUtilizationMetric(metrics.At(1), s.startTime, cpuTimes, s.prevCPUTimes)
	}

	// store cpu times so we can calculate utilization over the collection interval
	s.prevCPUTimes = make(map[string]cpu.TimesStat, len(cpuTimes))
	for _, cpuTime := range cpuTimes {
		s.prevCPUTimes[cpuTime.CPU] = cpuTime
	}

	return metrics, nil
}

func initializeCPUTimeMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, cpuTimes []cpu.TimesStat) {
	cpuTimeDescriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.DoubleDataPoints()
	ddps.Resize(len(cpuTimes) * cpuStatesLen)
	for i, cpuTime := range cpuTimes {
		appendCPUTimeStateDataPoints(ddps, i*cpuStatesLen, startTime, cpuTime)
	}
}

func initializeCPUUtilizationMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, cpuTimes []cpu.TimesStat, prevCPUTimes map[string]cpu.TimesStat) {
	cpuUtilizationDescriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.DoubleDataPoints()
	ddps.Resize(len(cpuTimes) * cpuStatesLen)
	for i, cpuTime := range cpuTimes {
		prevCPUTime, ok := prevCPUTimes[cpuTime.CPU]
		if !ok {
			continue
		}

		appendCPUUtilizationStateDataPoints(ddps, i*cpuStatesLen, startTime, cpuTime, prevCPUTime)
	}
}

const gopsCPUTotal string = "cpu-total"

func initializeCPUDataPoint(dataPoint pdata.DoubleDataPoint, startTime pdata.TimestampUnixNano, cpuLabel string, stateLabel string, value float64) {
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
