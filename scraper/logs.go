// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper // import "go.opentelemetry.io/collector/scraper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
)

// Logs is the base interface for logs scrapers.
type Logs interface {
	component.Component

	// ScrapeLogs is the base interface to indicate that how should logs be scraped.
	ScrapeLogs(context.Context) (plog.Logs, error)
}

// ScrapeLogsFunc is a helper function that is similar to Logs.ScrapeLogs.
type ScrapeLogsFunc ScrapeFunc[plog.Logs]

func (sf ScrapeLogsFunc) ScrapeLogs(ctx context.Context) (plog.Logs, error) {
	return sf(ctx)
}

type logs struct {
	baseScraper
	ScrapeLogsFunc
}

// NewLogs creates a new Logs scraper.
func NewLogs(scrape ScrapeLogsFunc, options ...Option) (Logs, error) {
	if scrape == nil {
		return nil, errNilFunc
	}
	bs := &logs{
		baseScraper:    newBaseScraper(options),
		ScrapeLogsFunc: scrape,
	}

	return bs, nil
}
