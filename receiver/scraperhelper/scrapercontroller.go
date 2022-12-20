// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"errors"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

// ScraperControllerSettings defines common settings for a scraper controller
// configuration. Scraper controller receivers can embed this struct, instead
// of receiver.Settings, and extend it with more fields if needed.
type ScraperControllerSettings struct {
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
}

// NewDefaultScraperControllerSettings returns default scraper controller
// settings with a collection interval of one minute.
func NewDefaultScraperControllerSettings(component.Type) ScraperControllerSettings {
	return ScraperControllerSettings{
		CollectionInterval: time.Minute,
	}
}

// ScraperControllerOption apply changes to internal options.
type ScraperControllerOption func(*controller)

// AddScraper configures the provided scrape function to be called
// with the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddScraper(scraper Scraper) ScraperControllerOption {
	return func(o *controller) {
		o.scrapers = append(o.scrapers, scraper)
	}
}

// WithTickerChannel allows you to override the scraper controllers ticker
// channel to specify when scrape is called. This is only expected to be
// used by tests.
func WithTickerChannel(tickerCh <-chan time.Time) ScraperControllerOption {
	return func(o *controller) {
		o.tickerCh = tickerCh
	}
}

type controller struct {
	id                 component.ID
	logger             *zap.Logger
	collectionInterval time.Duration
	nextConsumer       consumer.Metrics

	scrapers    []Scraper
	obsScrapers []*obsreport.Scraper

	tickerCh <-chan time.Time

	initialized bool
	done        chan struct{}
	terminated  chan struct{}

	obsrecv      *obsreport.Receiver
	recvSettings receiver.CreateSettings
}

// NewScraperControllerReceiver creates a Receiver with the configured options, that can control multiple scrapers.
func NewScraperControllerReceiver(
	cfg *ScraperControllerSettings,
	set receiver.CreateSettings,
	nextConsumer consumer.Metrics,
	options ...ScraperControllerOption,
) (component.Component, error) {
	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	if cfg.CollectionInterval <= 0 {
		return nil, errors.New("collection_interval must be a positive duration")
	}

	obsrecv, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             set.ID,
		Transport:              "",
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	sc := &controller{
		id:                 set.ID,
		logger:             set.Logger,
		collectionInterval: cfg.CollectionInterval,
		nextConsumer:       nextConsumer,
		done:               make(chan struct{}),
		terminated:         make(chan struct{}),
		obsrecv:            obsrecv,
		recvSettings:       set,
	}

	for _, op := range options {
		op(sc)
	}

	sc.obsScrapers = make([]*obsreport.Scraper, len(sc.scrapers))
	for i, scraper := range sc.scrapers {
		scrp, err := obsreport.NewScraper(obsreport.ScraperSettings{
			ReceiverID:             sc.id,
			Scraper:                scraper.ID(),
			ReceiverCreateSettings: sc.recvSettings,
		})

		sc.obsScrapers[i] = scrp

		if err != nil {
			return nil, err
		}
	}

	return sc, nil
}

// Start the receiver, invoked during service start.
func (sc *controller) Start(ctx context.Context, host component.Host) error {
	for _, scraper := range sc.scrapers {
		if err := scraper.Start(ctx, host); err != nil {
			return err
		}
	}

	sc.initialized = true
	sc.startScraping()
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (sc *controller) Shutdown(ctx context.Context) error {
	sc.stopScraping()

	// wait until scraping ticker has terminated
	if sc.initialized {
		<-sc.terminated
	}

	var errs error
	for _, scraper := range sc.scrapers {
		errs = multierr.Append(errs, scraper.Shutdown(ctx))
	}

	return errs
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (sc *controller) startScraping() {
	go func() {
		if sc.tickerCh == nil {
			ticker := time.NewTicker(sc.collectionInterval)
			defer ticker.Stop()

			sc.tickerCh = ticker.C
		}

		for {
			select {
			case <-sc.tickerCh:
				sc.scrapeMetricsAndReport(context.Background())
			case <-sc.done:
				sc.terminated <- struct{}{}
				return
			}
		}
	}()
}

// scrapeMetricsAndReport calls the Scrape function for each of the configured
// Scrapers, records observability information, and passes the scraped metrics
// to the next component.
func (sc *controller) scrapeMetricsAndReport(ctx context.Context) {
	metrics := pmetric.NewMetrics()

	for i, scraper := range sc.scrapers {
		scrp := sc.obsScrapers[i]
		ctx = scrp.StartMetricsOp(ctx)
		md, err := scraper.Scrape(ctx)

		if err != nil {
			sc.logger.Error("Error scraping metrics", zap.Error(err), zap.Stringer("scraper", scraper.ID()))
			if !scrapererror.IsPartialScrapeError(err) {
				scrp.EndMetricsOp(ctx, 0, err)
				continue
			}
		}
		scrp.EndMetricsOp(ctx, md.MetricCount(), err)
		md.ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
	}

	dataPointCount := metrics.DataPointCount()
	ctx = sc.obsrecv.StartMetricsOp(ctx)
	err := sc.nextConsumer.ConsumeMetrics(ctx, metrics)
	sc.obsrecv.EndMetricsOp(ctx, "", dataPointCount, err)
}

// stopScraping stops the ticker
func (sc *controller) stopScraping() {
	close(sc.done)
}
