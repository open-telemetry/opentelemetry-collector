// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package controller provides functionality used in scraperhelper and xscraperhelper.

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/extensionscrapercontroller"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/scraper"
)

type Controller[T component.Component] struct {
	collectionInterval time.Duration
	initialDelay       time.Duration
	Timeout            time.Duration

	Scrapers   []T
	scrapeFunc func(context.Context, *Controller[T]) error
	tickerCh   <-chan time.Time

	done chan struct{}
	wg   sync.WaitGroup

	controllers []component.ID
	deregFuncs  []extensionscrapercontroller.DeregisterFunc

	Obsrecv *receiverhelper.ObsReport
}

func NewController[T component.Component](
	cfg *ControllerConfig,
	rSet receiver.Settings,
	scrapers []T,
	scrapeFunc func(context.Context, *Controller[T]) error,
	tickerCh <-chan time.Time,
) (*Controller[T], error) {
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             rSet.ID,
		Transport:              "",
		ReceiverCreateSettings: rSet,
	})
	if err != nil {
		return nil, err
	}

	cs := &Controller[T]{
		collectionInterval: cfg.CollectionInterval,
		initialDelay:       cfg.InitialDelay,
		Timeout:            cfg.Timeout,
		Scrapers:           scrapers,
		scrapeFunc:         scrapeFunc,
		done:               make(chan struct{}),
		tickerCh:           tickerCh,
		controllers:        cfg.Controllers,
		Obsrecv:            obsrecv,
	}

	if cfg.Timeout > 0 {
		cs.scrapeFunc = func(ctx context.Context, c *Controller[T]) error {
			ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
			defer cancel()
			return scrapeFunc(ctx, c)
		}
	}

	return cs, nil
}

// Start the receiver, invoked during service start.
func (sc *Controller[T]) Start(ctx context.Context, host component.Host) (err error) {
	var startedScrapers int
	success := false
	defer func() {
		if success {
			return
		}
		err = multierr.Append(err, sc.teardown(ctx, sc.Scrapers[:startedScrapers]))
	}()

	for _, scrp := range sc.Scrapers {
		if err := scrp.Start(ctx, host); err != nil {
			return err
		}
		startedScrapers++
	}

	exts := host.GetExtensions()
	for _, controllerID := range sc.controllers {
		ext, found := exts[controllerID]
		if !found {
			return fmt.Errorf("extension %q not found", controllerID)
		}
		ce, ok := ext.(extensionscrapercontroller.ControllerExtension)
		if !ok {
			return fmt.Errorf("extension %q is not a scraper controller extension", controllerID)
		}
		deregFn, err := ce.RegisterScraper(ctx, extensionscrapercontroller.ScrapeFunc(func(callCtx context.Context) error {
			return sc.scrapeFunc(callCtx, sc)
		}))
		if err != nil {
			return fmt.Errorf("failed to register scraper with extension %q: %w", controllerID, err)
		}
		sc.deregFuncs = append(sc.deregFuncs, deregFn)
	}

	if sc.collectionInterval > 0 {
		sc.startScraping(ctx)
	}
	success = true
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (sc *Controller[T]) Shutdown(ctx context.Context) error {
	// Signal the ticker goroutine to stop and wait for it to exit.
	close(sc.done)
	sc.wg.Wait()
	return sc.teardown(ctx, sc.Scrapers)
}

// teardown invokes all deregister functions concurrently, and then shuts
// down all given scrapers.
func (sc *Controller[T]) teardown(ctx context.Context, scrapers []T) error {
	deregFuncs := sc.deregFuncs
	sc.deregFuncs = nil

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs error
	)
	record := func(e error) {
		mu.Lock()
		errs = multierr.Append(errs, e)
		mu.Unlock()
	}
	for _, deregFn := range deregFuncs {
		if deregFn == nil {
			// Defensive: prevent nil function calls in case a
			// controller extension returns a nil DeregisterFunc.
			continue
		}
		wg.Go(func() {
			if err := deregFn(ctx); err != nil {
				record(err)
			}
		})
	}
	wg.Wait()
	for _, scrp := range scrapers {
		wg.Go(func() {
			if err := scrp.Shutdown(ctx); err != nil {
				record(err)
			}
		})
	}
	wg.Wait()
	return errs
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (sc *Controller[T]) startScraping(ctx context.Context) {
	ctx = context.WithoutCancel(ctx)
	sc.wg.Go(func() {
		if sc.initialDelay > 0 {
			select {
			case <-time.After(sc.initialDelay):
			case <-sc.done:
				return
			}
		}

		if sc.tickerCh == nil {
			ticker := time.NewTicker(sc.collectionInterval)
			defer ticker.Stop()

			sc.tickerCh = ticker.C
		}
		// Call scrape method during initialization to ensure
		// that scrapers start from when the component starts
		// instead of waiting for the full duration to start.
		_ = sc.scrapeFunc(ctx, sc)
		for {
			select {
			case <-sc.tickerCh:
				_ = sc.scrapeFunc(ctx, sc)
			case <-sc.done:
				return
			}
		}
	})
}

func GetSettings(sType component.Type, rSet receiver.Settings) scraper.Settings {
	id := component.NewID(sType)
	telemetry := rSet.TelemetrySettings
	telemetry.Logger = telemetry.Logger.With(zap.String("scraper", id.String()))
	return scraper.Settings{
		ID:                id,
		TelemetrySettings: telemetry,
		BuildInfo:         rSet.BuildInfo,
	}
}
