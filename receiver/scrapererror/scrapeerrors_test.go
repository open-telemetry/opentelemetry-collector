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

package scrapererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
				errs.AddPartial(1, fmt.Errorf("err: %s", errors.New("bad scrape")))
				return errs
			},
			expectedErr:         "[bad scrapes; err: bad scrape]",
			expectedFailedCount: 11,
			expectedScrape:      true,
		},
		{
			errs: func() ScrapeErrors {
				var errs ScrapeErrors
				errs.Add(errors.New("bad regular"))
				errs.Add(fmt.Errorf("err: %s", errors.New("bad reg")))
				return errs
			},
			expectedErr: "[bad regular; err: bad reg]",
		},
		{
			errs: func() ScrapeErrors {
				var errs ScrapeErrors
				errs.AddPartial(2, errors.New("bad two scrapes"))
				errs.AddPartial(10, fmt.Errorf("%d scrapes failed: %s", 10, errors.New("bad things happened")))
				errs.Add(errors.New("bad event"))
				errs.Add(fmt.Errorf("event: %s", errors.New("something happened")))
				return errs
			},
			expectedErr:         "[bad two scrapes; 10 scrapes failed: bad things happened; bad event; event: something happened]",
			expectedFailedCount: 12,
			expectedScrape:      true,
		},
	}

	for _, tc := range testCases {
		scrapeErrs := tc.errs()
		if (scrapeErrs.Combine() == nil) != tc.expectNil {
			t.Errorf("%+v.Combine() == nil? Got: %t. Want: %t", scrapeErrs, scrapeErrs.Combine() == nil, tc.expectNil)
		}
		if scrapeErrs.Combine() != nil && tc.expectedErr != scrapeErrs.Combine().Error() {
			t.Errorf("%+v.Combine() = %q. Want: %q", scrapeErrs, scrapeErrs.Combine(), tc.expectedErr)
		}
		if tc.expectedScrape {
			partialScrapeErr, ok := scrapeErrs.Combine().(PartialScrapeError)
			if !ok {
				t.Errorf("%+v.Combine() = %q. Want: PartialScrapeError", scrapeErrs, scrapeErrs.Combine())
			} else if tc.expectedFailedCount != partialScrapeErr.Failed {
				t.Errorf("%+v.Combine().Failed. Got %d Failed count. Want: %d", scrapeErrs, partialScrapeErr.Failed, tc.expectedFailedCount)
			}
		}
	}
}
