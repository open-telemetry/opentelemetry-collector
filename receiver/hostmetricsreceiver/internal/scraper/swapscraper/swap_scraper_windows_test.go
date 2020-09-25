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

// +build windows

package swapscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/windows/pdh"
)

func TestScrapeMetrics_Errors(t *testing.T) {
	type testCase struct {
		name                               string
		pageSize                           uint64
		getPageFileStats                   func() ([]*pageFileData, error)
		pageReadsPerSecCounterReturnValue  interface{}
		pageWritesPerSecCounterReturnValue interface{}
		expectedError                      string
		expectedUsedValue                  int64
		expectedFreeValue                  int64
	}

	testPageSize := uint64(4096)
	testPageFileData := &pageFileData{usedPages: 100, totalPages: 300}

	testCases := []testCase{
		{
			name:     "standard",
			pageSize: testPageSize,
			getPageFileStats: func() ([]*pageFileData, error) {
				return []*pageFileData{testPageFileData}, nil
			},
			expectedUsedValue: int64(testPageFileData.usedPages * testPageSize),
			expectedFreeValue: int64((testPageFileData.totalPages - testPageFileData.usedPages) * testPageSize),
		},
		{
			name:             "pageFileError",
			getPageFileStats: func() ([]*pageFileData, error) { return nil, errors.New("err1") },
			expectedError:    "err1",
		},
		{
			name:                              "readsPerSecCounterError",
			pageReadsPerSecCounterReturnValue: errors.New("err2"),
			expectedError:                     "err2",
		},
		{
			name:                               "writesPerSecCounterError",
			pageReadsPerSecCounterReturnValue:  float64(100),
			pageWritesPerSecCounterReturnValue: errors.New("err3"),
			expectedError:                      "err3",
		},
		{
			name:                              "multipleErrors",
			getPageFileStats:                  func() ([]*pageFileData, error) { return nil, errors.New("err1") },
			pageReadsPerSecCounterReturnValue: errors.New("err2"),
			expectedError:                     "[err1; err2]",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newSwapScraper(context.Background(), &Config{})
			if test.getPageFileStats != nil {
				scraper.pageFileStats = test.getPageFileStats
			}
			if test.pageSize > 0 {
				scraper.pageSize = test.pageSize
			} else {
				assert.Greater(t, pageSize, uint64(0))
				assert.Zero(t, pageSize%4096) // page size on Windows should always be a multiple of 4KB
			}

			err := scraper.Initialize(context.Background())
			require.NoError(t, err, "Failed to initialize swap scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			scraper.pageReadsPerSecCounter = pdh.NewMockPerfCounter(test.pageReadsPerSecCounterReturnValue)
			scraper.pageWritesPerSecCounter = pdh.NewMockPerfCounter(test.pageWritesPerSecCounterReturnValue)

			metrics, err := scraper.ScrapeMetrics(context.Background())
			if test.expectedError != "" {
				assert.EqualError(t, err, test.expectedError)
				return
			}

			swapUsageMetric := metrics.At(0)
			assert.Equal(t, test.expectedUsedValue, swapUsageMetric.IntSum().DataPoints().At(0).Value())
			assert.Equal(t, test.expectedFreeValue, swapUsageMetric.IntSum().DataPoints().At(1).Value())
		})
	}
}
