// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// mockScraper implements component.Component for testing.
type mockScraper struct {
	component.StartFunc
	component.ShutdownFunc
}

func newTestController(t *testing.T, cfg *ControllerConfig, scrapers []component.Component, tickerCh <-chan time.Time) *Controller[component.Component] {
	t.Helper()
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		scrapers,
		func(context.Context, *Controller[component.Component]) error { return nil },
		tickerCh,
	)
	require.NoError(t, err)
	return ctrl
}

func TestNewController(t *testing.T) {
	t.Parallel()

	scrapeFunc := func(context.Context, *Controller[component.Component]) error { return nil }

	for _, tc := range []struct {
		name     string
		cfg      *ControllerConfig
		scrapers []component.Component
		tickerCh <-chan time.Time
	}{
		{
			name: "default config",
			cfg: func() *ControllerConfig {
				cfg := NewDefaultControllerConfig()
				return &cfg
			}(),
		},
		{
			name: "custom collection interval and timeout",
			cfg: &ControllerConfig{
				CollectionInterval: 5 * time.Second,
				InitialDelay:       2 * time.Second,
				Timeout:            10 * time.Second,
			},
		},
		{
			name: "with ticker channel",
			cfg: func() *ControllerConfig {
				cfg := NewDefaultControllerConfig()
				return &cfg
			}(),
			tickerCh: make(<-chan time.Time),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			scrapers := tc.scrapers
			if scrapers == nil {
				scrapers = []component.Component{}
			}

			ctrl, err := NewController(
				tc.cfg,
				receivertest.NewNopSettings(receivertest.NopType),
				scrapers,
				scrapeFunc,
				tc.tickerCh,
			)

			require.NoError(t, err)
			require.NotNil(t, ctrl)

			assert.Equal(t, tc.cfg.CollectionInterval, ctrl.collectionInterval)
			assert.Equal(t, tc.cfg.InitialDelay, ctrl.initialDelay)
			assert.Equal(t, tc.cfg.Timeout, ctrl.Timeout)
			assert.Equal(t, scrapers, ctrl.Scrapers)
			assert.NotNil(t, ctrl.Obsrecv)
			assert.NotNil(t, ctrl.done)
		})
	}
}

func TestStartScrapersStarted(t *testing.T) {
	t.Parallel()

	var started int
	startFunc := component.StartFunc(func(context.Context, component.Host) error {
		started++
		return nil
	})
	scrapers := []component.Component{
		&mockScraper{StartFunc: startFunc},
		&mockScraper{StartFunc: startFunc},
	}

	tickerCh := make(chan time.Time)
	cfg := &ControllerConfig{CollectionInterval: time.Minute}
	ctrl := newTestController(t, cfg, scrapers, tickerCh)

	require.NoError(t, ctrl.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 2, started)
	require.NoError(t, ctrl.Shutdown(context.Background()))
}

func TestStartScraperError(t *testing.T) {
	t.Parallel()

	errScraper := errors.New("scraper start failed")
	scrapers := []component.Component{
		&mockScraper{StartFunc: component.StartFunc(func(context.Context, component.Host) error {
			return errScraper
		})},
	}

	tickerCh := make(chan time.Time)
	cfg := &ControllerConfig{CollectionInterval: time.Minute}
	ctrl := newTestController(t, cfg, scrapers, tickerCh)

	err := ctrl.Start(context.Background(), componenttest.NewNopHost())
	require.ErrorIs(t, err, errScraper)
}

func TestStartScrapingWithNilTickerCh(t *testing.T) {
	t.Parallel()

	var scrapeCount atomic.Int32
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		scrapeCount.Add(1)
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: 50 * time.Millisecond,
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil, // nil tickerCh — will create a real ticker internally.
	)
	require.NoError(t, err)

	require.NoError(t, ctrl.Start(context.Background(), componenttest.NewNopHost()))

	// Should get initial scrape + at least one ticker scrape.
	require.Eventually(t, func() bool {
		return scrapeCount.Load() >= 2
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, ctrl.Shutdown(context.Background()))
}

func TestShutdownScrapers(t *testing.T) {
	t.Parallel()

	var shutdownOrder []int
	scrapers := []component.Component{
		&mockScraper{ShutdownFunc: component.ShutdownFunc(func(context.Context) error {
			shutdownOrder = append(shutdownOrder, 1)
			return nil
		})},
		&mockScraper{ShutdownFunc: component.ShutdownFunc(func(context.Context) error {
			shutdownOrder = append(shutdownOrder, 2)
			return nil
		})},
	}

	tickerCh := make(chan time.Time)
	cfg := &ControllerConfig{CollectionInterval: time.Minute}
	ctrl := newTestController(t, cfg, scrapers, tickerCh)

	require.NoError(t, ctrl.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, ctrl.Shutdown(context.Background()))

	assert.Equal(t, []int{1, 2}, shutdownOrder)
}

