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

package loadscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name        string
		loadFunc    func() (*load.AvgStat, error)
		expectedErr string
	}

	testCases := []testCase{
		{
			name: "Standard",
		},
		{
			name:        "Load Error",
			loadFunc:    func() (*load.AvgStat, error) { return nil, errors.New("err1") },
			expectedErr: "err1",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newLoadScraper(context.Background(), zap.NewNop(), &Config{})
			if test.loadFunc != nil {
				scraper.load = test.loadFunc
			}

			err := scraper.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err, "Failed to initialize load scraper: %v", err)
			defer func() { assert.NoError(t, scraper.shutdown(context.Background())) }()

			metrics, err := scraper.scrape(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)

				isPartial := scrapererror.IsPartialScrapeError(err)
				assert.True(t, isPartial)
				if isPartial {
					assert.Equal(t, metricsLen, err.(scrapererror.PartialScrapeError).Failed)
				}

				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			// expect 3 metrics
			assert.Equal(t, 3, metrics.Len())

			// expect a single datapoint for 1m, 5m & 15m load metrics
			assertMetricHasSingleDatapoint(t, metrics.At(0), metadata.Metrics.SystemCPULoadAverage1m.New())
			assertMetricHasSingleDatapoint(t, metrics.At(1), metadata.Metrics.SystemCPULoadAverage5m.New())
			assertMetricHasSingleDatapoint(t, metrics.At(2), metadata.Metrics.SystemCPULoadAverage15m.New())

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertMetricHasSingleDatapoint(t *testing.T, metric pdata.Metric, descriptor pdata.Metric) {
	internal.AssertDescriptorEqual(t, descriptor, metric)
	assert.Equal(t, 1, metric.DoubleGauge().DataPoints().Len())
}
