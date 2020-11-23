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

package scraperhelper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

func TestCombineScrapeErrors(t *testing.T) {
	testCases := []struct {
		errors                    []error
		expected                  string
		expectNil                 bool
		expectedPartialScrapeErr  bool
		expectedFailedScrapeCount int
	}{
		{
			errors:    []error{},
			expectNil: true,
		},
		{
			errors: []error{
				fmt.Errorf("foo"),
			},
			expected: "foo",
		},
		{
			errors: []error{
				fmt.Errorf("foo"),
				fmt.Errorf("bar"),
			},
			expected: "[foo; bar]",
		},
		{
			errors: []error{
				fmt.Errorf("foo"),
				fmt.Errorf("bar"),
				consumererror.NewPartialScrapeError(fmt.Errorf("partial"), 0)},
			expected:                  "[foo; bar; partial]",
			expectedPartialScrapeErr:  true,
			expectedFailedScrapeCount: 0,
		},
		{
			errors: []error{
				fmt.Errorf("foo"),
				fmt.Errorf("bar"),
				consumererror.NewPartialScrapeError(fmt.Errorf("partial 1"), 2),
				consumererror.NewPartialScrapeError(fmt.Errorf("partial 2"), 3)},
			expected:                  "[foo; bar; partial 1; partial 2]",
			expectedPartialScrapeErr:  true,
			expectedFailedScrapeCount: 5,
		},
	}

	for _, tc := range testCases {
		got := CombineScrapeErrors(tc.errors)

		if tc.expectNil {
			assert.NoError(t, got, tc.expected)
		} else {
			assert.EqualError(t, got, tc.expected)
		}

		partialErr, isPartial := got.(consumererror.PartialScrapeError)
		assert.Equal(t, tc.expectedPartialScrapeErr, isPartial)

		if tc.expectedPartialScrapeErr && isPartial {
			assert.Equal(t, tc.expectedFailedScrapeCount, partialErr.Failed)
		}
	}
}
