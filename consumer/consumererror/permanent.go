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

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"strconv"
)

// noDataCount indicates an unknown number of records.
const noDataCount = -1

// Permanent is an error that will always be returned if its source
// receives the same inputs.
type Permanent struct {
	error
	rejected int
}

// Deprecated [v0.76.0] Use consumererror.NewPermanentWithCount instead.
func NewPermanent(err error) error {
	return Permanent{error: err, rejected: noDataCount}
}

// NewPermanentWithCount wraps an error to indicate that it is a permanent error, i.e. an
// error that will be always returned if its source receives the same inputs.
// The error should also include the number of rejected records.
func NewPermanentWithCount(err error, rejected int) error {
	return Permanent{error: err, rejected: rejected}
}

func (p Permanent) Error() string {
	rejected := "unknown"

	if p.rejected != noDataCount {
		rejected = strconv.Itoa(p.rejected)
	}

	return "Permanent error (" + rejected + " rejected): " + p.error.Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (p Permanent) Unwrap() error {
	return p.error
}

func (p Permanent) Rejected() int {
	return p.rejected
}

// IsPermanent checks if an error was wrapped with the NewPermanent function, which
// is used to indicate that a given error will always be returned in the case
// that its sources receives the same input.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &Permanent{})
}
