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

package memoryscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

func TestScrapeMetrics(t *testing.T) {
	type testCase struct {
		name              string
		virtualMemoryFunc func() (*mem.VirtualMemoryStat, error)
		expectedErr       string
	}

	testCases := []testCase{
		{
			name: "Standard",
		},
		{
			name:              "Error",
			virtualMemoryFunc: func() (*mem.VirtualMemoryStat, error) { return nil, errors.New("err1") },
			expectedErr:       "err1",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newMemoryScraper(context.Background(), &Config{})
			if test.virtualMemoryFunc != nil {
				scraper.virtualMemory = test.virtualMemoryFunc
			}

			err := scraper.Initialize(context.Background())
			require.NoError(t, err, "Failed to initialize memory scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			metrics, err := scraper.ScrapeMetrics(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			assert.Equal(t, 1, metrics.Len())

			assertMemoryUsageMetricValid(t, metrics.At(0), metadata.Metrics.SystemMemoryUsage)

			if runtime.GOOS == "linux" {
				assertMemoryUsageMetricHasLinuxSpecificStateLabels(t, metrics.At(0))
			} else if runtime.GOOS != "windows" {
				internal.AssertIntSumMetricLabelHasValue(t, metrics.At(0), 2, metadata.Labels.MemState, metadata.LabelMemState.Inactive)
			}

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertMemoryUsageMetricValid(t *testing.T, metric pdata.Metric, descriptor pdata.Metric) {
	internal.AssertDescriptorEqual(t, descriptor, metric)
	assert.GreaterOrEqual(t, metric.IntSum().DataPoints().Len(), 2)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 0, metadata.Labels.MemState, metadata.LabelMemState.Used)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 1, metadata.Labels.MemState, metadata.LabelMemState.Free)
}

func assertMemoryUsageMetricHasLinuxSpecificStateLabels(t *testing.T, metric pdata.Metric) {
	internal.AssertIntSumMetricLabelHasValue(t, metric, 2, metadata.Labels.MemState, metadata.LabelMemState.Buffered)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 3, metadata.Labels.MemState, metadata.LabelMemState.Cached)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 4, metadata.Labels.MemState, metadata.LabelMemState.SlabReclaimable)
	internal.AssertIntSumMetricLabelHasValue(t, metric, 5, metadata.Labels.MemState, metadata.LabelMemState.SlabUnreclaimable)
}
