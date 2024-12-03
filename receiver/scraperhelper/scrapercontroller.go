// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
)

// ScraperControllerOption apply changes to internal options.
type ScraperControllerOption interface {
	apply(*controller)
}

type scraperControllerOptionFunc func(*controller)

func (of scraperControllerOptionFunc) apply(e *controller) {
	of(e)
}

// AddScraper configures the provided scrape function to be called
// with the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddScraper(t component.Type, scraper scraper.Metrics) ScraperControllerOption {
	return scraperControllerOptionFunc(func(o *controller) {
		o.scrapers = append(o.scrapers, scraperWithID{
			Metrics: scraper,
			id:      component.NewID(t),
		})
	})
}

// Deprecated: [v0.115.0] use AddScraper.
func AddScraperWithType(t component.Type, scrp Scraper) ScraperControllerOption {
	// Ignore the error since it cannot happen because the Scrape func cannot be nil here.
	newScrp, _ := scraper.NewMetrics(scrp.Scrape, scraper.WithStart(scrp.Start), scraper.WithShutdown(scrp.Shutdown))
	return AddScraper(t, newScrp)
}

// WithTickerChannel allows you to override the scraper controller's ticker
// channel to specify when scrape is called. This is only expected to be
// used by tests.
func WithTickerChannel(tickerCh <-chan time.Time) ScraperControllerOption {
	return scraperControllerOptionFunc(func(o *controller) {
		o.tickerCh = tickerCh
	})
}

type controller struct {
	collectionInterval time.Duration
	initialDelay       time.Duration
	timeout            time.Duration
	nextConsumer       consumer.Metrics

	scrapers    []scraperWithID
	obsScrapers []scraper.Metrics

	tickerCh <-chan time.Time

	done chan struct{}
	wg   sync.WaitGroup

	obsrecv *receiverhelper.ObsReport
}

type scraperWithID struct {
	scraper.Metrics
	id component.ID
}

// NewScraperControllerReceiver creates a Receiver with the configured options, that can control multiple scrapers.
func NewScraperControllerReceiver(
	cfg *ControllerConfig,
	set receiver.Settings,
	nextConsumer consumer.Metrics,
	options ...ScraperControllerOption,
) (component.Component, error) {
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "",
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	sc := &controller{
		collectionInterval: cfg.CollectionInterval,
		initialDelay:       cfg.InitialDelay,
		timeout:            cfg.Timeout,
		nextConsumer:       nextConsumer,
		done:               make(chan struct{}),
		obsrecv:            obsrecv,
	}

	for _, op := range options {
		op.apply(sc)
	}

	sc.obsScrapers = make([]scraper.Metrics, len(sc.scrapers))
	for i := range sc.scrapers {
		telSet := set.TelemetrySettings
		telSet.Logger = telSet.Logger.With(zap.String("scraper", sc.scrapers[i].id.String()))
		var obsScrp scraper.ScrapeMetricsFunc
		obsScrp, err = newObsMetrics(sc.scrapers[i].ScrapeMetrics, set.ID, sc.scrapers[i].id, telSet)
		if err != nil {
			return nil, err
		}
		sc.obsScrapers[i], err = scraper.NewMetrics(obsScrp, scraper.WithStart(sc.scrapers[i].Start), scraper.WithShutdown(sc.scrapers[i].Shutdown))
		if err != nil {
			return nil, err
		}
	}

	return sc, nil
}

// Start the receiver, invoked during service start.
func (sc *controller) Start(ctx context.Context, host component.Host) error {
	for _, scrp := range sc.obsScrapers {
		if err := scrp.Start(ctx, host); err != nil {
			return err
		}
	}

	sc.startScraping()
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (sc *controller) Shutdown(ctx context.Context) error {
	// Signal the goroutine to stop.
	close(sc.done)
	sc.wg.Wait()
	var errs error
	for _, scrp := range sc.obsScrapers {
		errs = multierr.Append(errs, scrp.Shutdown(ctx))
	}

	return errs
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (sc *controller) startScraping() {
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
		// Call scrape method on initialization to ensure
		// that scrapers start from when the component starts
		// instead of waiting for the full duration to start.
		sc.scrapeMetricsAndReport()
		for {
			select {
			case <-sc.tickerCh:
				sc.scrapeMetricsAndReport()
			case <-sc.done:
				return
			}
		}
	}()
}

// scrapeMetricsAndReport calls the Scrape function for each of the configured
// Scrapers, records observability information, and passes the scraped metrics
// to the next component.
func (sc *controller) scrapeMetricsAndReport() {
	ctx, done := withScrapeContext(sc.timeout)
	defer done()

	metrics := pmetric.NewMetrics()
	for i := range sc.obsScrapers {
		md, err := sc.obsScrapers[i].ScrapeMetrics(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
	}

	dataPointCount := metrics.DataPointCount()
	ctx = sc.obsrecv.StartMetricsOp(ctx)
	err := sc.nextConsumer.ConsumeMetrics(ctx, metrics)
	sc.obsrecv.EndMetricsOp(ctx, "", dataPointCount, err)
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
