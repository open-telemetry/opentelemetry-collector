// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var errNilFunc = errors.New("nil scrape func")

// Deprecated: [v0.115.0] use scraper.ScrapeMetricsFunc.
type ScrapeFunc func(context.Context) (pmetric.Metrics, error)

func (sf ScrapeFunc) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	return sf(ctx)
}

// Deprecated: [v0.115.0] use scraper.Metrics.
type Scraper interface {
	component.Component
	Scrape(context.Context) (pmetric.Metrics, error)
}

// Deprecated: [v0.115.0] use scraper.Option.
type ScraperOption interface {
	apply(*baseScraper)
}

type scraperOptionFunc func(*baseScraper)

func (of scraperOptionFunc) apply(e *baseScraper) {
	of(e)
}

// Deprecated: [v0.115.0] use scraper.WithStart.
func WithStart(start component.StartFunc) ScraperOption {
	return scraperOptionFunc(func(o *baseScraper) {
		o.StartFunc = start
	})
}

// Deprecated: [v0.115.0] use scraper.WithShutdown.
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

// Deprecated: [v0.115.0] use scraper.NewMetrics.
func NewScraperWithoutType(scrape ScrapeFunc, options ...ScraperOption) (Scraper, error) {
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
