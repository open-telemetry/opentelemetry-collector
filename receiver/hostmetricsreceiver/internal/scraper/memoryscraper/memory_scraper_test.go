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
		// expect 1 metric
		assert.Equal(t, 1, metrics.Len())

		// for memory used metric, expect a datapoint for each state label, including at least 2 states, one of which is 'Used'
		hostMemoryUsedMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, metricMemoryUsedDescriptor, hostMemoryUsedMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, hostMemoryUsedMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, hostMemoryUsedMetric, 0, stateLabelName, usedStateLabelValue)
	})
}

func TestScrapeMetrics_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		return
	}

	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, metrics pdata.MetricSlice) {
		// expect 1 metric
		assert.Equal(t, 1, metrics.Len())

		// for memory used metric, expect a datapoint for all 6 state labels
		hostMemoryUsedMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, metricMemoryUsedDescriptor, hostMemoryUsedMetric.MetricDescriptor())
		assert.Equal(t, 6, hostMemoryUsedMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelHasValue(t, hostMemoryUsedMetric, 0, stateLabelName, usedStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostMemoryUsedMetric, 1, stateLabelName, freeStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostMemoryUsedMetric, 2, stateLabelName, bufferedStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostMemoryUsedMetric, 3, stateLabelName, cachedStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostMemoryUsedMetric, 4, stateLabelName, slabReclaimableStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostMemoryUsedMetric, 5, stateLabelName, slabUnreclaimableStateLabelValue)
	})
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	scraper := newMemoryScraper(context.Background(), config)
	err := scraper.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize memory scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	metrics, err := scraper.ScrapeMetrics(context.Background())
	require.NoError(t, err, "Failed to scrape metrics: %v", err)

	assertFn(t, metrics)
}
