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

// +build !windows

package diskscraper

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestScrapeMetrics_Others(t *testing.T) {
	type testCase struct {
		name              string
		bootTimeFunc      func() (uint64, error)
		ioCountersFunc    func(names ...string) (map[string]disk.IOCountersStat, error)
		expectedStartTime pdata.TimestampUnixNano
		initializationErr string
		expectedErr       string
	}

	testCases := []testCase{
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
		{
			name:           "Error",
			ioCountersFunc: func(names ...string) (map[string]disk.IOCountersStat, error) { return nil, errors.New("err2") },
			expectedErr:    "err2",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newDiskScraper(context.Background(), &Config{})
			require.NoError(t, err, "Failed to create disk scraper: %v", err)

			if test.bootTimeFunc != nil {
				scraper.bootTime = test.bootTimeFunc
			}
			if test.ioCountersFunc != nil {
				scraper.ioCounters = test.ioCountersFunc
			}

			err = scraper.Initialize(context.Background())
			if test.initializationErr != "" {
				assert.EqualError(t, err, test.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize disk scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			metrics, err := scraper.ScrapeMetrics(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}

			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			assertInt64DiskMetricValid(t, metrics.At(0), diskIODescriptor, test.expectedStartTime)
			assertInt64DiskMetricValid(t, metrics.At(1), diskOpsDescriptor, test.expectedStartTime)
			assertDoubleDiskMetricValid(t, metrics.At(2), diskTimeDescriptor, test.expectedStartTime)
			assertDiskPendingOperationsMetricValid(t, metrics.At(3))

			if runtime.GOOS == "linux" {
				assertInt64DiskMetricValid(t, metrics.At(4), diskMergedDescriptor, test.expectedStartTime)
			}
		})
	}
}
