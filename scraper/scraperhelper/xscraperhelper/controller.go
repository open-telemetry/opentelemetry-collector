// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xscraperhelper provides utilities for scrapers.
package xscraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper"

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/xscraper"
)

type factoryWithConfig struct {
	f   xscraper.Factory
	cfg component.Config
}

type controllerOptions struct {
	tickerCh            <-chan time.Time
	factoriesWithConfig []factoryWithConfig
}

// ControllerOption apply changes to internal options.
type ControllerOption interface {
	apply(*controllerOptions)
}

type optionFunc func(*controllerOptions)

func (of optionFunc) apply(e *controllerOptions) {
	of(e)
}

// AddFactoryWithConfig configures the scraper.Factory and associated config that
// will be used to create a new scraper. The created scraper will be called with
// the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddFactoryWithConfig(f xscraper.Factory, cfg component.Config) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.factoriesWithConfig = append(o.factoriesWithConfig, factoryWithConfig{f: f, cfg: cfg})
	})
}

// NewProfilesController creates a receiver.Profiles with the configured options, that can control multiple xscraper.Profiles.
func NewProfilesController(cfg *ControllerConfig,
	rSet receiver.Settings,
	nextConsumer xconsumer.Profiles,
	options ...ControllerOption,
) (xreceiver.Profiles, error) {
	co := getOptions(options)
	scrapers := make([]xscraper.Profiles, 0, len(co.factoriesWithConfig))
	for _, fwc := range co.factoriesWithConfig {
		set := getSettings(fwc.f.Type(), rSet)
		s, err := fwc.f.CreateProfiles(context.Background(), set, fwc.cfg)
		if err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}
	return newController[xscraper.Profiles](
		cfg, rSet, scrapers, func(c *controller[xscraper.Profiles]) { scrapeProfiles(c, nextConsumer) }, co.tickerCh)
}

func getOptions(options []ControllerOption) controllerOptions {
	co := controllerOptions{}
	for _, op := range options {
		op.apply(&co)
	}
	return co
}

func getSettings(sType component.Type, rSet receiver.Settings) scraper.Settings {
	return scraper.Settings{
		ID:                component.NewID(sType),
		TelemetrySettings: rSet.TelemetrySettings,
		BuildInfo:         rSet.BuildInfo,
	}
}

func scrapeProfiles(c *controller[xscraper.Profiles], nextConsumer xconsumer.Profiles) {
	ctx, done := withScrapeContext(c.timeout)
	defer done()

	profiles := pprofile.NewProfiles()
	for i := range c.scrapers {
		md, err := c.scrapers[i].ScrapeProfiles(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
	}

	profilesRecordCount := profiles.SampleCount()
	ctx = c.obsrecv.StartMetricsOp(ctx)
	err := nextConsumer.ConsumeProfiles(ctx, profiles)
	c.obsrecv.EndMetricsOp(ctx, "", profilesRecordCount, err)
}

// withScrapeContext will return a context that has no deadline if timeout is 0
// which implies no explicit timeout had occurred, otherwise, a context
// with a deadline of the provided timeout is returned.
func withScrapeContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return context.WithCancel(context.Background())
	}
	return context.WithTimeout(context.Background(), timeout)
}

type controller[T component.Component] struct {
	collectionInterval time.Duration
	initialDelay       time.Duration
	timeout            time.Duration

	scrapers   []T
	scrapeFunc func(*controller[T])
	tickerCh   <-chan time.Time

	done chan struct{}
	wg   sync.WaitGroup

	obsrecv *receiverhelper.ObsReport
}

func newController[T component.Component](
	cfg *ControllerConfig,
	rSet receiver.Settings,
	scrapers []T,
	scrapeFunc func(*controller[T]),
	tickerCh <-chan time.Time,
) (*controller[T], error) {
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             rSet.ID,
		Transport:              "",
		ReceiverCreateSettings: rSet,
	})
	if err != nil {
		return nil, err
	}

	cs := &controller[T]{
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

// Start the receiver, invoked during service start.
func (sc *controller[T]) Start(ctx context.Context, host component.Host) error {
	for _, scrp := range sc.scrapers {
		if err := scrp.Start(ctx, host); err != nil {
			return err
		}
	}

	sc.startScraping()
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (sc *controller[T]) Shutdown(ctx context.Context) error {
	// Signal the goroutine to stop.
	close(sc.done)
	sc.wg.Wait()
	var errs []error
	for _, scrp := range sc.scrapers {
		errs = append(errs, scrp.Shutdown(ctx))
	}

	return errors.Join(errs...)
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (sc *controller[T]) startScraping() {
	sc.wg.Add(1)
	go func() {
		defer sc.wg.Done()
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
	}()
}
