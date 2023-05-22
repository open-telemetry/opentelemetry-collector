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

package errs

import "fmt"

var _ error = (*RequestError)(nil)

// RequestError represents an error returned during HTTP client operations
type RequestError struct {
	statusCode int
	message    string
	args       []any
}

// NewRequestError creates a new HTTP Client Request error with the given parameters
func NewRequestError(statusCode int, message string, args ...any) *RequestError {
	return &RequestError{
		message:    message,
		args:       args,
		statusCode: statusCode,
	}
}

func (r *RequestError) Error() string {
	return r.Message()
}

func (r *RequestError) Message() string {
	return fmt.Sprintf(r.message, r.args...)
}

func (r *RequestError) StatusCode() int {
	return r.statusCode
}
