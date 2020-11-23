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

package diskscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name              string
		config            Config
		bootTimeFunc      func() (uint64, error)
		newErrRegex       string
		initializationErr string
		expectMetrics     bool
		expectedStartTime pdata.TimestampUnixNano
	}

	testCases := []testCase{
		{
			name:          "Standard",
			expectMetrics: true,
		},
		{
			name:              "Validate Start Time",
			bootTimeFunc:      func() (uint64, error) { return 100, nil },
			expectMetrics:     true,
			expectedStartTime: 100 * 1e9,
		},
		{
			name:              "Boot Time Error",
			bootTimeFunc:      func() (uint64, error) { return 0, errors.New("err1") },
			initializationErr: "err1",
		},
		{
			name:          "Include Filter that matches nothing",
			config:        Config{Include: MatchConfig{filterset.Config{MatchType: "strict"}, []string{"@*^#&*$^#)"}}},
			expectMetrics: false,
		},
		{
			name:        "Invalid Include Filter",
			config:      Config{Include: MatchConfig{Devices: []string{"test"}}},
			newErrRegex: "^error creating device include filters:",
		},
		{
			name:        "Invalid Exclude Filter",
			config:      Config{Exclude: MatchConfig{Devices: []string{"test"}}},
			newErrRegex: "^error creating device exclude filters:",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newDiskScraper(context.Background(), &test.config)
			if test.newErrRegex != "" {
				require.Error(t, err)
				require.Regexp(t, test.newErrRegex, err)
				return
			}
			require.NoError(t, err, "Failed to create disk scraper: %v", err)

			if test.bootTimeFunc != nil {
				scraper.bootTime = test.bootTimeFunc
			}

			err = scraper.start(context.Background(), componenttest.NewNopHost())
			if test.initializationErr != "" {
				assert.EqualError(t, err, test.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize disk scraper: %v", err)

			metrics, err := scraper.scrape(context.Background())
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			if !test.expectMetrics {
				assert.Equal(t, 0, metrics.Len())
				return
			}

			assert.GreaterOrEqual(t, metrics.Len(), 4)

			assertInt64DiskMetricValid(t, metrics.At(0), diskIODescriptor, true, test.expectedStartTime)
			assertInt64DiskMetricValid(t, metrics.At(1), diskOpsDescriptor, true, test.expectedStartTime)
			assertDoubleDiskMetricValid(t, metrics.At(2), diskIOTimeDescriptor, false, test.expectedStartTime)
			assertDoubleDiskMetricValid(t, metrics.At(3), diskOperationTimeDescriptor, true, test.expectedStartTime)
			assertDiskPendingOperationsMetricValid(t, metrics.At(4))

			if runtime.GOOS == "linux" {
				assertInt64DiskMetricValid(t, metrics.At(5), diskMergedDescriptor, true, test.expectedStartTime)
			}

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertInt64DiskMetricValid(t *testing.T, metric pdata.Metric, expectedDescriptor pdata.Metric, expectDirectionLabels bool, startTime pdata.TimestampUnixNano) {
	internal.AssertDescriptorEqual(t, expectedDescriptor, metric)
	if startTime != 0 {
		internal.AssertIntSumMetricStartTimeEquals(t, metric, startTime)
	}

	minExpectedPoints := 1
	if expectDirectionLabels {
		minExpectedPoints = 2
	}
	assert.GreaterOrEqual(t, metric.IntSum().DataPoints().Len(), minExpectedPoints)

	internal.AssertIntSumMetricLabelExists(t, metric, 0, deviceLabelName)
	if expectDirectionLabels {
		internal.AssertIntSumMetricLabelHasValue(t, metric, 0, directionLabelName, readDirectionLabelValue)
		internal.AssertIntSumMetricLabelHasValue(t, metric, 1, directionLabelName, writeDirectionLabelValue)
	}
}

func assertDoubleDiskMetricValid(t *testing.T, metric pdata.Metric, expectedDescriptor pdata.Metric, expectDirectionLabels bool, startTime pdata.TimestampUnixNano) {
	internal.AssertDescriptorEqual(t, expectedDescriptor, metric)
	if startTime != 0 {
		internal.AssertDoubleSumMetricStartTimeEquals(t, metric, startTime)
	}

	minExpectedPoints := 1
	if expectDirectionLabels {
		minExpectedPoints = 2
	}
	assert.GreaterOrEqual(t, metric.DoubleSum().DataPoints().Len(), minExpectedPoints)

	internal.AssertDoubleSumMetricLabelExists(t, metric, 0, deviceLabelName)
	if expectDirectionLabels {
		internal.AssertDoubleSumMetricLabelHasValue(t, metric, 0, directionLabelName, readDirectionLabelValue)
		internal.AssertDoubleSumMetricLabelHasValue(t, metric, metric.DoubleSum().DataPoints().Len()-1, directionLabelName, writeDirectionLabelValue)
	}
}

func assertDiskPendingOperationsMetricValid(t *testing.T, metric pdata.Metric) {
	internal.AssertDescriptorEqual(t, diskPendingOperationsDescriptor, metric)
	assert.GreaterOrEqual(t, metric.IntSum().DataPoints().Len(), 1)
	internal.AssertIntSumMetricLabelExists(t, metric, 0, deviceLabelName)
}
