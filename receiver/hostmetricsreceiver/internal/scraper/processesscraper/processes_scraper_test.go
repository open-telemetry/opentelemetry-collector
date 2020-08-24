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

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

var systemSpecificMetrics = map[string][]pdata.MetricDescriptor{
	"linux":   {processesRunningDescriptor, processesBlockedDescriptor},
	"darwin":  {processesRunningDescriptor, processesBlockedDescriptor},
	"freebsd": {processesRunningDescriptor, processesBlockedDescriptor},
	"openbsd": {processesRunningDescriptor, processesBlockedDescriptor},
}

func TestScrapeMetrics(t *testing.T) {
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

	expectedMetrics := systemSpecificMetrics[runtime.GOOS]

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newProcessesScraper(context.Background(), &Config{})
			if test.miscFunc != nil {
				scraper.misc = test.miscFunc
			}

			err := scraper.Initialize(context.Background())
			require.NoError(t, err, "Failed to initialize processes scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			metrics, err := scraper.ScrapeMetrics(context.Background())
			if len(expectedMetrics) > 0 && test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			assert.Equal(t, len(expectedMetrics), metrics.Len())
			for i, expectedMetricDescriptor := range expectedMetrics {
				assertProcessesMetricValid(t, metrics.At(i), expectedMetricDescriptor)
			}

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertProcessesMetricValid(t *testing.T, metric pdata.Metric, descriptor pdata.MetricDescriptor) {
	internal.AssertDescriptorEqual(t, descriptor, metric.MetricDescriptor())
	assert.Equal(t, metric.Int64DataPoints().Len(), 1)
	assert.Equal(t, metric.Int64DataPoints().At(0).LabelsMap().Len(), 0)
}
