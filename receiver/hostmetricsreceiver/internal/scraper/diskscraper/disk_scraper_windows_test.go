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

package diskscraper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/windows/pdh"
)

func TestScrapeMetrics_Error(t *testing.T) {
	type testCase struct {
		name                                   string
		diskReadBytesPerSecCounterReturnValue  interface{}
		diskWriteBytesPerSecCounterReturnValue interface{}
		diskReadsPerSecCounterReturnValue      interface{}
		diskWritesPerSecCounterReturnValue     interface{}
		avgDiskSecsPerReadCounterReturnValue   interface{}
		avgDiskSecsPerWriteCounterReturnValue  interface{}
		diskQueueLengthCounterReturnValue      interface{}
		expectedErr                            string
	}

	testCases := []testCase{
		{
			name:                                  "readBytesPerSecCounterError",
			diskReadBytesPerSecCounterReturnValue: errors.New("err1"),
			expectedErr:                           "err1",
		},
		{
			name:                                   "writeBytesPerSecCounterError",
			diskWriteBytesPerSecCounterReturnValue: errors.New("err1"),
			expectedErr:                            "err1",
		},
		{
			name:                              "readsPerSecCounterError",
			diskReadsPerSecCounterReturnValue: errors.New("err1"),
			expectedErr:                       "err1",
		},
		{
			name:                               "writesPerSecCounterError",
			diskWritesPerSecCounterReturnValue: errors.New("err1"),
			expectedErr:                        "err1",
		},
		{
			name:                                 "avgSecsPerReadCounterError",
			avgDiskSecsPerReadCounterReturnValue: errors.New("err1"),
			expectedErr:                          "err1",
		},
		{
			name:                                  "avgSecsPerReadWriteError",
			avgDiskSecsPerWriteCounterReturnValue: errors.New("err1"),
			expectedErr:                           "err1",
		},
		{
			name:                              "avgDiskQueueLengthError",
			diskQueueLengthCounterReturnValue: errors.New("err1"),
			expectedErr:                       "err1",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newDiskScraper(context.Background(), &Config{})
			require.NoError(t, err, "Failed to create disk scraper: %v", err)

			err = scraper.Initialize(context.Background())
			require.NoError(t, err, "Failed to initialize disk scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			scraper.diskReadBytesPerSecCounter = pdh.NewMockPerfCounter(test.diskReadBytesPerSecCounterReturnValue)
			scraper.diskWriteBytesPerSecCounter = pdh.NewMockPerfCounter(test.diskWriteBytesPerSecCounterReturnValue)
			scraper.diskReadsPerSecCounter = pdh.NewMockPerfCounter(test.diskReadsPerSecCounterReturnValue)
			scraper.diskWritesPerSecCounter = pdh.NewMockPerfCounter(test.diskWritesPerSecCounterReturnValue)
			scraper.avgDiskSecsPerReadCounter = pdh.NewMockPerfCounter(test.avgDiskSecsPerReadCounterReturnValue)
			scraper.avgDiskSecsPerWriteCounter = pdh.NewMockPerfCounter(test.avgDiskSecsPerWriteCounterReturnValue)
			scraper.diskQueueLengthCounter = pdh.NewMockPerfCounter(test.diskQueueLengthCounterReturnValue)

			_, err = scraper.ScrapeMetrics(context.Background())
			assert.EqualError(t, err, test.expectedErr)
		})
	}
}
