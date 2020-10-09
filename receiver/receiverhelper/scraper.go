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

// Scrape metrics.
type ScrapeMetrics func(context.Context) (pdata.MetricSlice, error)

// Scrape resource metrics.
type ScrapeResourceMetrics func(context.Context) (pdata.ResourceMetricsSlice, error)

// Initialize performs any timely initialization tasks such as
// setting up performance counters for initial collection.
type Initialize func(ctx context.Context) error

// Close should clean up any unmanaged resources such as
// performance counter handles.
type Close func(ctx context.Context) error

// ScraperOption apply changes to internal options.
type ScraperOption func(*baseScraper)

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

type baseScraper struct {
	cfg        ScraperConfig
	initialize Initialize
	close      Close
}

// WithInitialize sets the function that will be called on startup.
func WithInitialize(initialize Initialize) ScraperOption {
	return func(o *baseScraper) {
		o.initialize = initialize
	}
}

// WithClose sets the function that will be called on shutdown.
func WithClose(close Close) ScraperOption {
	return func(o *baseScraper) {
		o.close = close
	}
}

type metricsScraper struct {
	baseScraper
	scrape ScrapeMetrics
}

// newMetricsScraper creates a Scraper that calls Scrape at the specified
// collection interval, reports observability information, and passes the
// scraped metrics to the next consumer.
func newMetricsScraper(
	cfg ScraperConfig,
	scrape ScrapeMetrics,
	options ...ScraperOption,
) *metricsScraper {
	ms := &metricsScraper{
		baseScraper: baseScraper{cfg: cfg},
		scrape:      scrape,
	}

	for _, op := range options {
		op(&ms.baseScraper)
	}

	return ms
}

type resourceMetricsScraper struct {
	baseScraper
	scrape ScrapeResourceMetrics
}

// newResourceMetricsScraper creates a Scraper that calls Scrape at the
// specified collection interval, reports observability information, and
// passes the scraped resource metrics to the next consumer.
func newResourceMetricsScraper(
	cfg ScraperConfig,
	scrape ScrapeResourceMetrics,
	options ...ScraperOption,
) *resourceMetricsScraper {
	rms := &resourceMetricsScraper{
		baseScraper: baseScraper{cfg: cfg},
		scrape:      scrape,
	}

	for _, op := range options {
		op(&rms.baseScraper)
	}

	return rms
}
