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

package strict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	validStrictFilters = []string{
		"exact_string_match",
		".*/suffix",
		"(a|b)",
	}
)

func TestNewStrictFilterSet(t *testing.T) {
	tests := []struct {
		name    string
		filters []string
		success bool
	}{
		{
			name:    "validFilters",
			filters: validStrictFilters,
			success: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs := NewFilterSet(test.filters)
			assert.Equal(t, test.success, fs != nil)
		})
	}
}

func TestStrictMatches(t *testing.T) {
	fs := NewFilterSet(validStrictFilters)
	assert.NotNil(t, fs)

	matches := []string{
		"exact_string_match",
		".*/suffix",
		"(a|b)",
	}

	for _, m := range matches {
		t.Run(m, func(t *testing.T) {
			assert.True(t, fs.Matches(m))
		})
	}

	mismatches := []string{
		"not_exact_string_match",
		"random",
		"test/match/suffix",
		"prefix/metric/one",
		"c",
	}

	for _, m := range mismatches {
		t.Run(m, func(t *testing.T) {
			assert.False(t, fs.Matches(m))
		})
	}
}
