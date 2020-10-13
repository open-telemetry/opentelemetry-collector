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
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// ScraperControllerOption apply changes to internal options.
type ScraperControllerOption func(*scraperController)

// WithDefaultCollectionInterval overrides the default collection
// interval (1 minute) that will be applied to all scrapers if not
// overridden by the individual scraper.
func WithDefaultCollectionInterval(defaultCollectionInterval time.Duration) ScraperControllerOption {
	return func(o *scraperController) {
		o.defaultCollectionInterval = defaultCollectionInterval
	}
}

// AddMetricsScraper configures the provided scrape function to be called
// with the specified options, and at the specified collection interval
// (one minute by default).
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddMetricsScraper(scraper MetricsScraper) ScraperControllerOption {
	return func(o *scraperController) {
		// TODO: Instead of creating one wrapper per MetricsScraper, combine all with same collection interval.
		o.metricScrapers = append(o.metricScrapers, &multiMetricScraper{scrapers: []MetricsScraper{scraper}})
	}
}

// AddResourceMetricsScraper configures the provided scrape function to
// be called with the specified options, and at the specified collection
// interval (one minute by default).
//
// Observability information will be reported, and the scraped resource
// metrics will be passed to the next consumer.
func AddResourceMetricsScraper(scrape ResourceMetricsScraper) ScraperControllerOption {
	return func(o *scraperController) {
		o.metricScrapers = append(o.metricScrapers, scrape)
	}
}

type scraperController struct {
	defaultCollectionInterval time.Duration
	nextConsumer              consumer.MetricsConsumer

	metricScrapers []ResourceMetricsScraper
	done           chan struct{}
}

// NewScraperControllerReceiver creates a Receiver with the configured options, that can control multiple scrapers.
func NewScraperControllerReceiver(_ configmodels.Receiver, nextConsumer consumer.MetricsConsumer, options ...ScraperControllerOption) (component.Receiver, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	mr := &scraperController{
		defaultCollectionInterval: time.Minute,
		nextConsumer:              nextConsumer,
		done:                      make(chan struct{}),
	}

	for _, op := range options {
		op(mr)
	}

	return mr, nil
}

// Start the receiver, invoked during service start.
func (mr *scraperController) Start(ctx context.Context, _ component.Host) error {
	if err := mr.initializeScrapers(ctx); err != nil {
		return err
	}

	mr.startScraping()
	return nil
}

// Shutdown the receiver, invoked during service shutdown.
func (mr *scraperController) Shutdown(ctx context.Context) error {
	mr.stopScraping()

	var errors []error

	if err := mr.closeScrapers(ctx); err != nil {
		errors = append(errors, err)
	}

	return componenterror.CombineErrors(errors)
}

// initializeScrapers initializes all the scrapers
func (mr *scraperController) initializeScrapers(ctx context.Context) error {
	for _, scraper := range mr.metricScrapers {
		if err := scraper.Initialize(ctx); err != nil {
			return err
		}
	}

	return nil
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (mr *scraperController) startScraping() {
	// TODO: use one ticker for each set of scrapers that have the same collection interval.

	for i := 0; i < len(mr.metricScrapers); i++ {
		scraper := mr.metricScrapers[i]
		go func() {
			collectionInterval := mr.defaultCollectionInterval
			if scraper.CollectionInterval() != 0 {
				collectionInterval = scraper.CollectionInterval()
			}

			ticker := time.NewTicker(collectionInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					mr.scrapeMetricsAndReport(context.Background(), scraper)
				case <-mr.done:
					return
				}
			}
		}()
	}
}

// scrapeMetricsAndReport calls the Scrape function of the provided Scraper, records observability information,
// and passes the scraped metrics to the next component.
func (mr *scraperController) scrapeMetricsAndReport(ctx context.Context, rms ResourceMetricsScraper) {
	// TODO: Add observability metrics support
	metrics, err := rms.Scrape(ctx)
	if err != nil {
		return
	}

	mr.nextConsumer.ConsumeMetrics(ctx, resourceMetricsSliceToMetricData(metrics))
}

func resourceMetricsSliceToMetricData(resourceMetrics pdata.ResourceMetricsSlice) pdata.Metrics {
	md := pdata.NewMetrics()
	resourceMetrics.MoveAndAppendTo(md.ResourceMetrics())
	return md
}

// stopScraping stops the ticker
func (mr *scraperController) stopScraping() {
	close(mr.done)
}

// closeScrapers closes all the scrapers
func (mr *scraperController) closeScrapers(ctx context.Context) error {
	var errs []error

	for _, scraper := range mr.metricScrapers {
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

func (mms *multiMetricScraper) Initialize(ctx context.Context) error {
	for _, scraper := range mms.scrapers {
		if err := scraper.Initialize(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (mms *multiMetricScraper) CollectionInterval() time.Duration {
	return mms.scrapers[0].CollectionInterval()
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

func (mms *multiMetricScraper) Scrape(ctx context.Context) (pdata.ResourceMetricsSlice, error) {
	rms := pdata.NewResourceMetricsSlice()
	rms.Resize(1)
	rm := rms.At(0)
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	ilm := ilms.At(0)

	var errs []error
	for _, scraper := range mms.scrapers {
		metrics, err := scraper.Scrape(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		metrics.MoveAndAppendTo(ilm.Metrics())
	}
	return rms, componenterror.CombineErrors(errs)
}
