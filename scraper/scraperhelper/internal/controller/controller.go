// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package controller provides functionality used in scraperhelper and xscraperhelper.

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/scraper"
)

type Controller[T component.Component] struct {
	collectionInterval time.Duration
	initialDelay       time.Duration
	Timeout            time.Duration

	Scrapers   []T
	scrapeFunc func(*Controller[T])
	tickerCh   <-chan time.Time

	done chan struct{}
	wg   sync.WaitGroup

	Obsrecv *receiverhelper.ObsReport
}

func NewController[T component.Component](
	cfg *ControllerConfig,
	rSet receiver.Settings,
	scrapers []T,
	scrapeFunc func(*Controller[T]),
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
		Obsrecv:            obsrecv,
	}

	return cs, nil
}

// Start the receiver, invoked during service start.
func (sc *Controller[T]) Start(ctx context.Context, host component.Host) error {
	for _, scrp := range sc.Scrapers {
		if err := scrp.Start(ctx, host); err != nil {
			return err
		}
	}

	sc.startScraping()
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (sc *Controller[T]) Shutdown(ctx context.Context) error {
	// Signal the goroutine to stop.
	close(sc.done)
	sc.wg.Wait()
	var errs error
	for _, scrp := range sc.Scrapers {
		errs = multierr.Append(errs, scrp.Shutdown(ctx))
	}

	return errs
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (sc *Controller[T]) startScraping() {
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
		sc.scrapeFunc(sc)
		for {
			select {
			case <-sc.tickerCh:
				sc.scrapeFunc(sc)
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

// WithScrapeContext will return a context that has no deadline if timeout is 0
// which implies no explicit timeout had occurred, otherwise, a context
// with a deadline of the provided timeout is returned.
func WithScrapeContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return context.WithCancel(context.Background())
	}
	return context.WithTimeout(context.Background(), timeout)
}
