// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapeErrorsAddPartial(t *testing.T) {
	err1 := errors.New("err 1")
	err2 := errors.New("err 2")
	expected := []error{
		PartialScrapeError{error: err1, Failed: 1},
		PartialScrapeError{error: err2, Failed: 10},
	}

	var errs ScrapeErrors
	errs.AddPartial(1, err1)
	errs.AddPartial(10, err2)
	assert.Equal(t, expected, errs.errs)
}

func TestScrapeErrorsAdd(t *testing.T) {
	err1 := errors.New("err a")
	err2 := errors.New("err b")
	expected := []error{err1, err2}

	var errs ScrapeErrors
	errs.Add(err1)
	errs.Add(err2)
	assert.Equal(t, expected, errs.errs)
}

func TestScrapeErrorsCombine(t *testing.T) {
	testCases := []struct {
		errs                func() ScrapeErrors
		expectedErr         string
		expectedFailedCount int
		expectNil           bool
		expectedScrape      bool
	}{
		{
			errs: func() ScrapeErrors {
				var errs ScrapeErrors
				return errs
			},
			expectNil: true,
		},
		{
			errs: func() ScrapeErrors {
				var errs ScrapeErrors
				errs.AddPartial(10, errors.New("bad scrapes"))
				errs.AddPartial(1, fmt.Errorf("err: %w", errors.New("bad scrape")))
				return errs
			},
			expectedErr:         "bad scrapes; err: bad scrape",
			expectedFailedCount: 11,
			expectedScrape:      true,
		},
		{
			errs: func() ScrapeErrors {
				var errs ScrapeErrors
				errs.Add(errors.New("bad regular"))
				errs.Add(fmt.Errorf("err: %w", errors.New("bad reg")))
				return errs
			},
			expectedErr: "bad regular; err: bad reg",
		},
		{
			errs: func() ScrapeErrors {
				var errs ScrapeErrors
				errs.AddPartial(2, errors.New("bad two scrapes"))
				errs.AddPartial(10, fmt.Errorf("%d scrapes failed: %w", 10, errors.New("bad things happened")))
				errs.Add(errors.New("bad event"))
				errs.Add(fmt.Errorf("event: %w", errors.New("something happened")))
				return errs
			},
			expectedErr:         "bad two scrapes; 10 scrapes failed: bad things happened; bad event; event: something happened",
			expectedFailedCount: 12,
			expectedScrape:      true,
		},
	}

	for _, tt := range testCases {
		scrapeErrs := tt.errs()
		if tt.expectNil {
			require.NoError(t, scrapeErrs.Combine())
			continue
		}
		require.EqualError(t, scrapeErrs.Combine(), tt.expectedErr)
		if tt.expectedScrape {
			var partialScrapeErr PartialScrapeError
			if !errors.As(scrapeErrs.Combine(), &partialScrapeErr) {
				t.Errorf("%+v.Combine() = %q. Want: PartialScrapeError", scrapeErrs, scrapeErrs.Combine())
			} else if tt.expectedFailedCount != partialScrapeErr.Failed {
				t.Errorf("%+v.Combine().Failed. Got %d Failed count. Want: %d", scrapeErrs, partialScrapeErr.Failed, tt.expectedFailedCount)
			}
		}
	}
}
