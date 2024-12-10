// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper // import "go.opentelemetry.io/collector/scraper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
)

var errNilFunc = errors.New("nil scrape func")

// ScrapeFunc scrapes data.
type ScrapeFunc[T any] func(context.Context) (T, error)

// Option apply changes to internal options.
type Option interface {
	apply(*baseScraper)
}

type scraperOptionFunc func(*baseScraper)

func (of scraperOptionFunc) apply(e *baseScraper) {
	of(e)
}

// WithStart sets the function that will be called on startup.
func WithStart(start component.StartFunc) Option {
	return scraperOptionFunc(func(o *baseScraper) {
		o.StartFunc = start
	})
}

// WithShutdown sets the function that will be called on shutdown.
func WithShutdown(shutdown component.ShutdownFunc) Option {
	return scraperOptionFunc(func(o *baseScraper) {
		o.ShutdownFunc = shutdown
	})
}

type baseScraper struct {
	component.StartFunc
	component.ShutdownFunc
}

// newBaseScraper returns the internal settings starting from the default and applying all options.
func newBaseScraper(options []Option) baseScraper {
	// Start from the default options:
	bs := baseScraper{}

	for _, op := range options {
		op.apply(&bs)
	}

	return bs
}
