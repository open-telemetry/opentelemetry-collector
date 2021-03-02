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

package processesscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name        string
		miscFunc    func() (*load.MiscStat, error)
		expectedErr string
	}

	testCases := []testCase{
		{
			name: "Standard",
		},
		{
			name:        "Error",
			miscFunc:    func() (*load.MiscStat, error) { return nil, errors.New("err1") },
			expectedErr: "err1",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			expectProcessesCountMetric := (runtime.GOOS == "linux" || runtime.GOOS == "openbsd" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd")
			expectProcessesCreatedMetric := (runtime.GOOS == "linux" || runtime.GOOS == "openbsd")

			scraper := newProcessesScraper(context.Background(), &Config{})
			if test.miscFunc != nil {
				scraper.misc = test.miscFunc
			}

			err := scraper.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err, "Failed to initialize processes scraper: %v", err)

			metrics, err := scraper.scrape(context.Background())

			expectedMetricCount := 0
			if expectProcessesCountMetric {
				expectedMetricCount++
			}
			if expectProcessesCreatedMetric {
				expectedMetricCount++
			}

			if (expectProcessesCountMetric || expectProcessesCreatedMetric) && test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)

				isPartial := scrapererror.IsPartialScrapeError(err)
				assert.True(t, isPartial)
				if isPartial {
					assert.Equal(t, expectedMetricCount, err.(scrapererror.PartialScrapeError).Failed)
				}

				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			assert.Equal(t, expectedMetricCount, metrics.Len())

			if expectProcessesCountMetric {
				assertProcessesCountMetricValid(t, metrics.At(0))
			}
			if expectProcessesCreatedMetric {
				assertProcessesCreatedMetricValid(t, metrics.At(1))
			}

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertProcessesCountMetricValid(t *testing.T, metric pdata.Metric) {
	internal.AssertDescriptorEqual(t, metadata.Metrics.SystemProcessesCount.New(), metric)
	assert.Equal(t, 2, metric.IntSum().DataPoints().Len())
	internal.AssertIntSumMetricLabelHasValue(t, metric, 0, "status", "running")
	internal.AssertIntSumMetricLabelHasValue(t, metric, 1, "status", "blocked")
}

func assertProcessesCreatedMetricValid(t *testing.T, metric pdata.Metric) {
	internal.AssertDescriptorEqual(t, metadata.Metrics.SystemProcessesCreated.New(), metric)
	assert.Equal(t, 1, metric.IntSum().DataPoints().Len())
	assert.Equal(t, 0, metric.IntSum().DataPoints().At(0).LabelsMap().Len())
}
