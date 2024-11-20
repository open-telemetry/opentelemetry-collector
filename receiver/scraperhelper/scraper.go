// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper"
)

var errNilFunc = errors.New("nil scrape func")

// ScrapeFunc scrapes metrics.
type ScrapeFunc func(context.Context) (pmetric.Metrics, error)

func (sf ScrapeFunc) ScrapeMetrics(ctx context.Context) (pmetric.Metrics, error) {
	return sf(ctx)
}

// Deprecated: [v0.115.0] use ScrapeMetrics.
func (sf ScrapeFunc) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	return sf.ScrapeMetrics(ctx)
}

// Deprecated: [v0.115.0] use scraper.Metrics.
type Scraper interface {
	component.Component
	Scrape(context.Context) (pmetric.Metrics, error)
}

// ScraperOption apply changes to internal options.
type ScraperOption interface {
	apply(*baseScraper)
}

type scraperOptionFunc func(*baseScraper)

func (of scraperOptionFunc) apply(e *baseScraper) {
	of(e)
}

// WithStart sets the function that will be called on startup.
func WithStart(start component.StartFunc) ScraperOption {
	return scraperOptionFunc(func(o *baseScraper) {
		o.StartFunc = start
	})
}

// WithShutdown sets the function that will be called on shutdown.
func WithShutdown(shutdown component.ShutdownFunc) ScraperOption {
	return scraperOptionFunc(func(o *baseScraper) {
		o.ShutdownFunc = shutdown
	})
}

type baseScraper struct {
	component.StartFunc
	component.ShutdownFunc
	ScrapeFunc
}

// NewScraper creates a scraper.Metrics that calls ScrapeMetrics at the specified collection interval,
// reports observability information, and passes the scraped metrics to the next consumer.
func NewScraper(scrape ScrapeFunc, options ...ScraperOption) (scraper.Metrics, error) {
	return newBaseScraper(scrape, options...)
}

// TODO: Remove this and embed into NewScraper when  NewScraperWithoutType is removed.
func newBaseScraper(scrape ScrapeFunc, options ...ScraperOption) (*baseScraper, error) {
	if scrape == nil {
		return nil, errNilFunc
	}
	bs := &baseScraper{
		ScrapeFunc: scrape,
	}
	for _, op := range options {
		op.apply(bs)
	}

	return bs, nil
}

// Deprecated: [v0.115.0] use NewScraper.
func NewScraperWithoutType(scrape ScrapeFunc, options ...ScraperOption) (Scraper, error) {
	return newBaseScraper(scrape, options...)
}
