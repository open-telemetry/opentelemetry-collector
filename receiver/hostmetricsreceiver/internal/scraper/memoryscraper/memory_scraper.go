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

package memoryscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/mem"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
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

// ScrapeAndAppendMetrics
func (s *scraper) ScrapeAndAppendMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "memoryscraper.ScrapeAndAppendMetrics")
	defer span.End()

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 1)
	initializeMetricMemoryUsed(metrics.At(startIdx), memInfo)
	return nil
}

func initializeMetricMemoryUsed(metric pdata.Metric, memInfo *mem.VirtualMemoryStat) {
	metricMemoryUsedDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(memStatesLen)
	appendMemoryUsedStates(idps, memInfo)
}

func initializeMemoryUsedDataPoint(dataPoint pdata.Int64DataPoint, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
