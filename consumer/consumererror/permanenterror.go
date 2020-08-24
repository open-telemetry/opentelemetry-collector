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

// Package consumererror provides wrappers to easily classify errors. This allows
// appropriate action by error handlers without the need to know each individual
// error type/instance.
package consumererror

// permanent is an error that will be always returned if its source
// receives the same inputs.
type permanent struct {
	err error
}

// Permanent wraps an error to indicate that it is a permanent error, i.e.: an
// error that will be always returned if its source receives the same inputs.
func Permanent(err error) error {
	return permanent{err: err}
}

func (p permanent) Error() string {
	return "Permanent error: " + p.err.Error()
}

// IsPermanent checks if an error was wrapped with the Permanent function, that
// is used to indicate that a given error will always be returned in the case
// that its sources receives the same input.
func IsPermanent(err error) bool {
	if err != nil {
		_, isPermanent := err.(permanent)
		return isPermanent
	}
	return false
}
