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

package memorylimiterprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tilinna/clock"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memory"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memorytest"
)

func TestObserver(t *testing.T) {
	t.Parallel()

	type Event struct {
		step     string
		advance  time.Duration
		exceeded bool
	}

	for _, tc := range []struct {
		name       string
		memoryOpts []memorytest.MockOption
		conf       *Config

		events []Event
	}{
		{
			name: "simple system events under limits",
			memoryOpts: []memorytest.MockOption{
				memorytest.WithAssertMockedGC(
					memorytest.WithMethodCalled(0),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 10 * mibBytes},
					memorytest.WithMethodCalled(2),
				),
			},
			conf: &Config{
				MemoryLimitMiB:      80,
				MemorySpikeLimitMiB: 60,
			},
			events: []Event{
				{step: "started", advance: time.Second, exceeded: false},
				{step: "under limits", advance: time.Second, exceeded: false},
			},
		},
		{
			name: "breaches soft limit",
			memoryOpts: []memorytest.MockOption{
				memorytest.WithAssertMockedGC(
					memorytest.WithMethodCalled(1),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 30 * mibBytes},
					memorytest.WithMethodCalled(1),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 600 * mibBytes},
					memorytest.WithMethodCalled(2),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 30 * mibBytes},
					memorytest.WithMethodCalled(1),
				),
			},
			conf: &Config{
				MemoryLimitMiB:      800,
				MemorySpikeLimitMiB: 500,
			},
			events: []Event{
				{step: "started", advance: time.Second, exceeded: false},
				{step: "breaching soft", advance: time.Second, exceeded: true},
				{step: "forced GC, recovered", advance: 2 * minGCIntervalWhenSoftLimited, exceeded: false},
			},
		},
		{
			name: "breaches soft limit and recovers",
			memoryOpts: []memorytest.MockOption{
				memorytest.WithAssertMockedGC(
					memorytest.WithMethodCalled(0),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 300 * mibBytes},
					memorytest.WithMethodCalled(1),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 1600 * mibBytes},
					memorytest.WithMethodCalled(1),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 300 * mibBytes},
					memorytest.WithMethodCalled(1),
				),
			},
			conf: &Config{
				MemoryLimitMiB:      2000,
				MemorySpikeLimitMiB: 1500,
			},
			events: []Event{
				{step: "started", advance: time.Second, exceeded: false},
				{step: "breaching soft limit", advance: time.Second, exceeded: true},
				{step: "recovered naturall", advance: time.Second, exceeded: false},
			},
		},
		{
			name: "breaches hard limit and recovers",
			memoryOpts: []memorytest.MockOption{
				memorytest.WithAssertMockedGC(
					memorytest.WithMethodCalled(1),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 300 * mibBytes},
					memorytest.WithMethodCalled(1),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 6000 * mibBytes},
					memorytest.WithMethodCalled(2),
				),
				memorytest.WithAssertMockedStats(
					&memory.Stats{Alloc: 300 * mibBytes},
					memorytest.WithMethodCalled(1),
				),
			},
			conf: &Config{
				MemoryLimitMiB:      2000,
				MemorySpikeLimitMiB: 1500,
			},
			events: []Event{
				{step: "started", advance: time.Second, exceeded: false},
				{step: "breached hard limit", advance: time.Second, exceeded: true},
				{step: "recovered", advance: time.Second, exceeded: false},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, done := context.WithCancel(context.Background())
			defer done()

			clck := clock.NewMock(time.Unix(100, 0))

			ctx = clock.Context(ctx, clck)

			mem := memorytest.NewMockMemory(t, tc.memoryOpts...)

			limiter, err := memorytest.AsTotalFunc(mem).NewMemChecker(
				uint64(tc.conf.MemoryLimitMiB),
				uint64(tc.conf.MemorySpikeLimitMiB),
				uint64(tc.conf.MemoryLimitPercentage),
				uint64(tc.conf.MemorySpikePercentage),
			)
			require.NoError(t, err, "Must not error when loading mem checker")

			obs := newObserver(
				ctx,
				*limiter,
				zaptest.NewLogger(t),
				withMemoryStats(mem.Stats),
				withMemoryGC(mem.GC),
			)

			assert.NoError(t, obs.start(ctx, &host{ballastSize: 0}))

			for _, event := range tc.events {
				clck.Add(event.advance)
				obs.checkMemoryUsage()

				assert.Equal(t, event.exceeded, obs.memoryExceeded(), event.step)
			}
			assert.Zero(t, clck.Len(), "Must have closed all timers and tickers")
		})
	}
}
