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

package receiverhelper

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// ScraperControllerSettings defines common settings for a scraper controller
// configuration. Scraper controller receivers can embed this struct, instead
// of configmodels.ReceiverSettings, and extend it with more fields if needed.
type ScraperControllerSettings struct {
	configmodels.ReceiverSettings `mapstructure:"squash"`
	CollectionInterval            time.Duration `mapstructure:"collection_interval"`
}

// DefaultScraperControllerSettings returns default scraper controller
// settings with a collection interval of one minute.
func DefaultScraperControllerSettings(cfgType configmodels.Type) ScraperControllerSettings {
	return ScraperControllerSettings{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: string(cfgType),
			TypeVal: cfgType,
		},
		CollectionInterval: time.Minute,
	}
}

// ScraperControllerOption apply changes to internal options.
type ScraperControllerOption func(*scraperController)

// AddMetricsScraper configures the provided scrape function to be called
// with the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddMetricsScraper(scraper MetricsScraper) ScraperControllerOption {
	return func(o *scraperController) {
		o.metricsScrapers.scrapers = append(o.metricsScrapers.scrapers, scraper)
	}
}

// AddResourceMetricsScraper configures the provided scrape function to
// be called with the specified options, and at the specified collection
// interval.
//
// Observability information will be reported, and the scraped resource
// metrics will be passed to the next consumer.
func AddResourceMetricsScraper(scraper ResourceMetricsScraper) ScraperControllerOption {
	return func(o *scraperController) {
		o.resourceMetricScrapers = append(o.resourceMetricScrapers, scraper)
	}
}

// WithTickerChannel allows you to override the scraper controllers ticker
// channel to specify when scrape is called. This is only expected to be
// used by tests.
func WithTickerChannel(tickerCh <-chan time.Time) ScraperControllerOption {
	return func(o *scraperController) {
		o.tickerCh = tickerCh
	}
}

type scraperController struct {
	name               string
	collectionInterval time.Duration
	nextConsumer       consumer.MetricsConsumer

	metricsScrapers        *multiMetricScraper
	resourceMetricScrapers []ResourceMetricsScraper

	tickerCh <-chan time.Time

	initialized bool
	done        chan struct{}
	terminated  chan struct{}
}

// NewScraperControllerReceiver creates a Receiver with the configured options, that can control multiple scrapers.
func NewScraperControllerReceiver(cfg *ScraperControllerSettings, nextConsumer consumer.MetricsConsumer, options ...ScraperControllerOption) (component.Receiver, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	if cfg.CollectionInterval <= 0 {
		return nil, errors.New("collection_interval must be a positive duration")
	}

	sc := &scraperController{
		name:               cfg.Name(),
		collectionInterval: cfg.CollectionInterval,
		nextConsumer:       nextConsumer,
		metricsScrapers:    &multiMetricScraper{},
		done:               make(chan struct{}),
		terminated:         make(chan struct{}),
	}

	for _, op := range options {
		op(sc)
	}

	if len(sc.metricsScrapers.scrapers) > 0 {
		sc.resourceMetricScrapers = append(sc.resourceMetricScrapers, sc.metricsScrapers)
	}

	return sc, nil
}

// Start the receiver, invoked during service start.
func (sc *scraperController) Start(ctx context.Context, _ component.Host) error {
	if err := sc.initializeScrapers(ctx); err != nil {
		return err
	}

	sc.initialized = true
	sc.startScraping()
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (sc *scraperController) Shutdown(ctx context.Context) error {
	sc.stopScraping()

	// wait until scraping ticker has terminated
	if sc.initialized {
		<-sc.terminated
	}

	var errors []error

	if err := sc.closeScrapers(ctx); err != nil {
		errors = append(errors, err)
	}

	return componenterror.CombineErrors(errors)
}

// initializeScrapers initializes all the scrapers
func (sc *scraperController) initializeScrapers(ctx context.Context) error {
	for _, scraper := range sc.resourceMetricScrapers {
		if err := scraper.Initialize(ctx); err != nil {
			return err
		}
	}

	return nil
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (sc *scraperController) startScraping() {
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
func (sc *scraperController) scrapeMetricsAndReport(ctx context.Context) {
	ctx = obsreport.ReceiverContext(ctx, sc.name, "", "")

	metrics := pdata.NewMetrics()

	for _, rms := range sc.resourceMetricScrapers {
		resourceMetrics, err := rms.Scrape(ctx, sc.name)
		if err != nil && !consumererror.IsPartialScrapeError(err) {
			continue
		}

		resourceMetrics.MoveAndAppendTo(metrics.ResourceMetrics())
	}

	metricCount, dataPointCount := metrics.MetricAndDataPointCount()

	ctx = obsreport.StartMetricsReceiveOp(ctx, sc.name, "")
	err := sc.nextConsumer.ConsumeMetrics(ctx, metrics)
	obsreport.EndMetricsReceiveOp(ctx, "", metricCount, dataPointCount, err)
}

// stopScraping stops the ticker
func (sc *scraperController) stopScraping() {
	close(sc.done)
}

// closeScrapers closes all the scrapers
func (sc *scraperController) closeScrapers(ctx context.Context) error {
	var errs []error

	for _, scraper := range sc.resourceMetricScrapers {
		if err := scraper.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}

var _ ResourceMetricsScraper = (*multiMetricScraper)(nil)

type multiMetricScraper struct {
	scrapers []MetricsScraper
}

func (mms *multiMetricScraper) Name() string {
	return ""
}

func (mms *multiMetricScraper) Initialize(ctx context.Context) error {
	for _, scraper := range mms.scrapers {
		if err := scraper.Initialize(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (mms *multiMetricScraper) Close(ctx context.Context) error {
	var errs []error
	for _, scraper := range mms.scrapers {
		if err := scraper.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

func (mms *multiMetricScraper) Scrape(ctx context.Context, receiverName string) (pdata.ResourceMetricsSlice, error) {
	rms := pdata.NewResourceMetricsSlice()
	rms.Resize(1)
	rm := rms.At(0)
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	ilm := ilms.At(0)

	var errs []error
	for _, scraper := range mms.scrapers {
		metrics, err := scraper.Scrape(ctx, receiverName)
		if err != nil {
			errs = append(errs, err)
			if !consumererror.IsPartialScrapeError(err) {
				continue
			}
		}

		metrics.MoveAndAppendTo(ilm.Metrics())
	}
	return rms, CombineScrapeErrors(errs)
}
