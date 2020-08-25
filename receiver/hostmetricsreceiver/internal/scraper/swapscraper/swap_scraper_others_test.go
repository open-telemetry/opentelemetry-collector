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

package swapscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestScrapeMetrics_Errors(t *testing.T) {
	type testCase struct {
		name              string
		bootTimeFunc      func() (uint64, error)
		virtualMemoryFunc func() (*mem.VirtualMemoryStat, error)
		swapMemoryFunc    func() (*mem.SwapMemoryStat, error)
		expectedStartTime pdata.TimestampUnixNano
		initializationErr string
		expectedError     string
	}

	testCases := []testCase{
		{
			name:              "bootTimeError",
			bootTimeFunc:      func() (uint64, error) { return 0, errors.New("err1") },
			initializationErr: "err1",
		},
		{
			name:              "virtualMemoryError",
			virtualMemoryFunc: func() (*mem.VirtualMemoryStat, error) { return nil, errors.New("err1") },
			expectedError:     "err1",
		},
		{
			name:           "swapMemoryError",
			swapMemoryFunc: func() (*mem.SwapMemoryStat, error) { return nil, errors.New("err2") },
			expectedError:  "err2",
		},
		{
			name:              "multipleErrors",
			virtualMemoryFunc: func() (*mem.VirtualMemoryStat, error) { return nil, errors.New("err1") },
			swapMemoryFunc:    func() (*mem.SwapMemoryStat, error) { return nil, errors.New("err2") },
			expectedError:     "[err1; err2]",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newSwapScraper(context.Background(), &Config{})
			if test.bootTimeFunc != nil {
				scraper.bootTime = test.bootTimeFunc
			}
			if test.virtualMemoryFunc != nil {
				scraper.virtualMemory = test.virtualMemoryFunc
			}
			if test.swapMemoryFunc != nil {
				scraper.swapMemory = test.swapMemoryFunc
			}

			err := scraper.Initialize(context.Background())
			if test.initializationErr != "" {
				assert.EqualError(t, err, test.initializationErr)
				return
			}
			require.NoError(t, err, "Failed to initialize swap scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			metrics, err := scraper.ScrapeMetrics(context.Background())
			if test.expectedError != "" {
				assert.EqualError(t, err, test.expectedError)
				return
			}

			assert.Equal(t, 3, metrics.Len())
			assertSwapUsageMetricValid(t, metrics.At(0))
			assertPagingMetricValid(t, metrics.At(1), test.expectedStartTime)
			assertPageFaultsMetricValid(t, metrics.At(2), test.expectedStartTime)
		})
	}
}
