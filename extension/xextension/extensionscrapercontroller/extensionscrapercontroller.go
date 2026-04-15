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
	// will call scrapeFunc when it determines a scrape should occur.
	// The returned RegistrationHandle must be used to deregister the scraper
	// during shutdown.
	//
	// Implementations may call scrapeFunc concurrently. After Deregister on
	// the returned handle returns, implementations must not start new
	// invocations of scrapeFunc; already in-flight invocations may continue
	// to run.
	RegisterScraper(ctx context.Context, scrapeFunc func(context.Context) error) (RegistrationHandle, error)
}

// RegistrationHandle is returned by ControllerExtension.RegisterScraper and
// is used to deregister the scraper during shutdown.
type RegistrationHandle interface {
	// Deregister removes the scraper registration from the extension.
	// After Deregister returns, the extension must not start any new
	// invocations of the associated scrapeFunc. Deregister need not wait
	// for in-flight invocations to complete.
	Deregister(ctx context.Context) error
}

// DeregisterFunc is a function that implements RegistrationHandle.
// A nil DeregisterFunc is valid and returns nil on Deregister.
type DeregisterFunc func(ctx context.Context) error

var _ RegistrationHandle = DeregisterFunc(nil)

// Deregister calls the underlying function. If the receiver is nil, it returns nil.
func (f DeregisterFunc) Deregister(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}
