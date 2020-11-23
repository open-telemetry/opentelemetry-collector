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
	"testing"

	"github.com/shirou/gopsutil/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

func TestScrape_Others(t *testing.T) {
	type testCase struct {
		name           string
		ioCountersFunc func(names ...string) (map[string]disk.IOCountersStat, error)
		expectedErr    string
	}

	testCases := []testCase{
		{
			name:           "Error",
			ioCountersFunc: func(names ...string) (map[string]disk.IOCountersStat, error) { return nil, errors.New("err1") },
			expectedErr:    "err1",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newDiskScraper(context.Background(), &Config{})
			require.NoError(t, err, "Failed to create disk scraper: %v", err)

			if test.ioCountersFunc != nil {
				scraper.ioCounters = test.ioCountersFunc
			}

			err = scraper.start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err, "Failed to initialize disk scraper: %v", err)

			_, err = scraper.scrape(context.Background())
			assert.EqualError(t, err, test.expectedErr)

			isPartial := consumererror.IsPartialScrapeError(err)
			assert.True(t, isPartial)
			if isPartial {
				assert.Equal(t, metricsLen, err.(consumererror.PartialScrapeError).Failed)
			}
		})
	}
}
