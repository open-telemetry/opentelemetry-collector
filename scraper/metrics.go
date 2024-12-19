// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper // import "go.opentelemetry.io/collector/scraper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Metrics is the base interface for metrics scrapers.
type Metrics interface {
	component.Component

	ScrapeMetrics(context.Context) (pmetric.Metrics, error)
}

// ScrapeMetricsFunc is a helper function that is similar to Metrics.ScrapeMetrics.
type ScrapeMetricsFunc ScrapeFunc[pmetric.Metrics]

func (sf ScrapeMetricsFunc) ScrapeMetrics(ctx context.Context) (pmetric.Metrics, error) {
	return sf(ctx)
}

type metrics struct {
	baseScraper
	ScrapeMetricsFunc
}

// NewMetrics creates a new Metrics scraper.
func NewMetrics(scrape ScrapeMetricsFunc, options ...Option) (Metrics, error) {
	if scrape == nil {
		return nil, errNilFunc
	}
	bs := &metrics{
		baseScraper:       newBaseScraper(options),
		ScrapeMetricsFunc: scrape,
	}

	return bs, nil
}
