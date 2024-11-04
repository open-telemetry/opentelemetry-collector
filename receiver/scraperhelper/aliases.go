// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package scraperhelper provides aliases only for go.opentelemetry.io/receiver/scraper/scraperhelper
// It will be deleted in a future version.
package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import "go.opentelemetry.io/collector/receiver/scraper/scraperhelper"

type (
	ControllerConfig        = scraperhelper.ControllerConfig
	ScrapeFunc              = scraperhelper.ScrapeFunc
	Scraper                 = scraperhelper.Scraper
	ScraperOption           = scraperhelper.ScraperOption
	ScraperControllerOption = scraperhelper.ScraperControllerOption
)

var (
	NewDefaultControllerConfig   = scraperhelper.NewDefaultControllerConfig
	Validate                     = (*ControllerConfig).Validate
	Scrape                       = (ScrapeFunc).Scrape
	WithStart                    = scraperhelper.WithStart
	WithShutdown                 = scraperhelper.WithShutdown
	NewScraper                   = scraperhelper.NewScraper
	AddScraper                   = scraperhelper.AddScraper
	WithTickerChannel            = scraperhelper.WithTickerChannel
	NewScraperControllerReceiver = scraperhelper.NewScraperControllerReceiver
)
