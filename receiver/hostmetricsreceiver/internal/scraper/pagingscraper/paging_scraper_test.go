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

package pagingscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name              string
		bootTimeFunc      func() (uint64, error)
		expectedStartTime pdata.Timestamp
		initializationErr string
	}

	testCases := []testCase{
		{
			name: "Standard",
		},
		{
			name:              "Validate Start Time",
			bootTimeFunc:      func() (uint64, error) { return 100, nil },
			expectedStartTime: 100 * 1e9,
		},
		{
			name:              "Boot Time Error",
			bootTimeFunc:      func() (uint64, error) { return 0, errors.New("err1") },
			initializationErr: "err1",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newPagingScraper(context.Background(), &Config{})
			if test.bootTimeFunc != nil {
				scraper.bootTime = test.bootTimeFunc
			}

			err := scraper.start(context.Background(), componenttest.NewNopHost())
			if test.initializationErr != "" {
				assert.EqualError(t, err, test.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize paging scraper: %v", err)

			metrics, err := scraper.scrape(context.Background())
			require.NoError(t, err)

			// expect 3 metrics (windows does not currently support the faults metric)
			expectedMetrics := 3
			if runtime.GOOS == "windows" {
				expectedMetrics = 2
			}
			assert.Equal(t, expectedMetrics, metrics.Len())

			assertPagingUsageMetricValid(t, metrics.At(0))
			internal.AssertSameTimeStampForMetrics(t, metrics, 0, 1)

			assertPagingOperationsMetricValid(t, metrics.At(1), test.expectedStartTime)
			if runtime.GOOS != "windows" {
				assertPageFaultsMetricValid(t, metrics.At(2), test.expectedStartTime)
			}
			internal.AssertSameTimeStampForMetrics(t, metrics, 1, metrics.Len())
		})
	}
}

func assertPagingUsageMetricValid(t *testing.T, hostPagingUsageMetric pdata.Metric) {
	internal.AssertDescriptorEqual(t, metadata.Metrics.SystemPagingUsage.New(), hostPagingUsageMetric)

	// it's valid for a system to have no swap space  / paging file, so if no data points were returned, do no validation
	if hostPagingUsageMetric.IntSum().DataPoints().Len() == 0 {
		return
	}

	// expect at least used, free & cached datapoint
	expectedDataPoints := 3
	// windows does not return a cached datapoint
	if runtime.GOOS == "windows" {
		expectedDataPoints = 2
	}

	assert.GreaterOrEqual(t, hostPagingUsageMetric.IntSum().DataPoints().Len(), expectedDataPoints)
	internal.AssertIntSumMetricLabelHasValue(t, hostPagingUsageMetric, 0, "state", "used")
	internal.AssertIntSumMetricLabelHasValue(t, hostPagingUsageMetric, 1, "state", "free")
	// on non-windows, also expect a cached state label
	if runtime.GOOS != "windows" {
		internal.AssertIntSumMetricLabelHasValue(t, hostPagingUsageMetric, 2, "state", "cached")
	}
	// on windows, also expect the page file device name label
	if runtime.GOOS == "windows" {
		internal.AssertIntSumMetricLabelExists(t, hostPagingUsageMetric, 0, "device")
		internal.AssertIntSumMetricLabelExists(t, hostPagingUsageMetric, 1, "device")
	}
}

func assertPagingOperationsMetricValid(t *testing.T, pagingMetric pdata.Metric, startTime pdata.Timestamp) {
	internal.AssertDescriptorEqual(t, metadata.Metrics.SystemPagingOperations.New(), pagingMetric)
	if startTime != 0 {
		internal.AssertIntSumMetricStartTimeEquals(t, pagingMetric, startTime)
	}

	// expect an in & out datapoint, for both major and minor paging types (windows does not currently support minor paging data)
	expectedDataPoints := 4
	if runtime.GOOS == "windows" {
		expectedDataPoints = 2
	}
	assert.Equal(t, expectedDataPoints, pagingMetric.IntSum().DataPoints().Len())

	internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 0, "type", "major")
	internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 0, "direction", "page_in")
	internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 1, "type", "major")
	internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 1, "direction", "page_out")
	if runtime.GOOS != "windows" {
		internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 2, "type", "minor")
		internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 2, "direction", "page_in")
		internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 3, "type", "minor")
		internal.AssertIntSumMetricLabelHasValue(t, pagingMetric, 3, "direction", "page_out")
	}
}

func assertPageFaultsMetricValid(t *testing.T, pageFaultsMetric pdata.Metric, startTime pdata.Timestamp) {
	internal.AssertDescriptorEqual(t, metadata.Metrics.SystemPagingFaults.New(), pageFaultsMetric)
	if startTime != 0 {
		internal.AssertIntSumMetricStartTimeEquals(t, pageFaultsMetric, startTime)
	}

	assert.Equal(t, 2, pageFaultsMetric.IntSum().DataPoints().Len())
	internal.AssertIntSumMetricLabelHasValue(t, pageFaultsMetric, 0, "type", "major")
	internal.AssertIntSumMetricLabelHasValue(t, pageFaultsMetric, 1, "type", "minor")
}
