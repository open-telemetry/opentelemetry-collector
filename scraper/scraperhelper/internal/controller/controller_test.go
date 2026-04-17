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
		func(*Controller[component.Component]) {},
		tickerCh,
	)
	require.NoError(t, err)
	return ctrl
}

func TestNewController(t *testing.T) {
	t.Parallel()

	scrapeFunc := func(*Controller[component.Component]) {}

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
	scrapeFunc := func(*Controller[component.Component]) {
		scrapeCount.Add(1)
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
	scrapeFunc := func(*Controller[component.Component]) {
		scrapeCount.Add(1)
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
	scrapeFunc := func(*Controller[component.Component]) {
		scrapeCount.Add(1)
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
	scrapeFunc := func(*Controller[component.Component]) {
		scraped.Store(true)
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

func TestWithScrapeContext(t *testing.T) {
	t.Parallel()

	t.Run("zero timeout returns context without deadline", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := WithScrapeContext(0)
		defer cancel()
		_, hasDeadline := ctx.Deadline()
		assert.False(t, hasDeadline)
	})

	t.Run("positive timeout returns context with deadline", func(t *testing.T) {
		t.Parallel()
		timeout := 5 * time.Second
		ctx, cancel := WithScrapeContext(timeout)
		defer cancel()
		deadline, hasDeadline := ctx.Deadline()
		assert.True(t, hasDeadline)
		// The deadline should be approximately now + timeout.
		assert.WithinDuration(t, time.Now().Add(timeout), deadline, time.Second)
	})
}
