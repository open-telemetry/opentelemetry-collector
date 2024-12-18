// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapererror // import "go.opentelemetry.io/collector/scraper/scrapererror"

import "errors"

// PartialScrapeError is an error to represent
// that a subset of data were failed to be scraped.
type PartialScrapeError struct {
	error
	Failed int
}

// NewPartialScrapeError creates PartialScrapeError for failed data.
// Use this error type only when a subset of data was failed to be scraped.
func NewPartialScrapeError(err error, failed int) PartialScrapeError {
	return PartialScrapeError{
		error:  err,
		Failed: failed,
	}
}

// IsPartialScrapeError checks if an error was wrapped with PartialScrapeError.
func IsPartialScrapeError(err error) bool {
	var partialScrapeErr PartialScrapeError
	return errors.As(err, &partialScrapeErr)
}
