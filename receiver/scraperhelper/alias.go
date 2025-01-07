// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [v0.117.0] import "go.opentelemetry.io/collector/scraper/scraperhelper"
package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	nsch "go.opentelemetry.io/collector/scraper/scraperhelper"
)

// Deprecated: [v0.117.0] import "go.opentelemetry.io/collector/scraper/scraperhelper"
type ControllerConfig = nsch.ControllerConfig

// Deprecated: [v0.117.0] import "go.opentelemetry.io/collector/scraper/scraperhelper"
var NewDefaultControllerConfig = nsch.NewDefaultControllerConfig

// Deprecated: [v0.117.0] import "go.opentelemetry.io/collector/scraper/scraperhelper"
type ScraperControllerOption = nsch.MetricsScraperControllerOption

// Deprecated: [v0.117.0] import "go.opentelemetry.io/collector/scraper/scraperhelper"
var AddScraper = nsch.AddMetricsScraper

// Deprecated: [v0.117.0] import "go.opentelemetry.io/collector/scraper/scraperhelper"
var WithTickerChannel = nsch.WithMetricsTickerChannel

// Deprecated: [v0.117.0] import "go.opentelemetry.io/collector/scraper/scraperhelper"
var NewScraperControllerReceiver = nsch.NewMetricsScraperControllerReceiver
