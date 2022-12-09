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
	"testing"

	"github.com/stretchr/testify/mock"

	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memory"
)

// MockMemory can be used to set expectations within tests
type MockMemory interface {
	Stats(*memory.Stats)
	Total() (uint64, error)
	GC()
}

type mockMemory struct {
	mock.Mock
}

// MethodOptions allows to add expectations
// to the method being called.
type MethodOption func(*mock.Call)

// MockOption allows to set expectations for the
// configured MockMemory
type MockOption func(*mockMemory)

// WithMethodCalled is used to define how many times
// the configured method is expected to be called.
func WithMethodCalled(times int) MethodOption {
	return func(c *mock.Call) {
		if times < 1 {
			c.Maybe()
			times = 0
		}
		c.Times(times)
	}
}

// AsStatsFunc is a convenience function to convert the mocked
// memory as the memory.StatsFunc since casting can be problematic
func AsStatsFunc(mm MockMemory) memory.StatsFunc {
	return mm.Stats
}

// AsTotalFunc is a convenience function to convert the mocked
// memory as the memory.TotalFunc since casting can be problematic
func AsTotalFunc(mm MockMemory) memory.TotalFunc {
	return mm.Total
}

// AsGCFunc is a convenience function to convert the mocked
// memory as the memory.GCFunc since casting can be problematic
func AsGCFunc(mm MockMemory) memory.GCFunc {
	return mm.GC
}

func (mm *mockMemory) Stats(s *memory.Stats) {
	args := mm.Called()
	(*s) = *(args.Get(0).(*memory.Stats))

}

func (mm *mockMemory) Total() (uint64, error) {
	args := mm.Called()
	return args.Get(0).(uint64), args.Error(1)
}

func (mm *mockMemory) GC() {
	mm.Called()
}

// WithAssertMockedStats defines what `MockMemory.Stats()` will return.
func WithAssertMockedStats(updated *memory.Stats, opts ...MethodOption) MockOption {
	return func(mm *mockMemory) {
		call := mm.On("Stats").Return(updated)
		for _, opt := range opts {
			opt(call)
		}
	}
}

// WithAssertMockedTotal defines what `MockMemory.Total()` will return.
func WithAssertMockedTotal(size uint64, err error, opts ...MethodOption) MockOption {
	return func(mm *mockMemory) {
		call := mm.On("Total").Return(size, err)
		for _, opt := range opts {
			opt(call)
		}
	}
}

// WithAssertMockedGC defines what `MockMemory.GC()` will return.
func WithAssertMockedGC(opts ...MethodOption) MockOption {
	return func(mm *mockMemory) {
		call := mm.On("GC").Return()
		for _, opt := range opts {
			opt(call)
		}
	}
}

// NewMockMemory returns a MockMemory that can be configured to return
// values as defined by the options to be used within tests.
//
// An example of configuration is:
//
//	memorytest.NewMockMemory(
//		t,
//		memorytest.WithAssertMockedTotal(1200, nil, memorytest.WithMethodCalled(1)),
//		memorytest.WithAssertMockedGC(memorytest.WithMethodCalled(0)),
//		memorytest.WithAssertMockedStats(&memory.Stats{Alloc:40}, memorytest.WithMethodCalled(6)),
//	)
//
// The above configuration will return a `memorytest.MockMemory` that is expecting
// `MockMemory.Total` is to be called once with the values of `(1200, nil)`
// `MockMemory.Stats` is to be called six times with the value of `&memory.Stats{Alloc:40}`
// If the associated `*testing.T` does not meat the expectations set, the test will fail on cleanup.
func NewMockMemory(tb testing.TB, opts ...MockOption) MockMemory {
	tb.Helper()

	m := &mockMemory{}
	for _, opt := range opts {
		opt(m)
	}
	tb.Cleanup(func() {
		m.AssertExpectations(tb)
	})
	return m
}
