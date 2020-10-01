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

	"go.opentelemetry.io/collector/consumer/pdata"
)

// Scraper provides a function to scrape metrics.
type Scrape func(context.Context) (pdata.Metrics, error)

// Initialize performs any timely initialization tasks such as
// setting up performance counters for initial collection.
type Initialize func(ctx context.Context) error

// Close should clean up any unmanaged resources such as
// performance counter handles.
type Close func(ctx context.Context) error

// ScraperOption apply changes to internal options.
type ScraperOption func(*scraper)

// ScraperConfig is the configuration of a scraper. Specific scrapers must implement this
// interface and will typically embed ScraperSettings struct or a struct that extends it.
type ScraperConfig interface {
	CollectionInterval() time.Duration
	SetCollectionInterval(collectionInterval time.Duration)
}

// ScraperSettings defines common settings for a scraper configuration.
// Specific scrapers can embed this struct and extend it with more fields if needed.
type ScraperSettings struct {
	CollectionIntervalVal time.Duration `mapstructure:"collection_interval"`
}

// CollectionInterval gets the scraper collection interval.
func (ss *ScraperSettings) CollectionInterval() time.Duration {
	return ss.CollectionIntervalVal
}

// SetCollectionInterval sets the scraper collection interval.
func (ss *ScraperSettings) SetCollectionInterval(collectionInterval time.Duration) {
	ss.CollectionIntervalVal = collectionInterval
}

type scraper struct {
	cfg        ScraperConfig
	scrape     Scrape
	initialize Initialize
	close      Close
}

// NewScraper creates a Scraper that calls Scrape at the specified collection
// interval, reports observability information, and passes the scraped metrics
// to the next consumer.
func newScraper(
	cfg ScraperConfig,
	scrape Scrape,
	options ...ScraperOption,
) *scraper {
	bs := &scraper{cfg: cfg, scrape: scrape}

	for _, op := range options {
		op(bs)
	}

	return bs
}

// WithInitialize sets the function that will be called on startup.
func WithInitialize(initialize Initialize) ScraperOption {
	return func(o *scraper) {
		o.initialize = initialize
	}
}

// WithClose sets the function that will be called on shutdown.
func WithClose(close Close) ScraperOption {
	return func(o *scraper) {
		o.close = close
	}
}
