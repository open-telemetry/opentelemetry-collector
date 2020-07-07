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

package filesystemscraper

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
		// expect at least 1 metric
		assert.GreaterOrEqual(t, metrics.Len(), 1)

		// for filesystem used metric, expect a used & free datapoint for at least one drive
		fileSystemUsageMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, fileSystemUsageDescriptor, fileSystemUsageMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, fileSystemUsageMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, fileSystemUsageMetric, 0, stateLabelName, usedLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, fileSystemUsageMetric, 1, stateLabelName, freeLabelValue)
	})
}

func TestScrapeMetrics_Unix(t *testing.T) {
	if !isUnix() {
		return
	}

	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, metrics pdata.MetricSlice) {
		// expect 2 metrics
		assert.Equal(t, 2, metrics.Len())

		// for filesystem used metric, expect a used, free & reserved datapoint for at least one drive
		fileSystemUsageMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, fileSystemUsageDescriptor, fileSystemUsageMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, fileSystemUsageMetric.Int64DataPoints().Len(), 3)
		internal.AssertInt64MetricLabelHasValue(t, fileSystemUsageMetric, 0, stateLabelName, usedLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, fileSystemUsageMetric, 1, stateLabelName, freeLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, fileSystemUsageMetric, 2, stateLabelName, reservedLabelValue)

		// for filesystem inodes used metric, expect a used, free & reserved datapoint for at least one drive
		fileSystemINodesUsageMetric := metrics.At(1)
		internal.AssertDescriptorEqual(t, fileSystemINodesUsageDescriptor, fileSystemINodesUsageMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, fileSystemINodesUsageMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, fileSystemINodesUsageMetric, 0, stateLabelName, usedLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, fileSystemINodesUsageMetric, 1, stateLabelName, freeLabelValue)
	})
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	scraper := newFileSystemScraper(context.Background(), config)
	err := scraper.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize file system scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	metrics, err := scraper.ScrapeMetrics(context.Background())
	require.NoError(t, err, "Failed to scrape metrics: %v", err)

	assertFn(t, metrics)
}

func isUnix() bool {
	for _, unixOS := range []string{"linux", "darwin", "freebsd", "openbsd", "solaris"} {
		if runtime.GOOS == unixOS {
			return true
		}
	}

	return false
}
