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

package pagingscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/perfcounters"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

func TestScrape_Errors(t *testing.T) {
	type testCase struct {
		name              string
		pageSize          uint64
		getPageFileStats  func() ([]*pageFileData, error)
		scrapeErr         error
		getObjectErr      error
		getValuesErr      error
		expectedErr       string
		expectedErrCount  int
		expectedUsedValue int64
		expectedFreeValue int64
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
			expectedErr:      "err1",
			expectedErrCount: pagingUsageMetricsLen,
		},
		{
			name:             "scrapeError",
			scrapeErr:        errors.New("err1"),
			expectedErr:      "err1",
			expectedErrCount: pagingMetricsLen,
		},
		{
			name:             "getObjectErr",
			getObjectErr:     errors.New("err1"),
			expectedErr:      "err1",
			expectedErrCount: pagingMetricsLen,
		},
		{
			name:             "getValuesErr",
			getValuesErr:     errors.New("err1"),
			expectedErr:      "err1",
			expectedErrCount: pagingMetricsLen,
		},
		{
			name:             "multipleErrors",
			getPageFileStats: func() ([]*pageFileData, error) { return nil, errors.New("err1") },
			getObjectErr:     errors.New("err2"),
			expectedErr:      "[err1; err2]",
			expectedErrCount: pagingUsageMetricsLen + pagingMetricsLen,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper := newPagingScraper(context.Background(), &Config{})
			if test.getPageFileStats != nil {
				scraper.pageFileStats = test.getPageFileStats
			}
			if test.pageSize > 0 {
				scraper.pageSize = test.pageSize
			} else {
				assert.Greater(t, pageSize, uint64(0))
				assert.Zero(t, pageSize%4096) // page size on Windows should always be a multiple of 4KB
			}
			scraper.perfCounterScraper = perfcounters.NewMockPerfCounterScraperError(test.scrapeErr, test.getObjectErr, test.getValuesErr)

			err := scraper.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err, "Failed to initialize paging scraper: %v", err)

			metrics, err := scraper.scrape(context.Background())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)

				isPartial := scrapererror.IsPartialScrapeError(err)
				assert.True(t, isPartial)
				if isPartial {
					assert.Equal(t, test.expectedErrCount, err.(scrapererror.PartialScrapeError).Failed)
				}

				return
			}

			pagingUsageMetric := metrics.At(0)
			assert.Equal(t, test.expectedUsedValue, pagingUsageMetric.IntSum().DataPoints().At(0).Value())
			assert.Equal(t, test.expectedFreeValue, pagingUsageMetric.IntSum().DataPoints().At(1).Value())
		})
	}
}
