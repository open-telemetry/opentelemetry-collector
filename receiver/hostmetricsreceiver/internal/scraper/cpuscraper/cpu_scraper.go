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

package cpuscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

// scraper for CPU Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano

	// for mocking
	bootTime func() (uint64, error)
	times    func(bool) ([]cpu.TimesStat, error)
}

// newCPUScraper creates a set of CPU related metrics
func newCPUScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg, bootTime: host.BootTime, times: cpu.Times}
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime * 1e9)
	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(_ context.Context) (dataold.MetricSlice, error) {
	metrics := dataold.NewMetricSlice()

	now := internal.TimeToUnixNano(time.Now())
	cpuTimes, err := s.times( /*percpu=*/ true)
	if err != nil {
		return metrics, err
	}

	metrics.Resize(1)
	initializeCPUTimeMetric(metrics.At(0), s.startTime, now, cpuTimes)
	return metrics, nil
}

func initializeCPUTimeMetric(metric dataold.Metric, startTime, now pdata.TimestampUnixNano, cpuTimes []cpu.TimesStat) {
	cpuTimeDescriptor.CopyTo(metric.MetricDescriptor())

	ddps := metric.DoubleDataPoints()
	ddps.Resize(len(cpuTimes) * cpuStatesLen)
	for i, cpuTime := range cpuTimes {
		appendCPUTimeStateDataPoints(ddps, i*cpuStatesLen, startTime, now, cpuTime)
	}
}

const gopsCPUTotal string = "cpu-total"

func initializeCPUTimeDataPoint(dataPoint dataold.DoubleDataPoint, startTime, now pdata.TimestampUnixNano, cpuLabel string, stateLabel string, value float64) {
	labelsMap := dataPoint.LabelsMap()
	// ignore cpu label if reporting "total" cpu usage
	if cpuLabel != gopsCPUTotal {
		labelsMap.Insert(cpuLabelName, cpuLabel)
	}
	labelsMap.Insert(stateLabelName, stateLabel)

	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
