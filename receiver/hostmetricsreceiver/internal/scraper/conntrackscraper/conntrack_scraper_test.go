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

package conntrackscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

func TestScrape(t *testing.T) {
	type testCase struct {
		name        string
		scraperFunc func() ([]net.FilterStat, error)
		expectedErr string
	}

	testCases := []testCase{
		{
			name: "Standard",
			scraperFunc: func() ([]net.FilterStat, error) {
				return []net.FilterStat{{ConnTrackCount: 1, ConnTrackMax: 2}}, nil
			},
			expectedErr: "",
		},
		{
			name:        "Error",
			scraperFunc: func() ([]net.FilterStat, error) { return nil, errors.New("err1") },
			expectedErr: "err1",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			expectConntrackMetrics := runtime.GOOS == "linux"
			scraper := newConntrackScraper(context.Background(), &Config{})
			if test.scraperFunc != nil {
				scraper.conntrack = test.scraperFunc
			}

			err := scraper.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err, "Failed to initialize conntrack scraper: %v", err)

			metrics, err := scraper.scrape(context.Background())

			expectedMetricCount := 0
			if expectConntrackMetrics {
				expectedMetricCount += 2
			}

			if expectConntrackMetrics && test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)
			assert.Equal(t, expectedMetricCount, metrics.Len())

			if expectConntrackMetrics {
				assertNetworkConntrackMetricValid(t, metrics.At(0), metadata.Metrics.SystemConntrackCount.New())
				assertNetworkConntrackMetricValid(t, metrics.At(1), metadata.Metrics.SystemConntrackMax.New())
			}
			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertNetworkConntrackMetricValid(t *testing.T, metric pdata.Metric, descriptor pdata.Metric) {
	internal.AssertDescriptorEqual(t, descriptor, metric)
	assert.Equal(t, 1, metric.IntSum().DataPoints().Len())
}
