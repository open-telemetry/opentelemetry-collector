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

		// for disk byts metric, expect a receive & transmit datapoint for at least one drive
		hostDiskBytesMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, metricDiskBytesDescriptor, hostDiskBytesMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, hostDiskBytesMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, hostDiskBytesMetric, 0, directionLabelName, receiveDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostDiskBytesMetric, 1, directionLabelName, transmitDirectionLabelValue)

		// for disk operations metric, expect a receive & transmit datapoint for at least one drive
		hostDiskOpsMetric := metrics.At(1)
		internal.AssertDescriptorEqual(t, metricDiskOpsDescriptor, hostDiskOpsMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, hostDiskOpsMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, hostDiskOpsMetric, 0, directionLabelName, receiveDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostDiskOpsMetric, 1, directionLabelName, transmitDirectionLabelValue)

		// for disk time metric, expect a receive & transmit datapoint for at least one drive
		hostDiskTimeMetric := metrics.At(2)
		internal.AssertDescriptorEqual(t, metricDiskTimeDescriptor, hostDiskTimeMetric.MetricDescriptor())
		assert.GreaterOrEqual(t, hostDiskTimeMetric.Int64DataPoints().Len(), 2)
		internal.AssertInt64MetricLabelHasValue(t, hostDiskTimeMetric, 0, directionLabelName, receiveDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostDiskTimeMetric, 1, directionLabelName, transmitDirectionLabelValue)
	})
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
