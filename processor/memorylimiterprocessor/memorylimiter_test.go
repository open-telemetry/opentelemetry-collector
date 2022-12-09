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

package memorylimiterprocessor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tilinna/clock"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memory"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/memorytest"
)

type host struct {
	ballastSize uint64
	component.Host
}

func (h *host) GetExtensions() map[component.ID]component.Component {
	ret := make(map[component.ID]component.Component)
	ret[component.NewID("ballast")] = &ballastExtension{ballastSize: h.ballastSize}
	return ret
}

type ballastExtension struct {
	ballastSize uint64
	component.StartFunc
	component.ShutdownFunc
}

func (be *ballastExtension) GetBallastSize() uint64 {
	return be.ballastSize
}

func newObsReport(tb testing.TB) *obsreport.Processor {
	tb.Helper()

	set := obsreport.ProcessorSettings{
		ProcessorID:             component.NewID(typeStr),
		ProcessorCreateSettings: componenttest.NewNopProcessorCreateSettings(),
	}
	set.ProcessorCreateSettings.MetricsLevel = configtelemetry.LevelNone

	proc, err := obsreport.NewProcessor(set)
	require.NoError(tb, err)

	return proc
}

func TestNewMemoryLimiter(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		conf *Config
		opts []func(*memorySettings)

		err error
	}{
		{
			name: "Valid Memory Limiter",
			conf: &Config{
				CheckInterval:       time.Minute,
				MemoryLimitMiB:      1000,
				MemorySpikeLimitMiB: 800,
			},
			err: nil,
		},
		{
			name: "Total memory failed",
			conf: &Config{
				CheckInterval:         time.Minute,
				MemoryLimitPercentage: 90,
			},
			err: errForcedDrop,
			opts: []func(*memorySettings){
				func(ms *memorySettings) {
					ms.total = func() (uint64, error) {
						return 0, errForcedDrop
					}
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lim, err := newMemoryLimiter(
				context.Background(),
				componenttest.NewNopProcessorCreateSettings(),
				tc.conf,
				tc.opts...,
			)

			assert.ErrorIs(t, err, tc.err, "Must match expected error")
			if tc.err != nil {
				assert.Nil(t, lim, "Must be nil if error returned")
				return
			}
			assert.NotNil(t, lim, "Must have a valid limiter if no error is returned")
		})
	}
}

func TestTelemetryProcessing(t *testing.T) {
	type consumeEvent struct {
		name string
		err  error
	}

	for _, signal := range []struct {
		telemetry string
		consume   func(lim *memoryLimiter) error
	}{
		{
			telemetry: "logs",
			consume: func(lim *memoryLimiter) error {
				_, err := lim.processLogs(context.Background(), plog.NewLogs())
				return err
			},
		},
		{
			telemetry: "traces",
			consume: func(lim *memoryLimiter) error {
				_, err := lim.processTraces(context.Background(), ptrace.NewTraces())
				return err
			},
		},
		{
			telemetry: "metrics",
			consume: func(lim *memoryLimiter) error {
				_, err := lim.processMetrics(context.Background(), pmetric.NewMetrics())
				return err
			},
		},
	} {
		signal := signal
		for _, tc := range []struct {
			memOpts []memorytest.MockOption
			events  []consumeEvent
		}{
			{
				memOpts: []memorytest.MockOption{
					memorytest.WithAssertMockedGC(memorytest.WithMethodCalled(2)),
					memorytest.WithAssertMockedStats(
						&memory.Stats{Alloc: 10 * mibBytes},
						memorytest.WithMethodCalled(1),
					),
					memorytest.WithAssertMockedStats(
						&memory.Stats{Alloc: 60 * mibBytes},
						memorytest.WithMethodCalled(2),
					),
					memorytest.WithAssertMockedStats(
						&memory.Stats{Alloc: 10 * mibBytes},
						memorytest.WithMethodCalled(2),
					),
					memorytest.WithAssertMockedStats(
						&memory.Stats{Alloc: 92 * mibBytes},
						memorytest.WithMethodCalled(1),
					),
					memorytest.WithAssertMockedStats(
						&memory.Stats{Alloc: 10 * mibBytes},
						memorytest.WithMethodCalled(1),
					),
				},
				events: []consumeEvent{
					{name: "started with gc set to now", err: nil},
					{name: "breached soft limit, withing gc wait time, force dropping to recover", err: errForcedDrop},
					{name: "breached soft limit, exceeding gc wait time, force dropping to recover", err: nil},
					{name: "recovered", err: nil},
					{name: "breached hard limit, recovers after gc", err: nil},
				},
			},
		} {
			tc := tc
			t.Run(signal.telemetry, func(t *testing.T) {
				mem := memorytest.NewMockMemory(t, tc.memOpts...)
				ctx, done := context.WithCancel(context.Background())
				t.Cleanup(done)
				clk := clock.NewMock(time.Unix(100, 0))

				ctx = clock.Context(ctx, clk)

				lim, err := newMemoryLimiter(
					ctx,
					component.ProcessorCreateSettings{
						ID: component.NewID(typeStr),
						TelemetrySettings: component.TelemetrySettings{
							Logger: zaptest.NewLogger(t),
						},
						BuildInfo: component.NewDefaultBuildInfo(),
					},
					&Config{
						MemoryLimitMiB:      100,
						MemorySpikeLimitMiB: 60,
						CheckInterval:       5 * time.Second,
					},
					func(ms *memorySettings) {
						ms.total = memorytest.AsTotalFunc(mem)
						ms.usage = memorytest.AsStatsFunc(mem)
						ms.gc = memorytest.AsGCFunc(mem)
					},
				)
				require.NoError(t, err, "Must not error creating limiter")

				assert.NoError(t, lim.start(ctx, componenttest.NewNopHost()), "Must not error starting component")
				for _, event := range tc.events {
					clk.AddNext()
					// Even though virtual time is progressing, it still requires actual time for the background
					// process to be scheduled and run again.
					assert.Eventually(
						t,
						func() bool {
							return errors.Is(signal.consume(lim), event.err)
						},
						300*time.Millisecond, // If the test flakes, extends this time
						100*time.Millisecond,
						event.name,
					)
				}
				assert.NoError(t, lim.shutdown(ctx), "Must not error shutting down component")
				assert.NoError(t, lim.shutdown(ctx), "Must match the expected error")
				assert.Zero(t, clk.Len(), "Must have shutdown timer correctly after shutdown")
			})
		}

	}

}

func TestBallastSize(t *testing.T) {
	t.Parallel()

	cfg := createDefaultConfig().(*Config)
	cfg.CheckInterval = 10 * time.Second
	cfg.MemoryLimitMiB = 1024
	got, err := newMemoryLimiter(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg)
	require.NoError(t, err)
	require.NoError(t, got.start(context.Background(), &host{ballastSize: 113}))
	assert.EqualValues(t, 113, got.memObserver.ballastSize)
	require.NoError(t, got.shutdown(context.Background()))
}
