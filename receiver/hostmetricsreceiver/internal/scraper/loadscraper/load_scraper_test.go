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

package loadscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, pdata.MetricSlice)

func TestScrapeMetrics(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, metrics pdata.MetricSlice) {
		// expect 3 metrics
		assert.Equal(t, 3, metrics.Len())

		// expect a single datapoint for 1m, 5m & 15m load metrics
		assertMetricHasSingleDatapoint(t, metrics.At(0), metric1MLoadDescriptor)
		assertMetricHasSingleDatapoint(t, metrics.At(1), metric5MLoadDescriptor)
		assertMetricHasSingleDatapoint(t, metrics.At(2), metric15MLoadDescriptor)
	})
}

func assertMetricHasSingleDatapoint(t *testing.T, metric pdata.Metric, descriptor pdata.MetricDescriptor) {
	internal.AssertDescriptorEqual(t, descriptor, metric.MetricDescriptor())
	assert.Equal(t, 1, metric.DoubleDataPoints().Len())
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	scraper := newLoadScraper(context.Background(), zap.NewNop(), config)
	err := scraper.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize load scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	metrics, err := scraper.ScrapeMetrics(context.Background())
	require.NoError(t, err, "Failed to scrape metrics: %v", err)

	assertFn(t, metrics)
}
