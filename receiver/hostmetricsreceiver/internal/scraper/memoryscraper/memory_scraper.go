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

package memoryscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/mem"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// scraper for Memory Metrics
type scraper struct {
	config *Config
}

// newMemoryScraper creates a Memory Scraper
func newMemoryScraper(_ context.Context, cfg *Config) *scraper {
	return &scraper{config: cfg}
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	return nil
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(ctx context.Context) (pdata.MetricSlice, error) {
	_, span := trace.StartSpan(ctx, "memoryscraper.ScrapeMetrics")
	defer span.End()

	metrics := pdata.NewMetricSlice()

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return metrics, err
	}

	metrics.Resize(1)
	initializeMemoryUsageMetric(metrics.At(0), memInfo)
	return metrics, nil
}

func initializeMemoryUsageMetric(metric pdata.Metric, memInfo *mem.VirtualMemoryStat) {
	memoryUsageDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(memStatesLen)
	appendMemoryUsageStateDataPoints(idps, memInfo)
}

func initializeMemoryUsageDataPoint(dataPoint pdata.Int64DataPoint, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
