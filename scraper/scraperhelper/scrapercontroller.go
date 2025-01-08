// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper"

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
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

// AddMetricsScraper configures the provided scrape function to be called
// with the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddMetricsScraper(t component.Type, scraper scraper.Metrics) ScraperControllerOption {
	return scraperControllerOptionFunc(func(o *controller) {
		o.metricsScrapers = append(o.metricsScrapers, scraperWithID{
			Metrics: scraper,
			id:      component.NewID(t),
		})
	})
}

func AddLogsScraper(t component.Type, scraper scraper.Logs) ScraperControllerOption {
	return scraperControllerOptionFunc(func(o *controller) {
		o.logsScrapers = append(o.logsScrapers, scraperWithID{
			Logs: scraper,
			id:   component.NewID(t),
		})
	})
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

	nextMetrics        consumer.Metrics
	metricsScrapers    []scraperWithID
	obsMetricsScrapers []scraper.Metrics

	nextLogs        consumer.Logs
	logsScrapers    []scraperWithID
	obsLogsScrapers []scraper.Logs

	tickerCh <-chan time.Time

	done chan struct{}
	wg   sync.WaitGroup

	obsrecv *receiverhelper.ObsReport
}

type scraperWithID struct {
	scraper.Metrics
	scraper.Logs
	id component.ID
}

// NewMetricsScraperControllerReceiver creates a Receiver with the configured options, that can control multiple metricsScrapers.
func NewMetricsScraperControllerReceiver(
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
		nextMetrics:        nextConsumer,
		done:               make(chan struct{}),
		obsrecv:            obsrecv,
	}

	for _, op := range options {
		op.apply(sc)
	}

	sc.obsMetricsScrapers = make([]scraper.Metrics, len(sc.metricsScrapers))
	for i := range sc.metricsScrapers {
		telSet := set.TelemetrySettings
		telSet.Logger = telSet.Logger.With(zap.String("scraper", sc.metricsScrapers[i].id.String()))
		var obsScrp scraper.ScrapeMetricsFunc
		obsScrp, err = newObsMetrics(sc.metricsScrapers[i].ScrapeMetrics, set.ID, sc.metricsScrapers[i].id, telSet)
		if err != nil {
			return nil, err
		}
		sc.obsMetricsScrapers[i], err = scraper.NewMetrics(obsScrp, scraper.WithStart(sc.metricsScrapers[i].Metrics.Start), scraper.WithShutdown(sc.metricsScrapers[i].Metrics.Shutdown))
		if err != nil {
			return nil, err
		}
	}

	return sc, nil
}

// NewLogsScraperControllerReceiver creates a Receiver with the configured options, that can control multiple logsScrapers.
func NewLogsScraperControllerReceiver(
	cfg *ControllerConfig,
	set receiver.Settings,
	nextConsumer consumer.Logs,
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
		nextLogs:           nextConsumer,
		done:               make(chan struct{}),
		obsrecv:            obsrecv,
	}

	for _, op := range options {
		op.apply(sc)
	}

	sc.obsLogsScrapers = make([]scraper.Logs, len(sc.logsScrapers))
	for i := range sc.logsScrapers {
		telSet := set.TelemetrySettings
		telSet.Logger = telSet.Logger.With(zap.String("scraper", sc.logsScrapers[i].id.String()))
		var obsScrp scraper.ScrapeLogsFunc
		// TODO: add an obs for logs
		obsScrp, err = newObsLogs(sc.logsScrapers[i].ScrapeLogs, set.ID, sc.logsScrapers[i].id, telSet)
		if err != nil {
			return nil, err
		}
		sc.obsLogsScrapers[i], err = scraper.NewLogs(obsScrp, scraper.WithStart(sc.logsScrapers[i].Logs.Start), scraper.WithShutdown(sc.logsScrapers[i].Logs.Shutdown))
		if err != nil {
			return nil, err
		}
	}

	return sc, nil
}

// Start the receiver, invoked during service start.
func (sc *controller) Start(ctx context.Context, host component.Host) error {
	if sc.nextMetrics != nil {
		for _, scrp := range sc.obsMetricsScrapers {
			if err := scrp.Start(ctx, host); err != nil {
				return err
			}
		}
	} else if sc.nextLogs != nil {
		for _, scrp := range sc.obsLogsScrapers {
			if err := scrp.Start(ctx, host); err != nil {
				return err
			}
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
	for _, scrp := range sc.obsMetricsScrapers {
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
		if sc.nextMetrics != nil {
			sc.scrapeMetricsAndReport()
			for {
				select {
				case <-sc.tickerCh:
					sc.scrapeMetricsAndReport()
				case <-sc.done:
					return
				}
			}
		} else if sc.nextLogs != nil {
			sc.scrapeLogsAndReport()
			for {
				select {
				case <-sc.tickerCh:
					sc.scrapeLogsAndReport()
				case <-sc.done:
					return
				}
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
	for i := range sc.obsMetricsScrapers {
		md, err := sc.obsMetricsScrapers[i].ScrapeMetrics(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
	}

	dataPointCount := metrics.DataPointCount()
	ctx = sc.obsrecv.StartMetricsOp(ctx)
	err := sc.nextMetrics.ConsumeMetrics(ctx, metrics)
	sc.obsrecv.EndMetricsOp(ctx, "", dataPointCount, err)
}

// scrapeLogsAndReport calls the Scrape function for each of the configured
// Scrapers, records observability information, and passes the scraped logs
// to the next component.
func (sc *controller) scrapeLogsAndReport() {
	ctx, done := withScrapeContext(sc.timeout)
	defer done()

	logs := plog.NewLogs()
	for i := range sc.obsLogsScrapers {
		md, err := sc.obsLogsScrapers[i].ScrapeLogs(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
	}

	logRecordCount := logs.LogRecordCount()
	ctx = sc.obsrecv.StartMetricsOp(ctx)
	err := sc.nextLogs.ConsumeLogs(ctx, logs)
	sc.obsrecv.EndMetricsOp(ctx, "", logRecordCount, err)
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
