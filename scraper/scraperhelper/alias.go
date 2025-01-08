// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

// Deprecated: [v0.118.0] use AddMetricsScraper.
var AddScraper = AddMetricsScraper

// Deprecated: [v0.118.0] use NewMetricsScraperControllerReceiver.
var NewScraperControllerReceiver = NewMetricsScraperControllerReceiver
