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

// ScraperSettings defines common settings for a scraper configuration.
// Specific scrapers can embed this struct and extend it with more fields if needed.
type ScraperSettings struct {
	CollectionIntervalVal time.Duration `mapstructure:"collection_interval"`
}

// CollectionInterval gets the scraper collection interval.
func (ss *ScraperSettings) CollectionInterval() time.Duration {
	return ss.CollectionIntervalVal
}

type BaseScraper interface {
	// Initialize performs any timely initialization tasks such as
	// setting up performance counters for initial collection.
	Initialize(ctx context.Context) error

	// CollectionInterval gets the scraper collection interval.
	CollectionInterval() time.Duration

	// Close should clean up any unmanaged resources such as
	// performance counter handles.
	Close(ctx context.Context) error
}

// MetricsScraper is an interface for scrapers that scrape metrics.
type MetricsScraper interface {
	BaseScraper
	Scrape(context.Context) (pdata.MetricSlice, error)
}

// ResourceMetricsScraper is an interface for scrapers that scrape resource metrics.
type ResourceMetricsScraper interface {
	BaseScraper
	Scrape(context.Context) (pdata.ResourceMetricsSlice, error)
}

var _ BaseScraper = (*baseScraper)(nil)

type baseScraper struct {
	ScraperSettings
	initialize Initialize
	close      Close
}

func (b baseScraper) Initialize(ctx context.Context) error {
	if b.initialize == nil {
		return nil
	}
	return b.initialize(ctx)
}

func (b baseScraper) Close(ctx context.Context) error {
	if b.close == nil {
		return nil
	}
	return b.close(ctx)
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
	ScrapeMetrics
}

var _ MetricsScraper = (*metricsScraper)(nil)

// NewMetricsScraper creates a Scraper that calls Scrape at the specified
// collection interval, reports observability information, and passes the
// scraped metrics to the next consumer.
func NewMetricsScraper(
	cfg ScraperSettings,
	scrape ScrapeMetrics,
	options ...ScraperOption,
) MetricsScraper {
	ms := &metricsScraper{
		baseScraper:   baseScraper{ScraperSettings: cfg},
		ScrapeMetrics: scrape,
	}

	for _, op := range options {
		op(&ms.baseScraper)
	}

	return ms
}

func (m metricsScraper) Scrape(ctx context.Context) (pdata.MetricSlice, error) {
	return m.ScrapeMetrics(ctx)
}

type resourceMetricsScraper struct {
	baseScraper
	ScrapeResourceMetrics
}

var _ ResourceMetricsScraper = (*resourceMetricsScraper)(nil)

// NewResourceMetricsScraper creates a Scraper that calls Scrape at the
// specified collection interval, reports observability information, and
// passes the scraped resource metrics to the next consumer.
func NewResourceMetricsScraper(
	cfg ScraperSettings,
	scrape ScrapeResourceMetrics,
	options ...ScraperOption,
) ResourceMetricsScraper {
	rms := &resourceMetricsScraper{
		baseScraper:           baseScraper{ScraperSettings: cfg},
		ScrapeResourceMetrics: scrape,
	}

	for _, op := range options {
		op(&rms.baseScraper)
	}

	return rms
}

func (r resourceMetricsScraper) Scrape(ctx context.Context) (pdata.ResourceMetricsSlice, error) {
	return r.ScrapeResourceMetrics(ctx)
}
