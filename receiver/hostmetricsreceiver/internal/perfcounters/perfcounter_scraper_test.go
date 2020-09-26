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

package perfcounters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/processor/filterset"
)

func Test_PerfCounterScraper(t *testing.T) {
	type testCase struct {
		name string
		// NewPerfCounter
		objects       []string
		newErr        string
		expectIndices []string
		// Filter
		includeFS    filterset.FilterSet
		excludeFS    filterset.FilterSet
		includeTotal bool
		// GetObject
		getObject    string
		getObjectErr string
		// GetCounterValues
		getCounters              []string
		getCountersErr           string
		expectedInstanceNames    []string
		excludedInstanceNames    []string
		expectedMinimumInstances int
	}

	excludedCommonDrives := []string{"C:"}
	excludeCommonDriveFilterSet, err := filterset.CreateFilterSet(excludedCommonDrives, &filterset.Config{MatchType: filterset.Strict})
	require.NoError(t, err)

	testCases := []testCase{
		{
			name:                  "Standard",
			objects:               []string{"Memory"},
			expectIndices:         []string{"4"},
			getObject:             "Memory",
			getCounters:           []string{"Committed Bytes"},
			expectedInstanceNames: []string{""},
		},
		{
			name:                     "Multiple Objects & Values",
			objects:                  []string{"Memory", "LogicalDisk"},
			expectIndices:            []string{"4", "236"},
			getObject:                "LogicalDisk",
			getCounters:              []string{"Disk Reads/sec", "Disk Writes/sec"},
			expectedMinimumInstances: 1,
		},
		{
			name:                  "Filtered",
			objects:               []string{"LogicalDisk"},
			expectIndices:         []string{"236"},
			excludeFS:             excludeCommonDriveFilterSet,
			includeTotal:          true,
			getObject:             "LogicalDisk",
			getCounters:           []string{"Disk Reads/sec"},
			excludedInstanceNames: excludedCommonDrives,
		},
		{
			name:    "New Error",
			objects: []string{"Memory", "Invalid Object 1", "Invalid Object 2"},
			newErr:  `Failed to retrieve perf counter object "Invalid Object 1"`,
		},
		{
			name:          "Get Object Error",
			objects:       []string{"Memory"},
			expectIndices: []string{"4"},
			getObject:     "Invalid Object 1",
			getObjectErr:  `Unable to find object "Invalid Object 1"`,
		},
		{
			name:           "Get Values Error",
			objects:        []string{"Memory"},
			expectIndices:  []string{"4"},
			getObject:      "Memory",
			getCounters:    []string{"Committed Bytes", "Invalid Counter 1", "Invalid Counter 2"},
			getCountersErr: `Unable to find counters ["Invalid Counter 1" "Invalid Counter 2"] in object "Memory"`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			s := &PerfLibScraper{}
			err := s.Initialize(test.objects...)
			if test.newErr != "" {
				assert.EqualError(t, err, test.newErr)
				return
			}
			require.NoError(t, err, "Failed to create new perf counter scraper: %v", err)

			assert.ElementsMatch(t, test.expectIndices, strings.Split(s.objectIndices, " "))

			c, err := s.Scrape()
			require.NoError(t, err, "Failed to scrape data: %v", err)

			p, err := c.GetObject(test.getObject)
			if test.getObjectErr != "" {
				assert.EqualError(t, err, test.getObjectErr)
				return
			}
			require.NoError(t, err, "Failed to get object: %v", err)

			p.Filter(test.includeFS, test.excludeFS, test.includeTotal)

			counterValues, err := p.GetValues(test.getCounters...)
			if test.getCountersErr != "" {
				assert.EqualError(t, err, test.getCountersErr)
				return
			}
			require.NoError(t, err, "Failed to get counter: %v", err)

			assert.GreaterOrEqual(t, len(counterValues), test.expectedMinimumInstances)

			if len(test.expectedInstanceNames) > 0 {
				for _, expectedName := range test.expectedInstanceNames {
					var gotName bool
					for _, cv := range counterValues {
						if cv.InstanceName == expectedName {
							gotName = true
							break
						}
					}
					assert.Truef(t, gotName, "Expected Instance %q was not returned", expectedName)
				}
			}

			if len(test.excludedInstanceNames) > 0 {
				for _, excludedName := range test.excludedInstanceNames {
					for _, cv := range counterValues {
						if cv.InstanceName == excludedName {
							assert.Fail(t, "", "Excluded Instance %q was returned", excludedName)
							break
						}
					}
				}
			}

			var includesTotal bool
			for _, cv := range counterValues {
				if cv.InstanceName == "_Total" {
					includesTotal = true
					break
				}
			}
			assert.Equalf(t, test.includeTotal, includesTotal, "_Total was returned: %v (expected the opposite)", test.includeTotal, includesTotal)
		})
	}
}
