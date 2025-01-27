// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper"

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
)

// ControllerOption apply changes to internal options.
type ControllerOption interface {
	apply(*controllerOptions)
}

type optionFunc func(*controllerOptions)

func (of optionFunc) apply(e *controllerOptions) {
	of(e)
}

// AddScraper configures the scraper.Metrics to be called with the specified options,
// and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddScraper(t component.Type, sc scraper.Metrics) ControllerOption {
	f := scraper.NewFactory(t, nil,
		scraper.WithMetrics(func(context.Context, scraper.Settings, component.Config) (scraper.Metrics, error) {
			return sc, nil
		}, component.StabilityLevelAlpha))
	return AddFactoryWithConfig(f, nil)
}

// AddFactoryWithConfig configures the scraper.Factory and associated config that
// will be used to create a new scraper. The created scraper will be called with
// the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddFactoryWithConfig(f scraper.Factory, cfg component.Config) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.factoriesWithConfig = append(o.factoriesWithConfig, factoryWithConfig{f: f, cfg: cfg})
	})
}

// WithTickerChannel allows you to override the scraper controller's ticker
// channel to specify when scrape is called. This is only expected to be
// used by tests.
func WithTickerChannel(tickerCh <-chan time.Time) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.tickerCh = tickerCh
	})
}

type factoryWithConfig struct {
	f   scraper.Factory
	cfg component.Config
}

type controllerOptions struct {
	tickerCh            <-chan time.Time
	factoriesWithConfig []factoryWithConfig
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
	var errs error
	for _, scrp := range sc.scrapers {
		errs = multierr.Append(errs, scrp.Shutdown(ctx))
	}

	return errs
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

// NewLogsController creates a receiver.Logs with the configured options, that can control multiple scraper.Logs.
func NewLogsController(cfg *ControllerConfig,
	rSet receiver.Settings,
	nextConsumer consumer.Logs,
	options ...ControllerOption,
) (receiver.Logs, error) {
	co := getOptions(options)
	scrapers := make([]scraper.Logs, 0, len(co.factoriesWithConfig))
	for _, fwc := range co.factoriesWithConfig {
		set := getSettings(fwc.f.Type(), rSet)
		s, err := fwc.f.CreateLogs(context.Background(), set, fwc.cfg)
		if err != nil {
			return nil, err
		}
		s, err = wrapObsLogs(s, rSet.ID, set.ID, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}
	return newController[scraper.Logs](
		cfg, rSet, scrapers, func(c *controller[scraper.Logs]) { scrapeLogs(c, nextConsumer) }, co.tickerCh)
}

// NewMetricsController creates a receiver.Metrics with the configured options, that can control multiple scraper.Metrics.
func NewMetricsController(cfg *ControllerConfig,
	rSet receiver.Settings,
	nextConsumer consumer.Metrics,
	options ...ControllerOption,
) (receiver.Metrics, error) {
	co := getOptions(options)
	scrapers := make([]scraper.Metrics, 0, len(co.factoriesWithConfig))
	for _, fwc := range co.factoriesWithConfig {
		set := getSettings(fwc.f.Type(), rSet)
		s, err := fwc.f.CreateMetrics(context.Background(), set, fwc.cfg)
		if err != nil {
			return nil, err
		}
		s, err = wrapObsMetrics(s, rSet.ID, set.ID, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}
	return newController[scraper.Metrics](
		cfg, rSet, scrapers, func(c *controller[scraper.Metrics]) { scrapeMetrics(c, nextConsumer) }, co.tickerCh)
}

func scrapeLogs(c *controller[scraper.Logs], nextConsumer consumer.Logs) {
	ctx, done := withScrapeContext(c.timeout)
	defer done()

	logs := plog.NewLogs()
	for i := range c.scrapers {
		md, err := c.scrapers[i].ScrapeLogs(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
	}

	logRecordCount := logs.LogRecordCount()
	ctx = c.obsrecv.StartMetricsOp(ctx)
	err := nextConsumer.ConsumeLogs(ctx, logs)
	c.obsrecv.EndMetricsOp(ctx, "", logRecordCount, err)
}

func scrapeMetrics(c *controller[scraper.Metrics], nextConsumer consumer.Metrics) {
	ctx, done := withScrapeContext(c.timeout)
	defer done()

	metrics := pmetric.NewMetrics()
	for i := range c.scrapers {
		md, err := c.scrapers[i].ScrapeMetrics(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
	}

	dataPointCount := metrics.DataPointCount()
	ctx = c.obsrecv.StartMetricsOp(ctx)
	err := nextConsumer.ConsumeMetrics(ctx, metrics)
	c.obsrecv.EndMetricsOp(ctx, "", dataPointCount, err)
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

// withScrapeContext will return a context that has no deadline if timeout is 0
// which implies no explicit timeout had occurred, otherwise, a context
// with a deadline of the provided timeout is returned.
func withScrapeContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return context.WithCancel(context.Background())
	}
	return context.WithTimeout(context.Background(), timeout)
}
