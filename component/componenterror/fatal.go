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

// fatal is an error that cannot be recovered from and will cause early
// termination of the collector
type fatal struct {
	err error
}

// NewFatal wraps an error to indicate that it is a fatal error
func NewFatal(err error) error {
	return fatal{err: err}
}

func (p fatal) Error() string {
	return "Fatal error: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p fatal) Unwrap() error {
	return p.err
}

// IsFatal checks if an error was wrapped with the NewFatal function
func IsFatal(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &fatal{})
}
