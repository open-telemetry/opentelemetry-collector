// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionscrapercontroller // import "go.opentelemetry.io/collector/extension/xextension/extensionscrapercontroller"

import (
	"context"

	"go.opentelemetry.io/collector/extension"
)

// ControllerExtension is an extension that controls when scraper-based
// receivers invoke their scrapers.
type ControllerExtension interface {
	extension.Extension

	// RegisterScraper registers a scraper with the extension. The extension
	// will call the provided ScrapeFunc when it determines a scrape should
	// occur. The returned DeregisterFunc must be called during shutdown to
	// deregister the scraper from the controller.
	//
	// Implementations may call the ScrapeFunc concurrently. DeregisterFunc
	// must not return until all in-flight calls to ScrapeFunc have completed,
	// and must guarantee that ScrapeFunc will not be called again after it
	// returns.
	RegisterScraper(context.Context, ScrapeFunc) (DeregisterFunc, error)
}

// ScrapeFunc is a function that is registered with
// ControllerExtension.RegisterScraper in order to perform a scrape.
type ScrapeFunc func(context.Context) error

// DeregisterFunc is a function returned by ControllerExtension.RegisterScraper
// and is used to deregister the scraper during shutdown.
type DeregisterFunc func(ctx context.Context) error
