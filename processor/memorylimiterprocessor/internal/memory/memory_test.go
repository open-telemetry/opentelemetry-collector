// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadStats(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		fn   StatsFunc

		ballastSize uint64
		expect      *Stats
		mismatched  bool
	}{
		{
			name:        "testing default function",
			ballastSize: 100,
		},
		{
			name: "testing expected stats",
			fn: func(s *Stats) {
				s.Alloc = 1000
			},
			ballastSize: 300,
			expect: &Stats{
				Alloc: 700,
			},
			mismatched: false,
		},
		{
			name: "mismatched ballast size",
			fn: func(s *Stats) {
				s.Alloc = 1000
			},
			ballastSize: 2000,
			expect: &Stats{
				Alloc: 1000,
			},
			mismatched: true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, mismatched := tc.fn.Load(tc.ballastSize)
			if tc.fn == nil {
				assert.NotNil(t, s, "Must be able to load stats from runtime")
				return
			}
			assert.EqualValues(t, tc.expect, s, "Must match the expected stat value")
			assert.Equal(t, tc.mismatched, mismatched, "Must match the expected mismatch value")
		})
	}
}
