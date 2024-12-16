// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

var (
	AddScraper                   = AddMetricsScraper
	NewScraperControllerReceiver = NewMetricsScraperControllerReceiver
	WithTickerChannel            = WithMetricsTickerChannel
)

type ScraperControllerOption = MetricsScraperControllerOption
