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

// NO_DATA_COUNT indicates an unknown quantity of telemetry items.
const NO_DATA_COUNT = -1

// Permanent is an error that will always be returned if its source
// receives the same inputs.
type Permanent struct {
	error
	successful int
	failed     int
}

// Deprecated [v0.75.0] Use consumererror.NewPermanentWithCounts instead.
func NewPermanent(err error) error {
	return Permanent{error: err, successful: NO_DATA_COUNT, failed: NO_DATA_COUNT}
}

// NewPermanentWithCounts wraps an error to indicate that it is a permanent error, i.e. an
// error that will be always returned if its source receives the same inputs.
// The error should also include the number of successful and failed telemetry items
// if the counts are known. Unknown counts should be set to consumererror.NO_DATA_COUNT.
func NewPermanentWithCounts(err error, successful, failed int) error {
	return Permanent{error: err, successful: successful, failed: failed}
}

func (p Permanent) Error() string {
	successful := "unknown"
	failed := "unknown"

	if p.successful != NO_DATA_COUNT {
		successful = strconv.Itoa(p.successful)
	}
	if p.failed != NO_DATA_COUNT {
		failed = strconv.Itoa(p.failed)
	}

	return "Permanent error (" + successful + " successful, " + failed + " failed): " + p.error.Error()
}

// Unwrap returns the wrapped error for use by `errors.Is` and `errors.As`.
func (p Permanent) Unwrap() error {
	return p.error
}

func (p Permanent) Successful() int {
	return p.successful
}

func (p Permanent) Failed() int {
	return p.failed
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
