// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapererror // import "go.opentelemetry.io/collector/scraper/scrapererror"

import (
	"go.uber.org/multierr"
)

// ScrapeErrors contains multiple PartialScrapeErrors and can also contain generic errors.
type ScrapeErrors struct {
	errs              []error
	failedScrapeCount int
}

// AddPartial adds a PartialScrapeError with the provided failed count and error.
func (s *ScrapeErrors) AddPartial(failed int, err error) {
	s.errs = append(s.errs, NewPartialScrapeError(err, failed))
	s.failedScrapeCount += failed
}

// Add adds a regular error.
func (s *ScrapeErrors) Add(err error) {
	s.errs = append(s.errs, err)
}

// Combine converts a slice of errors into one error.
// It will return a PartialScrapeError if at least one error in the slice is a PartialScrapeError.
func (s *ScrapeErrors) Combine() error {
	partialScrapeErr := false
	for _, err := range s.errs {
		if IsPartialScrapeError(err) {
			partialScrapeErr = true
		}
	}

	combined := multierr.Combine(s.errs...)
	if !partialScrapeErr {
		return combined
	}

	return NewPartialScrapeError(combined, s.failedScrapeCount)
}
