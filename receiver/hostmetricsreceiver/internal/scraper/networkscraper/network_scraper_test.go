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

package networkscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, pdata.MetricSlice)

func TestScrapeMetrics(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, metrics pdata.MetricSlice) {
		// expect 5 metrics
		assert.Equal(t, 5, metrics.Len())

		// for network packets, dropped packets, errors, & bytes metrics, expect a transmit & receive datapoint
		assertNetworkMetricMatchesDescriptorAndHasTransmitAndReceiveDataPoints(t, metrics.At(0), metricNetworkPacketsDescriptor)
		assertNetworkMetricMatchesDescriptorAndHasTransmitAndReceiveDataPoints(t, metrics.At(1), metricNetworkDroppedPacketsDescriptor)
		assertNetworkMetricMatchesDescriptorAndHasTransmitAndReceiveDataPoints(t, metrics.At(2), metricNetworkErrorsDescriptor)
		assertNetworkMetricMatchesDescriptorAndHasTransmitAndReceiveDataPoints(t, metrics.At(3), metricNetworkBytesDescriptor)

		// for tcp connections metric, expect at least one datapoint with a state label
		hostNetworkTCPConnectionsMetric := metrics.At(4)
		internal.AssertDescriptorEqual(t, metricNetworkTCPConnectionDescriptor, hostNetworkTCPConnectionsMetric.MetricDescriptor())
		internal.AssertInt64MetricLabelExists(t, hostNetworkTCPConnectionsMetric, 0, stateLabelName)
		assert.GreaterOrEqual(t, hostNetworkTCPConnectionsMetric.Int64DataPoints().Len(), 1)
	})
}

func assertNetworkMetricMatchesDescriptorAndHasTransmitAndReceiveDataPoints(t *testing.T, metric pdata.Metric, expectedDescriptor pdata.MetricDescriptor) {
	internal.AssertDescriptorEqual(t, expectedDescriptor, metric.MetricDescriptor())
	assert.Equal(t, 2, metric.Int64DataPoints().Len())
	internal.AssertInt64MetricLabelHasValue(t, metric, 0, directionLabelName, transmitDirectionLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, metric, 1, directionLabelName, receiveDirectionLabelValue)
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	scraper := newNetworkScraper(context.Background(), config)
	err := scraper.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize network scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	metrics, err := scraper.ScrapeMetrics(context.Background())
	require.NoError(t, err, "Failed to scrape metrics: %v", err)

	assertFn(t, metrics)
}
