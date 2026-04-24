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
	"go.opentelemetry.io/collector/extension/xextension/extensionscrapercontroller"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// mockScraper implements component.Component for testing.
type mockScraper struct {
	component.StartFunc
	component.ShutdownFunc
}

// mockHost implements component.Host for testing.
type mockHost struct {
	ext map[component.ID]component.Component
}

func (h *mockHost) GetExtensions() map[component.ID]component.Component {
	return h.ext
}

// mockControllerExtension implements extensionscrapercontroller.ControllerExtension.
type mockControllerExtension struct {
	component.StartFunc
	component.ShutdownFunc
	scrapeFunc    func(context.Context) error
	deregistered  bool
	registerErr   error
	deregisterErr error
}

func (m *mockControllerExtension) RegisterScraper(_ context.Context, scrapeFunc func(context.Context) error) (extensionscrapercontroller.RegistrationHandle, error) {
	if m.registerErr != nil {
		return nil, m.registerErr
	}
	m.scrapeFunc = scrapeFunc
	return extensionscrapercontroller.DeregisterFunc(func(context.Context) error {
		m.deregistered = true
		return m.deregisterErr
	}), nil
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
			name: "with controllers",
			cfg: &ControllerConfig{
				CollectionInterval: time.Minute,
				Controllers:        []component.ID{component.MustNewID("myext")},
			},
		},
		{
			name: "zero collection interval with controllers",
			cfg: &ControllerConfig{
				Controllers: []component.ID{component.MustNewID("myext")},
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

func TestStartExtensionNotFound(t *testing.T) {
	t.Parallel()

	cfg := &ControllerConfig{
		Controllers: []component.ID{component.MustNewID("missing")},
	}
	ctrl := newTestController(t, cfg, nil, nil)

	err := ctrl.Start(context.Background(), componenttest.NewNopHost())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `extension "missing" not found`)
}

func TestStartExtensionNotControllerExtension(t *testing.T) {
	t.Parallel()

	extID := component.MustNewID("notcontroller")
	cfg := &ControllerConfig{
		Controllers: []component.ID{extID},
	}
	ctrl := newTestController(t, cfg, nil, nil)

	// Provide an extension that does not implement ControllerExtension.
	host := &mockHost{ext: map[component.ID]component.Component{
		extID: &mockScraper{},
	}}

	err := ctrl.Start(context.Background(), host)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a scraper controller extension")
}

func TestStartExtensionRegisterError(t *testing.T) {
	t.Parallel()

	extID := component.MustNewID("myext")
	errRegister := errors.New("register failed")
	mockExt := &mockControllerExtension{registerErr: errRegister}

	cfg := &ControllerConfig{
		Controllers: []component.ID{extID},
	}
	ctrl := newTestController(t, cfg, nil, nil)

	host := &mockHost{ext: map[component.ID]component.Component{extID: mockExt}}
	err := ctrl.Start(context.Background(), host)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register scraper")
	assert.ErrorIs(t, err, errRegister)
}

func TestStartPartialFailureCleansUp(t *testing.T) {
	t.Parallel()

	// First extension registers successfully, second fails. Start must
	// deregister the first registration before returning.
	extID1 := component.MustNewID("ext1")
	extID2 := component.MustNewID("ext2")
	okExt := &mockControllerExtension{}
	errRegister := errors.New("register failed")
	failExt := &mockControllerExtension{registerErr: errRegister}

	// Scraper that records whether Shutdown was called.
	var scraperShutdown atomic.Bool
	scrp := &mockScraper{
		ShutdownFunc: func(context.Context) error {
			scraperShutdown.Store(true)
			return nil
		},
	}

	cfg := &ControllerConfig{
		Controllers: []component.ID{extID1, extID2},
	}
	ctrl := newTestController(t, cfg, []component.Component{scrp}, nil)

	host := &mockHost{ext: map[component.ID]component.Component{
		extID1: okExt,
		extID2: failExt,
	}}
	err := ctrl.Start(context.Background(), host)
	require.Error(t, err)
	require.ErrorIs(t, err, errRegister)

	assert.True(t, okExt.deregistered, "first extension should have been deregistered")
	assert.True(t, scraperShutdown.Load(), "already-started scraper should have been shut down")
	assert.Empty(t, ctrl.handles, "handles slice should be cleared after partial-start cleanup")
}

func TestShutdownWaitsForInFlightExtensionScrape(t *testing.T) {
	t.Parallel()

	// Verify Shutdown does not call scraper.Shutdown until an in-flight
	// extension-triggered scrape has returned, even if the extension's
	// Deregister returns immediately.
	extID := component.MustNewID("myext")
	mockExt := &mockControllerExtension{}

	scrapeStarted := make(chan struct{})
	releaseScrape := make(chan struct{})
	scrapeFinished := make(chan struct{})
	var shutdownBeforeScrapeDone atomic.Bool

	scrp := &mockScraper{
		ShutdownFunc: func(context.Context) error {
			select {
			case <-scrapeFinished:
			default:
				shutdownBeforeScrapeDone.Store(true)
			}
			return nil
		},
	}

	scrapeFn := func(context.Context, *Controller[component.Component]) error {
		close(scrapeStarted)
		<-releaseScrape
		close(scrapeFinished)
		return nil
	}

	cfg := &ControllerConfig{
		Controllers: []component.ID{extID},
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{scrp},
		scrapeFn,
		nil,
	)
	require.NoError(t, err)

	host := &mockHost{ext: map[component.ID]component.Component{extID: mockExt}}
	require.NoError(t, ctrl.Start(context.Background(), host))

	// Trigger a scrape via the extension in a goroutine so we can control timing.
	scrapeDone := make(chan error, 1)
	go func() { scrapeDone <- mockExt.scrapeFunc(context.Background()) }()
	<-scrapeStarted

	// Shutdown in a goroutine: it must block on the in-flight scrape.
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- ctrl.Shutdown(context.Background()) }()

	// Give Shutdown a moment to run Deregister and reach inFlight.Wait.
	select {
	case <-shutdownDone:
		t.Fatal("Shutdown returned before in-flight scrape completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseScrape)
	require.NoError(t, <-scrapeDone)
	require.NoError(t, <-shutdownDone)
	assert.False(t, shutdownBeforeScrapeDone.Load(),
		"scraper.Shutdown must not be called while extension-triggered scrape is in flight")
}

func TestStartExtensionRegistersAndDeregisters(t *testing.T) {
	t.Parallel()

	extID := component.MustNewID("myext")
	mockExt := &mockControllerExtension{}

	cfg := &ControllerConfig{
		Controllers: []component.ID{extID},
	}
	ctrl := newTestController(t, cfg, nil, nil)

	host := &mockHost{ext: map[component.ID]component.Component{extID: mockExt}}
	require.NoError(t, ctrl.Start(context.Background(), host))
	require.NotNil(t, mockExt.scrapeFunc)
	assert.False(t, mockExt.deregistered)

	require.NoError(t, ctrl.Shutdown(context.Background()))
	assert.True(t, mockExt.deregistered)
}

func TestStartExtensionCallbackInvokesScrapeFunc(t *testing.T) {
	t.Parallel()

	extID := component.MustNewID("myext")
	mockExt := &mockControllerExtension{}

	var scraped atomic.Bool
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		scraped.Store(true)
		return nil
	}

	cfg := &ControllerConfig{
		Controllers: []component.ID{extID},
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil,
	)
	require.NoError(t, err)

	host := &mockHost{ext: map[component.ID]component.Component{extID: mockExt}}
	require.NoError(t, ctrl.Start(context.Background(), host))

	// Invoke the callback registered with the extension.
	require.NotNil(t, mockExt.scrapeFunc)
	require.NoError(t, mockExt.scrapeFunc(context.Background()))
	assert.True(t, scraped.Load())

	require.NoError(t, ctrl.Shutdown(context.Background()))
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

func TestStartScrapingNoCollectionInterval(t *testing.T) {
	t.Parallel()

	var scraped atomic.Bool
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		scraped.Store(true)
		return nil
	}

	cfg := &ControllerConfig{
		CollectionInterval: 0,
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

	// With zero CollectionInterval, startScraping should not be called.
	time.Sleep(50 * time.Millisecond)
	assert.False(t, scraped.Load())

	require.NoError(t, ctrl.Shutdown(context.Background()))
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

func TestShutdownDeregisterError(t *testing.T) {
	t.Parallel()

	extID := component.MustNewID("myext")
	errDeregister := errors.New("deregister failed")
	errShutdown := errors.New("scraper shutdown failed")

	mockExt := &mockControllerExtension{deregisterErr: errDeregister}
	scrapers := []component.Component{
		&mockScraper{ShutdownFunc: component.ShutdownFunc(func(context.Context) error {
			return errShutdown
		})},
	}

	cfg := &ControllerConfig{
		Controllers: []component.ID{extID},
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		scrapers,
		func(context.Context, *Controller[component.Component]) error { return nil },
		nil,
	)
	require.NoError(t, err)

	host := &mockHost{ext: map[component.ID]component.Component{extID: mockExt}}
	require.NoError(t, ctrl.Start(context.Background(), host))

	err = ctrl.Shutdown(context.Background())
	require.Error(t, err)
	// Both the deregister error and the scraper shutdown error should be reported.
	require.ErrorIs(t, err, errDeregister)
	require.ErrorIs(t, err, errShutdown)
	assert.True(t, mockExt.deregistered)
}

func TestStartMultipleExtensions(t *testing.T) {
	t.Parallel()

	ext1ID := component.MustNewID("ext1")
	ext2ID := component.MustNewID("ext2")
	mockExt1 := &mockControllerExtension{}
	mockExt2 := &mockControllerExtension{}

	cfg := &ControllerConfig{
		Controllers: []component.ID{ext1ID, ext2ID},
	}
	ctrl := newTestController(t, cfg, nil, nil)

	host := &mockHost{ext: map[component.ID]component.Component{
		ext1ID: mockExt1,
		ext2ID: mockExt2,
	}}
	require.NoError(t, ctrl.Start(context.Background(), host))
	require.NotNil(t, mockExt1.scrapeFunc)
	require.NotNil(t, mockExt2.scrapeFunc)

	require.NoError(t, ctrl.Shutdown(context.Background()))
	assert.True(t, mockExt1.deregistered)
	assert.True(t, mockExt2.deregistered)
}

func TestExtensionScrapeUsesPassedContext(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}

	extID := component.MustNewID("myext")
	mockExt := &mockControllerExtension{}

	var capturedCtxValue any
	var capturedCtxErr error
	scrapeFunc := func(ctx context.Context, _ *Controller[component.Component]) error {
		capturedCtxValue = ctx.Value(ctxKey{})
		capturedCtxErr = ctx.Err()
		return nil
	}

	cfg := &ControllerConfig{
		Controllers: []component.ID{extID},
	}
	ctrl, err := NewController(
		cfg,
		receivertest.NewNopSettings(receivertest.NopType),
		[]component.Component{},
		scrapeFunc,
		nil,
	)
	require.NoError(t, err)

	startCtx, cancelStart := context.WithCancel(context.Background())
	host := &mockHost{ext: map[component.ID]component.Component{extID: mockExt}}
	require.NoError(t, ctrl.Start(startCtx, host))
	cancelStart() // should not impact the scrape context

	// Extension later fires a scrape with its own live context carrying a value.
	extCtx := context.WithValue(context.Background(), ctxKey{}, "from-extension")
	require.NotNil(t, mockExt.scrapeFunc)
	require.NoError(t, mockExt.scrapeFunc(extCtx))

	assert.NoError(t, capturedCtxErr, "scrape context must not be canceled")
	assert.Equal(t, "from-extension", capturedCtxValue,
		"scrape function must receive the context the extension passed, not the Start context")

	require.NoError(t, ctrl.Shutdown(context.Background()))
}

func TestStartScrapingContinuesAfterError(t *testing.T) {
	t.Parallel()

	tickerCh := make(chan time.Time)
	var scrapeCount atomic.Int32
	scrapeFunc := func(context.Context, *Controller[component.Component]) error {
		scrapeCount.Add(1)
		return errors.New("scrape failed")
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

	// Initial scrape (which errors) should not prevent the ticker from firing
	// additional scrapes.
	require.Eventually(t, func() bool {
		return scrapeCount.Load() >= 1
	}, time.Second, 10*time.Millisecond)

	tickerCh <- time.Now()
	require.Eventually(t, func() bool {
		return scrapeCount.Load() >= 2
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, ctrl.Shutdown(context.Background()))
}
