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

package loadscraper

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/load"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

// scraper for Load Metrics
type scraper struct {
	logger *zap.Logger
	config *Config

	// for mocking
	load func() (*load.AvgStat, error)
}

// newLoadScraper creates a set of Load related metrics
func newLoadScraper(_ context.Context, logger *zap.Logger, cfg *Config) *scraper {
	return &scraper{logger: logger, config: cfg, load: getSampledLoadAverages}
}

// Initialize
func (s *scraper) Initialize(ctx context.Context) error {
	return startSampling(ctx, s.logger)
}

// Close
func (s *scraper) Close(ctx context.Context) error {
	return stopSampling(ctx)
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(_ context.Context) (dataold.MetricSlice, error) {
	metrics := dataold.NewMetricSlice()

	now := internal.TimeToUnixNano(time.Now())
	avgLoadValues, err := s.load()
	if err != nil {
		return metrics, err
	}

	metrics.Resize(3)
	initializeLoadMetric(metrics.At(0), loadAvg1MDescriptor, now, avgLoadValues.Load1)
	initializeLoadMetric(metrics.At(1), loadAvg5mDescriptor, now, avgLoadValues.Load5)
	initializeLoadMetric(metrics.At(2), loadAvg15mDescriptor, now, avgLoadValues.Load15)
	return metrics, nil
}

func initializeLoadMetric(metric dataold.Metric, metricDescriptor dataold.MetricDescriptor, now pdata.TimestampUnixNano, value float64) {
	metricDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.DoubleDataPoints()
	idps.Resize(1)
	dp := idps.At(0)
	dp.SetTimestamp(now)
	dp.SetValue(value)
}
