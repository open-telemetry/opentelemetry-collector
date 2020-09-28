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

// Scraper provides a function to scrape metrics.
type Scraper interface {
	Scrape(context.Context) (pdata.Metrics, error)
}

// Scrape specifies the function that will be invoked every collection interval when
// the receiver is configured as a Scraper.
type Scrape func(context.Context) (pdata.Metrics, error)

// ScraperConfig provides functions to get and set the collection interval.
type ScraperConfig interface {
	configmodels.Receiver

	CollectionInterval() time.Duration
	SetCollectionInterval(time.Duration)
}

// ScraperSettings defines common settings for scraper configuration.
// Specific scrapers can embed this struct and extend it with more fields if needed.
type ScraperSettings struct {
	configmodels.ReceiverSettings `mapstructure:",squash"`
	CollectionIntervalVal         time.Duration `mapstructure:"collection_interval"`
}

// CollectionInterval gets the scraper collection interval.
func (ss *ScraperSettings) CollectionInterval() time.Duration {
	return ss.CollectionIntervalVal
}

// SetCollectionInterval sets the scraper collection interval.
func (ss *ScraperSettings) SetCollectionInterval(collectionInterval time.Duration) {
	ss.CollectionIntervalVal = collectionInterval
}

// ScraperOption apply changes to internal scraper options.
type ScraperOption func(*baseScraper)

type baseScraper struct {
	baseReceiver
	collectionInterval time.Duration
	scraper            Scraper
	nextConsumer       consumer.MetricsConsumer
	done               chan struct{}
}

// NewScraper creates a MetricsReceiver that calls Scrape at the specified collection
// interval, reports observability information, and passes the scraped metrics to the
// next consumer.
func NewScraper(
	config ScraperConfig,
	scraper Scraper,
	nextConsumer consumer.MetricsConsumer,
	options ...Option,
) (component.Receiver, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	bs := &baseScraper{
		baseReceiver:       newBaseReceiver(config.Name(), options...),
		collectionInterval: config.CollectionInterval(),
		scraper:            scraper,
		nextConsumer:       nextConsumer,
		done:               make(chan struct{}),
	}

	// wrap the start function with a call to start scraping
	start := bs.start
	bs.start = func(ctx context.Context, host component.Host) error {
		if start != nil {
			if err := start(ctx, host); err != nil {
				return err
			}
		}

		bs.startScraping()
		return nil
	}

	// wrap the shutdown function with a call to stop scraping
	shutdown := bs.shutdown
	bs.shutdown = func(ctx context.Context) error {
		bs.stopScraping()

		if shutdown != nil {
			shutdown(ctx)
		}
		return nil
	}

	return bs, nil
}

// startScraping initiates a ticker that calls Scrape based on the configured
// collection interval.
func (bs *baseScraper) startScraping() {
	go func() {
		ticker := time.NewTicker(bs.collectionInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				bs.scrapeAndReport(context.Background())
			case <-bs.done:
				return
			}
		}
	}()
}

// scrapeAndReport calls the Scrape function of the provided Scraper, records
// observability information, and passes the scraped metrics to the next component.
// TODO: Add observability metrics support
func (bs *baseScraper) scrapeAndReport(ctx context.Context) {
	metrics, err := bs.scraper.Scrape(ctx)
	if err != nil {
		return
	}

	bs.nextConsumer.ConsumeMetrics(ctx, metrics)
}

// stopScraping stops the ticker
func (bs *baseScraper) stopScraping() {
	close(bs.done)
}
