// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package controller provides functionality used in scraperhelper and xscraperhelper.

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/scraper"
)

// Schedule defines one collection schedule: an interval and the function to run at each tick.
// Used by NewControllerWithSchedules to run multiple collection intervals (e.g. per metric or per event).
type Schedule[T component.Component] struct {
	Config     ScheduleConfig
	ScrapeFunc func(*Controller[T])
}

type Controller[T component.Component] struct {
	collectionInterval time.Duration
	initialDelay       time.Duration
	timeout            time.Duration

	scrapers   []T
	scrapeFunc func(*Controller[T])
	tickerCh   <-chan time.Time

	// schedules is used when running multiple collection intervals (NewControllerWithSchedules).
	schedules []Schedule[T]

	done chan struct{}
	wg   sync.WaitGroup

	obsrecv *receiverhelper.ObsReport
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
		timeout:            cfg.Timeout,
		scrapers:           scrapers,
		scrapeFunc:         scrapeFunc,
		done:               make(chan struct{}),
		tickerCh:           tickerCh,
		obsrecv:            obsrecv,
	}

	return cs, nil
}

// NewControllerWithSchedules creates a controller that runs multiple collection schedules,
// each with its own interval and scrape function. This allows receivers to collect
// different metrics or events at different intervals (e.g. per-metric or per-event collection_interval).
// Each schedule runs in its own goroutine with its own ticker.
func NewControllerWithSchedules[T component.Component](
	rSet receiver.Settings,
	scrapers []T,
	schedules []Schedule[T],
) (*Controller[T], error) {
	if len(schedules) == 0 {
		return nil, errors.New("at least one schedule is required")
	}
	var errs error
	for i := range schedules {
		if err := schedules[i].Config.Validate(); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("schedule %d: %w", i, err))
		}
	}
	if errs != nil {
		return nil, errs
	}

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             rSet.ID,
		Transport:              "",
		ReceiverCreateSettings: rSet,
	})
	if err != nil {
		return nil, err
	}

	// Copy schedules slice so caller cannot mutate after creation.
	schedulesCopy := make([]Schedule[T], len(schedules))
	copy(schedulesCopy, schedules)

	cs := &Controller[T]{
		scrapers:  scrapers,
		schedules: schedulesCopy,
		done:      make(chan struct{}),
		obsrecv:   obsrecv,
		timeout:   0, // not used when schedules are set; each schedule uses its own Config.Timeout
	}

	return cs, nil
}

// Scrapers returns the scrapers managed by this controller.
func (sc *Controller[T]) Scrapers() []T { return sc.scrapers }

// Timeout returns the default timeout for scrape context (zero when using multiple schedules).
func (sc *Controller[T]) Timeout() time.Duration { return sc.timeout }

// Obsrecv returns the observability reporter for this controller.
func (sc *Controller[T]) Obsrecv() *receiverhelper.ObsReport { return sc.obsrecv }

// Start the receiver, invoked during service start.
func (sc *Controller[T]) Start(ctx context.Context, host component.Host) error {
	for _, scrp := range sc.scrapers {
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
	for _, scrp := range sc.scrapers {
		errs = multierr.Append(errs, scrp.Shutdown(ctx))
	}

	return errs
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval. When multiple schedules are configured, starts one goroutine per schedule.
func (sc *Controller[T]) startScraping() {
	if len(sc.schedules) > 0 {
		for i := range sc.schedules {
			i := i
			sched := &sc.schedules[i]
			sc.wg.Go(func() {
				sc.runSchedule(sched)
			})
		}
		return
	}

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

// runSchedule runs one schedule: initial delay, then ticker loop calling ScrapeFunc.
func (sc *Controller[T]) runSchedule(sched *Schedule[T]) {
	cfg := &sched.Config
	if cfg.InitialDelay > 0 {
		select {
		case <-time.After(cfg.InitialDelay):
		case <-sc.done:
			return
		}
	}

	ticker := time.NewTicker(cfg.CollectionInterval)
	defer ticker.Stop()

	// Initial scrape so collection starts immediately.
	sched.ScrapeFunc(sc)
	for {
		select {
		case <-ticker.C:
			sched.ScrapeFunc(sc)
		case <-sc.done:
			return
		}
	}
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
