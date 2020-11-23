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

package networkscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name                 string
		config               Config
		bootTimeFunc         func() (uint64, error)
		ioCountersFunc       func(bool) ([]net.IOCountersStat, error)
		connectionsFunc      func(string) ([]net.ConnectionStat, error)
		expectNetworkMetrics bool
		expectedStartTime    pdata.TimestampUnixNano
		newErrRegex          string
		initializationErr    string
		expectedErr          string
		expectedErrCount     int
	}

	testCases := []testCase{
		{
			name:                 "Standard",
			expectNetworkMetrics: true,
		},
		{
			name:                 "Validate Start Time",
			bootTimeFunc:         func() (uint64, error) { return 100, nil },
			expectNetworkMetrics: true,
			expectedStartTime:    100 * 1e9,
		},
		{
			name:                 "Include Filter that matches nothing",
			config:               Config{Include: MatchConfig{filterset.Config{MatchType: "strict"}, []string{"@*^#&*$^#)"}}},
			expectNetworkMetrics: false,
		},
		{
			name:        "Invalid Include Filter",
			config:      Config{Include: MatchConfig{Interfaces: []string{"test"}}},
			newErrRegex: "^error creating network interface include filters:",
		},
		{
			name:        "Invalid Exclude Filter",
			config:      Config{Exclude: MatchConfig{Interfaces: []string{"test"}}},
			newErrRegex: "^error creating network interface exclude filters:",
		},
		{
			name:              "Boot Time Error",
			bootTimeFunc:      func() (uint64, error) { return 0, errors.New("err1") },
			initializationErr: "err1",
		},
		{
			name:             "IOCounters Error",
			ioCountersFunc:   func(bool) ([]net.IOCountersStat, error) { return nil, errors.New("err2") },
			expectedErr:      "err2",
			expectedErrCount: networkMetricsLen,
		},
		{
			name:             "Connections Error",
			connectionsFunc:  func(string) ([]net.ConnectionStat, error) { return nil, errors.New("err3") },
			expectedErr:      "err3",
			expectedErrCount: connectionsMetricsLen,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newNetworkScraper(context.Background(), &test.config)
			if test.newErrRegex != "" {
				require.Error(t, err)
				require.Regexp(t, test.newErrRegex, err)
				return
			}
			require.NoError(t, err, "Failed to create network scraper: %v", err)

			if test.bootTimeFunc != nil {
				scraper.bootTime = test.bootTimeFunc
			}
			if test.ioCountersFunc != nil {
				scraper.ioCounters = test.ioCountersFunc
			}
			if test.connectionsFunc != nil {
				scraper.connections = test.connectionsFunc
			}

			err = scraper.start(context.Background(), componenttest.NewNopHost())
			if test.initializationErr != "" {
				assert.EqualError(t, err, test.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize network scraper: %v", err)

			metrics, err := scraper.scrape(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)

				isPartial := consumererror.IsPartialScrapeError(err)
				assert.True(t, isPartial)
				if isPartial {
					assert.Equal(t, test.expectedErrCount, err.(consumererror.PartialScrapeError).Failed)
				}

				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			expectedMetricCount := 1
			if test.expectNetworkMetrics {
				expectedMetricCount += 4
			}
			assert.Equal(t, expectedMetricCount, metrics.Len())

			idx := 0
			if test.expectNetworkMetrics {
				assertNetworkIOMetricValid(t, metrics.At(idx+0), networkPacketsDescriptor, test.expectedStartTime)
				assertNetworkIOMetricValid(t, metrics.At(idx+1), networkDroppedPacketsDescriptor, test.expectedStartTime)
				assertNetworkIOMetricValid(t, metrics.At(idx+2), networkErrorsDescriptor, test.expectedStartTime)
				assertNetworkIOMetricValid(t, metrics.At(idx+3), networkIODescriptor, test.expectedStartTime)
				internal.AssertSameTimeStampForMetrics(t, metrics, 0, 4)
				idx += 4
			}

			assertNetworkTCPConnectionsMetricValid(t, metrics.At(idx+0))
			internal.AssertSameTimeStampForMetrics(t, metrics, idx, idx+1)
		})
	}
}

func assertNetworkIOMetricValid(t *testing.T, metric pdata.Metric, descriptor pdata.Metric, startTime pdata.TimestampUnixNano) {
	internal.AssertDescriptorEqual(t, descriptor, metric)
	if startTime != 0 {
		internal.AssertIntSumMetricStartTimeEquals(t, metric, startTime)
	}
	assert.GreaterOrEqual(t, metric.IntSum().DataPoints().Len(), 2)
	internal.AssertIntSumMetricLabelExists(t, metric, 0, interfaceLabelName)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 0, directionLabelName, transmitDirectionLabelValue)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 1, directionLabelName, receiveDirectionLabelValue)
}

func assertNetworkTCPConnectionsMetricValid(t *testing.T, metric pdata.Metric) {
	internal.AssertDescriptorEqual(t, networkTCPConnectionsDescriptor, metric)
	internal.AssertIntSumMetricLabelExists(t, metric, 0, stateLabelName)
	assert.Equal(t, 12, metric.IntSum().DataPoints().Len())
}
