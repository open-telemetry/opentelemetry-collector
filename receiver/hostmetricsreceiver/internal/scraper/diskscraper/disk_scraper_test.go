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

package diskscraper

import (
	"context"
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

		// expect 3 metrics
		assert.Equal(t, 3, metrics.Len())

		// for disk byts metric, expect a read & write datapoint for at least one drive
		assertDiskMetricMatchesDescriptorAndHasReadAndWriteDataPoints(t, metrics.At(0), metricDiskBytesDescriptor)
		assertDiskMetricMatchesDescriptorAndHasReadAndWriteDataPoints(t, metrics.At(1), metricDiskOpsDescriptor)
		assertDiskMetricMatchesDescriptorAndHasReadAndWriteDataPoints(t, metrics.At(2), metricDiskTimeDescriptor)
	})
}

func assertDiskMetricMatchesDescriptorAndHasReadAndWriteDataPoints(t *testing.T, metric pdata.Metric, expectedDescriptor pdata.MetricDescriptor) {
	internal.AssertDescriptorEqual(t, expectedDescriptor, metric.MetricDescriptor())
	assert.GreaterOrEqual(t, metric.Int64DataPoints().Len(), 2)
	internal.AssertInt64MetricLabelHasValue(t, metric, 0, directionLabelName, readDirectionLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, metric, 1, directionLabelName, writeDirectionLabelValue)
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	config.SetCollectionInterval(5 * time.Millisecond)

	sink := &exportertest.SinkMetricsExporter{}

	scraper, err := NewDiskScraper(context.Background(), config, sink)
	require.NoError(t, err, "Failed to create disk scraper: %v", err)

	err = scraper.Start(context.Background())
	require.NoError(t, err, "Failed to start disk scraper: %v", err)
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
