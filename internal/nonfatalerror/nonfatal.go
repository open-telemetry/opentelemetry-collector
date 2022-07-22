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

package nonfatalerror // import "go.opentelemetry.io/collector/internal/nonfatalerror"

import "errors"

// nonFatal is an error that will be always returned if its source
// receives the same inputs.
type nonFatal struct {
	err error
}

// New wraps an error to indicate that it is a non fatal error, i.e. an
// error that needs to be reported but will not make the Collector fail.
func New(err error) error {
	return nonFatal{err: err}
}

func (p nonFatal) Error() string {
	return "Non fatal error: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p nonFatal) Unwrap() error {
	return p.err
}

// IsNonFatal checks if an error was wrapped with the New function, which
// is used to indicate that a given error needs to be reported but will not make
// the Collector fail.
func IsNonFatal(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &nonFatal{})
}
