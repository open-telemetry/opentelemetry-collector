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

package filesystemscraper

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, []pdata.Metrics)

func TestScrapeMetrics(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect at least 1 metric
		assert.GreaterOrEqual(t, metrics.Len(), 1)

		// for filesystem used metric, expect a used & free datapoint for at least one drive
		hostFileSystemUsedMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, metricFilesystemUsedDescriptor, hostFileSystemUsedMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, hostFileSystemUsedMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, hostFileSystemUsedMetric, 0, stateLabelName, usedLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostFileSystemUsedMetric, 1, stateLabelName, freeLabelValue)
	})
}

func TestScrapeMetrics_Unux(t *testing.T) {
	if !isUnix() {
		return
	}

	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect 2 metrics
		assert.Equal(t, 2, metrics.Len())

		// for filesystem used metric, expect a used, free & reserved datapoint for at least one drive
		hostFileSystemUsedMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, metricFilesystemUsedDescriptor, hostFileSystemUsedMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, hostFileSystemUsedMetric.Int64DataPoints().Len(), 3)
		internal.AssertInt64MetricLabelHasValue(t, hostFileSystemUsedMetric, 0, stateLabelName, usedLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostFileSystemUsedMetric, 1, stateLabelName, freeLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostFileSystemUsedMetric, 2, stateLabelName, reservedLabelValue)

		// for filesystem inodes used metric, expect a used, free & reserved datapoint for at least one drive
		hostFileSystemINodesUsedMetric := metrics.At(1)
		internal.AssertDescriptorEqual(t, metricFilesystemINodesUsedDescriptor, hostFileSystemINodesUsedMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, hostFileSystemINodesUsedMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, hostFileSystemINodesUsedMetric, 0, stateLabelName, usedLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostFileSystemINodesUsedMetric, 1, stateLabelName, freeLabelValue)
	})
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	config.SetCollectionInterval(5 * time.Millisecond)

	sink := &exportertest.SinkMetricsExporter{}

	scraper, err := NewFileSystemScraper(context.Background(), config, sink)
	require.NoError(t, err, "Failed to create filesystem scraper: %v", err)

	err = scraper.Start(context.Background())
	require.NoError(t, err, "Failed to start filesystem scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Shutdown(context.Background())) }()

	require.Eventually(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		assertFn(t, got)
		return true
	}, time.Second, 2*time.Millisecond, "No metrics were collected")
}

func isUnix() bool {
	for _, unixOS := range []string{"linux", "darwin", "freebsd", "openbsd", "solaris"} {
		if runtime.GOOS == unixOS {
			return true
		}
	}

	return false
}
