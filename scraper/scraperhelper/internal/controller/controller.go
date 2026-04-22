// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package controller provides functionality used in scraperhelper and xscraperhelper.

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

	Scrapers []T
	// scraperCollectionIntervals holds optional per-scraper collection intervals.
	// A value of zero means use collectionInterval for that scraper.
	scraperCollectionIntervals []time.Duration
	scrapeFunc                 func(*Controller[T], []int)
	tickerCh                   <-chan time.Time

	done chan struct{}
	wg   sync.WaitGroup

	Obsrecv *receiverhelper.ObsReport
}

func NewController[T component.Component](
	cfg *ControllerConfig,
	rSet receiver.Settings,
	scrapers []T,
	scrapeFunc func(*Controller[T], []int),
	tickerCh <-chan time.Time,
	scraperCollectionIntervals []time.Duration,
) (*Controller[T], error) {
	if len(scraperCollectionIntervals) == 0 {
		scraperCollectionIntervals = make([]time.Duration, len(scrapers))
	} else if len(scraperCollectionIntervals) != len(scrapers) {
		return nil, fmt.Errorf("scraperhelper: scraper collection intervals length %d must match scrapers length %d", len(scraperCollectionIntervals), len(scrapers))
	}
	for _, d := range scraperCollectionIntervals {
		if d < 0 {
			return nil, errors.New("scraperhelper: per-scraper collection_interval must not be negative")
		}
	}

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             rSet.ID,
		Transport:              "",
		ReceiverCreateSettings: rSet,
	})
	if err != nil {
		return nil, err
	}

	cs := &Controller[T]{
		collectionInterval:         cfg.CollectionInterval,
		initialDelay:               cfg.InitialDelay,
		Timeout:                    cfg.Timeout,
		Scrapers:                   scrapers,
		scraperCollectionIntervals: scraperCollectionIntervals,
		scrapeFunc:                 scrapeFunc,
		done:                       make(chan struct{}),
		tickerCh:                   tickerCh,
		Obsrecv:                    obsrecv,
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

// startScraping initiates tickers that call Scrape based on the configured
// collection intervals. When tickerCh is set (tests), every tick scrapes all
// scrapers and per-scraper intervals are ignored for scheduling.
func (sc *Controller[T]) startScraping() {
	sc.wg.Go(func() {
		if sc.initialDelay > 0 {
			select {
			case <-time.After(sc.initialDelay):
			case <-sc.done:
				return
			}
		}

		// Test hook: a single external channel drives full scrapes.
		if sc.tickerCh != nil {
			sc.scrapeFunc(sc, nil)
			for {
				select {
				case <-sc.tickerCh:
					sc.scrapeFunc(sc, nil)
				case <-sc.done:
					return
				}
			}
		}

		// Initial scrape includes all scrapers.
		sc.scrapeFunc(sc, nil)

		effective := make([]time.Duration, len(sc.Scrapers))
		for i := range sc.Scrapers {
			d := sc.scraperCollectionIntervals[i]
			if d <= 0 {
				d = sc.collectionInterval
			}
			effective[i] = d
		}

		groups := map[time.Duration][]int{}
		for i, d := range effective {
			groups[d] = append(groups[d], i)
		}

		var inner sync.WaitGroup
		for interval, groupIndices := range groups {
			inner.Add(1)
			clonedIndices := slices.Clone(groupIndices)
			go func() {
				defer inner.Done()
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						sc.scrapeFunc(sc, clonedIndices)
					case <-sc.done:
						return
					}
				}
			}()
		}
		inner.Wait()
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