func TestShutdownScraperErrors(t *testing.T) {
	t.Parallel()

	errShutdown1 := errors.New("shutdown error 1")
	errShutdown2 := errors.New("shutdown error 2")
	scrapers := []component.Component{
		&mockScraper{ShutdownFunc: component.ShutdownFunc(func(context.Context) error {
			return errShutdown1
		})},
		&mockScraper{ShutdownFunc: component.ShutdownFunc(func(context.Context) error {
			return errShutdown2
		})},
	}

	tickerCh := make(chan time.Time)
	cfg := &ControllerConfig{CollectionInterval: time.Minute}
	ctrl := newTestController(t, cfg, scrapers, tickerCh)

	require.NoError(t, ctrl.Start(context.Background(), componenttest.NewNopHost()))
	err := ctrl.Shutdown(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, errShutdown1)
	require.ErrorIs(t, err, errShutdown2)
}

func TestStartScraping(t *testing.T) {
	t.Parallel()

	tickerCh := make(chan time.Time)
	var scrapeCount atomic.Int32
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		scrapeCount.Add(1)
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: time.Minute,
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		tickerCh,
	)
	require.NoError(t, err)

	require.NoError(t, ctrl.Start(context.Background(), componenttest.NewNopHost()))

	// startScraping calls scrapeFunc immediately on start.
	require.Eventually(t, func() bool {
		return scrapeCount.Load() >= 1
	}, time.Second, 10*time.Millisecond)

	// Simulate a tick — should trigger another scrape.
	tickerCh <- time.Now()
	require.Eventually(t, func() bool {
		return scrapeCount.Load() >= 2
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, ctrl.Shutdown(context.Background()))
}

func TestStartScrapingWithInitialDelay(t *testing.T) {
	t.Parallel()

	tickerCh := make(chan time.Time)
	var scrapeCount atomic.Int32
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		scrapeCount.Add(1)
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: time.Minute,
		InitialDelay:       50 * time.Millisecond,
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		tickerCh,
	)
	require.NoError(t, err)

	require.NoError(t, ctrl.Start(context.Background(), componenttest.NewNopHost()))

	// The initial scrape should happen after the initial delay.
	require.Eventually(t, func() bool {
		return scrapeCount.Load() >= 1
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, ctrl.Shutdown(context.Background()))
}

func TestStartScrapingShutdownDuringInitialDelay(t *testing.T) {
	t.Parallel()

	var scraped atomic.Bool
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		scraped.Store(true)
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: time.Minute,
		InitialDelay:       time.Hour, // Very long delay — we'll shut down before it expires.
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil,
	)
	require.NoError(t, err)

	require.NoError(t, ctrl.Start(context.Background(), componenttest.NewNopHost()))
	// Shutdown immediately, which should cancel the initial delay wait.
	require.NoError(t, ctrl.Shutdown(context.Background()))

	assert.False(t, scraped.Load(), "scrapeFunc should not have been called")
}

func TestGetSettings(t *testing.T) {
	t.Parallel()

	sType := component.MustNewType("test_scraper")
	rSet := receivertest.NewNopSettings(receivertest.NopType)

	sSet := GetSettings(sType, rSet)

	assert.Equal(t, component.NewID(sType), sSet.ID)
	assert.Equal(t, rSet.BuildInfo, sSet.BuildInfo)
}

func TestScrapeFuncAppliesTimeout(t *testing.T) {
	t.Parallel()

	timeout := 5 * time.Second
	var deadline time.Time
	var hasDeadline bool
	scrapeFunc := func(ctx context.Context, _ *Controller[component.Component]) error {
		deadline, hasDeadline = ctx.Deadline()
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: time.Minute,
		Timeout:            timeout,
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil,
	)
	require.NoError(t, err)

	require.NoError(t, ctrl.scrapeFunc(context.Background(), ctrl))
	assert.True(t, hasDeadline)
	assert.WithinDuration(t, time.Now().Add(timeout), deadline, time.Second)
}

func TestScrapeFuncNoTimeout(t *testing.T) {
	t.Parallel()

	var hasDeadline bool
	scrapeFunc := func(ctx context.Context, _ *Controller[component.Component]) error {
		_, hasDeadline = ctx.Deadline()
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: time.Minute,
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil,
	)
	require.NoError(t, err)

	require.NoError(t, ctrl.scrapeFunc(context.Background(), ctrl))
	assert.False(t, hasDeadline)
}

func TestScrapeFuncPropagatesParentCancellation(t *testing.T) {
	t.Parallel()

	var gotErr error
	scrapeFunc := func(ctx context.Context, _ *Controller[component.Component]) error {
		gotErr = ctx.Err()
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: time.Minute,
		Timeout:            time.Hour,
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil,
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, ctrl.scrapeFunc(ctx, ctrl))
	assert.ErrorIs(t, gotErr, context.Canceled)
}

func TestScrapeFuncReturnsError(t *testing.T) {
	t.Parallel()

	scrapeErr := errors.New("scrape failed")
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		return scrapeErr
	}

	cfg := &ControllerConfig{
		CollectionInterval: time.Minute,
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil,
	)
	require.NoError(t, err)

	assert.ErrorIs(t, ctrl.scrapeFunc(context.Background(), ctrl), scrapeErr)
}
