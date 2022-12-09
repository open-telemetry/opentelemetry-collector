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

package memorytest

import (
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memory"
)

func TestNewMockMemory(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		opts       []MockOption
		invocation func(tb testing.TB, mm MockMemory)
	}{
		{
			name: "no method calls",
			opts: []MockOption{
				WithAssertMockedStats(nil, WithMethodCalled(0)),
				WithAssertMockedTotal(0, nil, WithMethodCalled(0)),
			},
			invocation: func(tb testing.TB, mm MockMemory) {},
		},
		{
			name: "expects total to be called",
			opts: []MockOption{
				WithAssertMockedTotal(100, nil, WithMethodCalled(1)),
			},
			invocation: func(tb testing.TB, mm MockMemory) {
				v, err := AsTotalFunc(mm).Load()
				assert.EqualValues(t, 100, v, "Must match the expected value")
				assert.NoError(t, err, "Must not error")
			},
		},
		{
			name: "expects total func to fail",
			opts: []MockOption{
				WithAssertMockedTotal(0, errors.New("boom"), WithMethodCalled(1)),
			},
			invocation: func(tb testing.TB, mm MockMemory) {
				v, err := AsTotalFunc(mm).Load()
				assert.Zero(t, v, "Must return 0")
				assert.Error(t, err, "Must return an error")
			},
		},
		{
			name: "expects stats to be called",
			opts: []MockOption{
				WithAssertMockedStats(&runtime.MemStats{Alloc: 1000}, WithMethodCalled(1)),
			},
			invocation: func(tb testing.TB, mm MockMemory) {
				s, mismatched := AsStatsFunc(mm).Load(0)
				assert.EqualValues(t, &memory.Stats{Alloc: 1000}, s, "Must match values")
				assert.False(t, mismatched, "Must not be mismatched")
			},
		},
		{
			name: "expects GC to be called",
			opts: []MockOption{
				WithAssertMockedGC(WithMethodCalled(1)),
			},
			invocation: func(tb testing.TB, mm MockMemory) {
				assert.NotPanics(tb, func() {
					AsGCFunc(mm).Do()
				}, "Must not panic during invocation")
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mm := NewMockMemory(t, tc.opts...)
			tc.invocation(t, mm)
		})
	}
}
