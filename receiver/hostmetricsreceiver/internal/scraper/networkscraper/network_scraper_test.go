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

package networkscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

func TestScrapeMetrics(t *testing.T) {
	type testCase struct {
		name            string
		ioCountersFunc  func(bool) ([]net.IOCountersStat, error)
		connectionsFunc func(string) ([]net.ConnectionStat, error)
		expectedErr     string
	}

	testCases := []testCase{
		{
			name: "Standard",
		},
		{
			name:           "IOCounters Error",
			ioCountersFunc: func(bool) ([]net.IOCountersStat, error) { return nil, errors.New("err1") },
			expectedErr:    "err1",
		},
		{
			name:            "Connections Error",
			connectionsFunc: func(string) ([]net.ConnectionStat, error) { return nil, errors.New("err2") },
			expectedErr:     "err2",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newNetworkScraper(context.Background(), &Config{})
			if test.ioCountersFunc != nil {
				scraper.ioCounters = test.ioCountersFunc
			}
			if test.connectionsFunc != nil {
				scraper.connections = test.connectionsFunc
			}

			err := scraper.Initialize(context.Background())
			require.NoError(t, err, "Failed to initialize network scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			metrics, err := scraper.ScrapeMetrics(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			assert.Equal(t, 5, metrics.Len())

			assertNetworkIOMetricValid(t, metrics.At(0), networkPacketsDescriptor)
			assertNetworkIOMetricValid(t, metrics.At(1), networkDroppedPacketsDescriptor)
			assertNetworkIOMetricValid(t, metrics.At(2), networkErrorsDescriptor)
			assertNetworkIOMetricValid(t, metrics.At(3), networkIODescriptor)

			assertNetworkTCPConnectionsMetricValid(t, metrics.At(4))
		})
	}
}

func assertNetworkIOMetricValid(t *testing.T, metric pdata.Metric, descriptor pdata.MetricDescriptor) {
	internal.AssertDescriptorEqual(t, descriptor, metric.MetricDescriptor())
	assert.Equal(t, 2, metric.Int64DataPoints().Len())
	internal.AssertInt64MetricLabelHasValue(t, metric, 0, directionLabelName, transmitDirectionLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, metric, 1, directionLabelName, receiveDirectionLabelValue)
}

func assertNetworkTCPConnectionsMetricValid(t *testing.T, metric pdata.Metric) {
	internal.AssertDescriptorEqual(t, networkTCPConnectionsDescriptor, metric.MetricDescriptor())
	internal.AssertInt64MetricLabelExists(t, metric, 0, stateLabelName)
	assert.Equal(t, 12, metric.Int64DataPoints().Len())
}
