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

package memoryscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/mem"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

const metricsLen = 1

// scraper for Memory Metrics
type scraper struct {
	config *Config

	// for mocking gopsutil mem.VirtualMemory
	virtualMemory func() (*mem.VirtualMemoryStat, error)
}

// newMemoryScraper creates a Memory Scraper
func newMemoryScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg, virtualMemory: mem.VirtualMemory}
}

// Scrape
func (s *scraper) Scrape(_ context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := internal.TimeToUnixNano(time.Now())
	memInfo, err := s.virtualMemory()
	if err != nil {
		return metrics, consumererror.NewPartialScrapeError(err, metricsLen)
	}

	metrics.Resize(metricsLen)
	initializeMemoryUsageMetric(metrics.At(0), now, memInfo)
	return metrics, nil
}

func initializeMemoryUsageMetric(metric pdata.Metric, now pdata.TimestampUnixNano, memInfo *mem.VirtualMemoryStat) {
	metadata.Metrics.SystemMemoryUsage.New().CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(memStatesLen)
	appendMemoryUsageStateDataPoints(idps, now, memInfo)
}

func initializeMemoryUsageDataPoint(dataPoint pdata.IntDataPoint, now pdata.TimestampUnixNano, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.MemState, stateLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
