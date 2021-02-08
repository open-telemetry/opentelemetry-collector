// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumererror

import (
	"fmt"
)

// ScrapeErrors contains multiple PartialScrapeErrors and can also contain generic errors.
type ScrapeErrors struct {
	errs              []error
	failedScrapeCount int
}

// Add adds a PartialScrapeError with the provided failed count and error.
func (s *ScrapeErrors) Add(failed int, err error) {
	s.errs = append(s.errs, NewPartialScrapeError(err, failed))
	s.failedScrapeCount += failed
}

// Addf adds a PartialScrapeError with the provided failed count and arguments to format an error.
func (s *ScrapeErrors) Addf(failed int, format string, a ...interface{}) {
	s.errs = append(s.errs, NewPartialScrapeError(fmt.Errorf(format, a...), failed))
	s.failedScrapeCount += failed
}

// Add adds a regular generic error.
func (s *ScrapeErrors) AddRegular(err error) {
	s.errs = append(s.errs, err)
}

// Add adds a regular generic error from the provided format specifier.
func (s *ScrapeErrors) AddRegularf(format string, a ...interface{}) {
	s.errs = append(s.errs, fmt.Errorf(format, a...))
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

	if !partialScrapeErr {
		return CombineErrors(s.errs)
	}

	return NewPartialScrapeError(CombineErrors(s.errs), s.failedScrapeCount)
}
