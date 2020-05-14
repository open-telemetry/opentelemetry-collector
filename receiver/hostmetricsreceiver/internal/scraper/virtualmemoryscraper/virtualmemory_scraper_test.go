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

package virtualmemoryscraper

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, pdata.MetricSlice)

func TestScrapeMetrics(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, metrics pdata.MetricSlice) {
		if runtime.GOOS != "windows" {
			assert.Equal(t, 0, metrics.Len())
			return
		}

		// expect 2 metrics
		assert.Equal(t, 2, metrics.Len())

		// expect a single datapoint for page file used metric
		hostSwapUsageMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, metricSwapUsageDescriptor, hostSwapUsageMetric.MetricDescriptor())

		// it's valid for a system to have no paging file, so if no data points were returned, do no validation
		if hostSwapUsageMetric.Int64DataPoints().Len() > 0 {
			assert.GreaterOrEqual(t, hostSwapUsageMetric.Int64DataPoints().Len(), 2)
			internal.AssertInt64MetricLabelExists(t, hostSwapUsageMetric, 0, pageFileLabelName)
			internal.AssertInt64MetricLabelHasValue(t, hostSwapUsageMetric, 0, stateLabelName, usedLabelValue)
			internal.AssertInt64MetricLabelExists(t, hostSwapUsageMetric, 1, pageFileLabelName)
			internal.AssertInt64MetricLabelHasValue(t, hostSwapUsageMetric, 1, stateLabelName, freeLabelValue)
		}

		// expect an in & out datapoint for paging metric with type always equal to major
		pagingPerSecondMetric := metrics.At(1)
		internal.AssertDescriptorEqual(t, metricPagingDescriptor, pagingPerSecondMetric.MetricDescriptor())
		assert.Equal(t, 2, pagingPerSecondMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelHasValue(t, pagingPerSecondMetric, 0, typeLabelName, majorTypeLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, pagingPerSecondMetric, 0, directionLabelName, inDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, pagingPerSecondMetric, 1, typeLabelName, majorTypeLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, pagingPerSecondMetric, 1, directionLabelName, outDirectionLabelValue)
	})
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	scraper := newVirtualMemoryScraper(context.Background(), config)
	err := scraper.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize virtual memory scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	metrics, err := scraper.ScrapeMetrics(context.Background())
	require.NoError(t, err, "Failed to scrape metrics: %v", err)

	assertFn(t, metrics)
}
