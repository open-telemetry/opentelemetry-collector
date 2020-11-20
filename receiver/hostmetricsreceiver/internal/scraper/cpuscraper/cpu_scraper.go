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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const metricsLen = 1

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

	now := internal.TimeToUnixNano(time.Now())
	cpuTimes, err := s.times( /*percpu=*/ true)
	if err != nil {
		return metrics, consumererror.NewPartialScrapeError(err, metricsLen)
	}

	metrics.Resize(metricsLen)
	initializeCPUTimeMetric(metrics.At(0), s.startTime, now, cpuTimes)
	return metrics, nil
}

func initializeCPUTimeMetric(metric pdata.Metric, startTime, now pdata.TimestampUnixNano, cpuTimes []cpu.TimesStat) {
	metadata.Metrics.SystemCPUTime.New().CopyTo(metric)

	ddps := metric.DoubleSum().DataPoints()
	ddps.Resize(len(cpuTimes) * cpuStatesLen)
	for i, cpuTime := range cpuTimes {
		appendCPUTimeStateDataPoints(ddps, i*cpuStatesLen, startTime, now, cpuTime)
	}
}

const gopsCPUTotal string = "cpu-total"

func initializeCPUTimeDataPoint(dataPoint pdata.DoubleDataPoint, startTime, now pdata.TimestampUnixNano, cpuLabel string, stateLabel string, value float64) {
	labelsMap := dataPoint.LabelsMap()
	// ignore cpu label if reporting "total" cpu usage
	if cpuLabel != gopsCPUTotal {
		labelsMap.Insert(metadata.Labels.Cpu, cpuLabel)
	}
	labelsMap.Insert(metadata.Labels.CPUState, stateLabel)

	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
