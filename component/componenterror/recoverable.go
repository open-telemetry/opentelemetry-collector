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

package componenterror // import "go.opentelemetry.io/collector/component/componenterror"

import "errors"

// recoverable is an error that indicates retries might eventually be successful
type recoverable struct {
	err error
}

// NewRecoverable wraps an error to indicate that it is a recoverable error, i.e. an
// error that can be retried, potentially with a successful outcome
func NewRecoverable(err error) error {
	return recoverable{err: err}
}

func (p recoverable) Error() string {
	return "Recoverable error: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p recoverable) Unwrap() error {
	return p.err
}

// IsRecoverable checks if an error was wrapped with the NewRecoverable function
func IsRecoverable(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &recoverable{})
}
