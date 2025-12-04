// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metadata // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadata"

const (
	// scraperKey used to identify scrapers in metrics and traces.
	ScraperKey  = "scraper"
	SpanNameSep = "/"
	// receiverKey used to identify receivers in metrics and traces.
	ReceiverKey = "receiver"
	// FormatKey used to identify the format of the data received.
	FormatKey = "format"
)
