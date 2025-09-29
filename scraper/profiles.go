// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper // import "go.opentelemetry.io/collector/scraper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// Profiles is the base interface for profiles scrapers.
type Profiles interface {
	component.Component

	// ScrapeProfiles is the base interface to indicate that how should profiles be scraped.
	ScrapeProfiles(context.Context) (pprofile.Profiles, error)
}

// ScrapeProfilesFunc is a helper function.
type ScrapeProfilesFunc ScrapeFunc[pprofile.Profiles]

func (sf ScrapeProfilesFunc) ScrapeProfiles(ctx context.Context) (pprofile.Profiles, error) {
	return sf(ctx)
}

type profiles struct {
	baseScraper
	ScrapeProfilesFunc
}

// NewProfiles creates a new Profiles scraper.
func NewProfiles(scrape ScrapeProfilesFunc, options ...Option) (Profiles, error) {
	if scrape == nil {
		return nil, errNilFunc
	}
	bs := &profiles{
		baseScraper:        newBaseScraper(options),
		ScrapeProfilesFunc: scrape,
	}
	return bs, nil
}
