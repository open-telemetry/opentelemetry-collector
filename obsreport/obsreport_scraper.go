// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import "go.opentelemetry.io/collector/receiver/scraperhelper"

// Scraper is a helper to add observability to a scraper.
//
// Deprecated: [0.86.0] Use scraperhelper.ObsReport instead.
type Scraper = scraperhelper.ObsReport

// ScraperSettings are settings for creating a Scraper.
//
// Deprecated: [0.86.0] Use scraperhelper.ObsReportSettings instead.
type ScraperSettings = scraperhelper.ObsReportSettings

// NewScraper creates a new Scraper.
//
// Deprecated: [0.86.0] Use scraperhelper.NewObsReport instead.
func NewScraper(cfg ScraperSettings) (*Scraper, error) {
	return scraperhelper.NewObsReport(cfg)
}
