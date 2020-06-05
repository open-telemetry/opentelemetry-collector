// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !windows

package virtualmemoryscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapeMetrics_Errors(t *testing.T) {
	type testCase struct {
		name               string
		virtualMemoryError error
		swapMemoryError    error
		expectedError      string
	}

	testCases := []testCase{
		{
			name:               "virtualMemoryError",
			virtualMemoryError: errors.New("err1"),
			expectedError:      "err1",
		},
		{
			name:            "swapMemoryError",
			swapMemoryError: errors.New("err2"),
			expectedError:   "err2",
		},
		{
			name:               "multipleErrors",
			virtualMemoryError: errors.New("err1"),
			swapMemoryError:    errors.New("err2"),
			expectedError:      "[err1; err2]",
		},
	}

	preservedGetVirtualMemory := getVirtualMemory
	preservedGetSwapMemory := getSwapMemory

	scraper := newVirtualMemoryScraper(context.Background(), &Config{})
	err := scraper.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize virtual memory scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.virtualMemoryError != nil {
				getVirtualMemory = func() (*mem.VirtualMemoryStat, error) { return nil, test.virtualMemoryError }
			}
			if test.swapMemoryError != nil {
				getSwapMemory = func() (*mem.SwapMemoryStat, error) { return nil, test.swapMemoryError }
			}

			_, err = scraper.ScrapeMetrics(context.Background())
			assert.EqualError(t, err, test.expectedError)

			getVirtualMemory = preservedGetVirtualMemory
			getSwapMemory = preservedGetSwapMemory
		})
	}
}
