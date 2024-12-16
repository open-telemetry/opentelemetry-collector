// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/plog"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
)

// LogsScraperControllerOption apply changes to internal options.
type LogsScraperControllerOption interface {
	apply(*logsController)
}

type logsScraperControllerOptionFunc func(*logsController)

func (of logsScraperControllerOptionFunc) apply(e *logsController) {
	of(e)
}

// AddLogsScraper configures the provided scrape function to be called
// with the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped logs
// will be passed to the next consumer.
func AddLogsScraper(t component.Type, scraper scraper.Logs) LogsScraperControllerOption {
	return logsScraperControllerOptionFunc(func(o *logsController) {
		o.scrapers = append(o.scrapers, logsScraperWithID{
			Logs: scraper,
			id:   component.NewID(t),
		})
	})
}

// WithLogsTickerChannel allows you to override the scraper controller's ticker
// channel to specify when scrape is called. This is only expected to be
// used by tests.
func WithLogsTickerChannel(tickerCh <-chan time.Time) LogsScraperControllerOption {
	return logsScraperControllerOptionFunc(func(o *logsController) {
		o.tickerCh = tickerCh
	})
}

type logsController struct {
	collectionInterval time.Duration
	initialDelay       time.Duration
	timeout            time.Duration
	nextConsumer       consumer.Logs

	scrapers    []logsScraperWithID
	obsScrapers []scraper.Logs

	tickerCh <-chan time.Time

	done chan struct{}
	wg   sync.WaitGroup

	obsrecv *receiverhelper.ObsReport
}
type logsScraperWithID struct {
	scraper.Logs
	id component.ID
}

// NewLogsScraperControllerReceiver creates a Receiver with the configured options, that can control multiple scrapers.
func NewLogsScraperControllerReceiver(
	cfg *ControllerConfig,
	set receiver.Settings,
	nextConsumer consumer.Logs,
	options ...LogsScraperControllerOption,
) (component.Component, error) {
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "",
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	sc := &logsController{
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

	sc.obsScrapers = make([]scraper.Logs, len(sc.scrapers))
	for i := range sc.scrapers {
		telSet := set.TelemetrySettings
		telSet.Logger = telSet.Logger.With(zap.String("scraper", sc.scrapers[i].id.String()))
		var obsScrp scraper.ScrapeLogsFunc
		obsScrp, err = newObsLogs(sc.scrapers[i].ScrapeLogs, set.ID, sc.scrapers[i].id, telSet)
		if err != nil {
			return nil, err
		}
		sc.obsScrapers[i], err = scraper.NewLogs(obsScrp, scraper.WithStart(sc.scrapers[i].Start), scraper.WithShutdown(sc.scrapers[i].Shutdown))
		if err != nil {
			return nil, err
		}
	}

	return sc, nil
}

// Start the receiver, invoked during service start.
func (sc *logsController) Start(ctx context.Context, host component.Host) error {
	for _, scrp := range sc.obsScrapers {
		if err := scrp.Start(ctx, host); err != nil {
			return err
		}
	}

	sc.startScraping()
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (sc *logsController) Shutdown(ctx context.Context) error {
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
func (sc *logsController) startScraping() {
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
		sc.scrapeLogsAndReport()
		for {
			select {
			case <-sc.tickerCh:
				sc.scrapeLogsAndReport()
			case <-sc.done:
				return
			}
		}
	}()
}

// scrapeLogsAndReport calls the Scrape function for each of the configured
// Scrapers, records observability information, and passes the scraped logs
// to the next component.
func (sc *logsController) scrapeLogsAndReport() {
	ctx, done := withScrapeContext(sc.timeout)
	defer done()

	logs := plog.NewLogs()
	for i := range sc.obsScrapers {
		md, err := sc.obsScrapers[i].ScrapeLogs(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
	}

	logRecordCount := logs.LogRecordCount()
	ctx = sc.obsrecv.StartLogsOp(ctx)
	err := sc.nextConsumer.ConsumeLogs(ctx, logs)
	sc.obsrecv.EndLogsOp(ctx, "", logRecordCount, err)
}
