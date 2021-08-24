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
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/pagingscraper/internal/metadata"
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
	if hostPagingUsageMetric.Sum().DataPoints().Len() == 0 {
		return
	}

	// expect at least used, free & cached datapoint
	expectedDataPoints := 3
	// windows does not return a cached datapoint
	if runtime.GOOS == "windows" {
		expectedDataPoints = 2
	}

	assert.GreaterOrEqual(t, hostPagingUsageMetric.Sum().DataPoints().Len(), expectedDataPoints)
	internal.AssertSumMetricHasAttributeValue(t, hostPagingUsageMetric, 0, "state", pdata.NewAttributeValueString(metadata.LabelState.Used))
	internal.AssertSumMetricHasAttributeValue(t, hostPagingUsageMetric, 1, "state", pdata.NewAttributeValueString(metadata.LabelState.Free))
	// on non-windows, also expect a cached state label
	if runtime.GOOS != "windows" {
		internal.AssertSumMetricHasAttributeValue(t, hostPagingUsageMetric, 2, "state", pdata.NewAttributeValueString(metadata.LabelState.Cached))
	}
	// on windows, also expect the page file device name label
	if runtime.GOOS == "windows" {
		internal.AssertSumMetricHasAttribute(t, hostPagingUsageMetric, 0, "device")
		internal.AssertSumMetricHasAttribute(t, hostPagingUsageMetric, 1, "device")
	}
}

func assertPagingOperationsMetricValid(t *testing.T, pagingMetric pdata.Metric, startTime pdata.Timestamp) {
	internal.AssertDescriptorEqual(t, metadata.Metrics.SystemPagingOperations.New(), pagingMetric)
	if startTime != 0 {
		internal.AssertSumMetricStartTimeEquals(t, pagingMetric, startTime)
	}

	// expect an in & out datapoint, for both major and minor paging types (windows does not currently support minor paging data)
	expectedDataPoints := 4
	if runtime.GOOS == "windows" {
		expectedDataPoints = 2
	}
	assert.Equal(t, expectedDataPoints, pagingMetric.Sum().DataPoints().Len())

	internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 0, "type", pdata.NewAttributeValueString(metadata.LabelType.Major))
	internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 0, "direction", pdata.NewAttributeValueString(metadata.LabelDirection.PageIn))
	internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 1, "type", pdata.NewAttributeValueString(metadata.LabelType.Major))
	internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 1, "direction", pdata.NewAttributeValueString(metadata.LabelDirection.PageOut))
	if runtime.GOOS != "windows" {
		internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 2, "type", pdata.NewAttributeValueString(metadata.LabelType.Minor))
		internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 2, "direction", pdata.NewAttributeValueString(metadata.LabelDirection.PageIn))
		internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 3, "type", pdata.NewAttributeValueString(metadata.LabelType.Minor))
		internal.AssertSumMetricHasAttributeValue(t, pagingMetric, 3, "direction", pdata.NewAttributeValueString(metadata.LabelDirection.PageOut))
	}
}

func assertPageFaultsMetricValid(t *testing.T, pageFaultsMetric pdata.Metric, startTime pdata.Timestamp) {
	internal.AssertDescriptorEqual(t, metadata.Metrics.SystemPagingFaults.New(), pageFaultsMetric)
	if startTime != 0 {
		internal.AssertSumMetricStartTimeEquals(t, pageFaultsMetric, startTime)
	}

	assert.Equal(t, 2, pageFaultsMetric.Sum().DataPoints().Len())
	internal.AssertSumMetricHasAttributeValue(t, pageFaultsMetric, 0, "type", pdata.NewAttributeValueString(metadata.LabelType.Major))
	internal.AssertSumMetricHasAttributeValue(t, pageFaultsMetric, 1, "type", pdata.NewAttributeValueString(metadata.LabelType.Minor))
}
