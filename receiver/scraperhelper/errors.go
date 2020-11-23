// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scraperhelper

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

// CombineScrapeErrors converts a list of errors into one error.
func CombineScrapeErrors(errs []error) error {
	partialScrapeErr := false
	for _, err := range errs {
		var partialError consumererror.PartialScrapeError
		if errors.As(err, &partialError) {
			partialScrapeErr = true
			break
		}
	}

	if !partialScrapeErr {
		return componenterror.CombineErrors(errs)
	}

	errMsgs := make([]string, 0, len(errs))
	failedScrapeCount := 0
	for _, err := range errs {
		if partialError, isPartial := err.(consumererror.PartialScrapeError); isPartial {
			failedScrapeCount += partialError.Failed
		}

		errMsgs = append(errMsgs, err.Error())
	}

	var err error
	if len(errs) == 1 {
		err = errs[0]
	} else {
		err = fmt.Errorf("[%s]", strings.Join(errMsgs, "; "))
	}

	return consumererror.NewPartialScrapeError(err, failedScrapeCount)
}
