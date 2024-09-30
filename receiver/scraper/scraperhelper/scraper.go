// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraper/scraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var errNilFunc = errors.New("nil scrape func")

// ScrapeFunc scrapes metrics.
type ScrapeFunc func(context.Context) (pmetric.Metrics, error)

func (sf ScrapeFunc) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	return sf(ctx)
}

// Scraper is the base interface for scrapers.
type Scraper interface {
	component.Component

	// ID returns the scraper id.
	ID() component.ID
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

var _ Scraper = (*baseScraper)(nil)

type baseScraper struct {
	component.StartFunc
	component.ShutdownFunc
	ScrapeFunc
	id component.ID
}

func (b *baseScraper) ID() component.ID {
	return b.id
}

// NewScraper creates a Scraper that calls Scrape at the specified collection interval,
// reports observability information, and passes the scraped metrics to the next consumer.
func NewScraper(t component.Type, scrape ScrapeFunc, options ...ScraperOption) (Scraper, error) {
	if scrape == nil {
		return nil, errNilFunc
	}
	bs := &baseScraper{
		ScrapeFunc: scrape,
		id:         component.NewID(t),
	}
	for _, op := range options {
		op.apply(bs)
	}

	return bs, nil
}
