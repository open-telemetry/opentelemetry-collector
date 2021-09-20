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

package consumererror

import (
	"errors"
	"strings"
)

type combined []error

var _ error = (*combined)(nil)

func (c combined) Error() string {
	var sb strings.Builder
	length := len(c)
	for i, err := range c {
		sb.WriteString(err.Error())
		if i != length-1 {
			sb.WriteString("; ")
		}
	}
	return "[" + sb.String() + "]"
}

func (c combined) Is(target error) bool {
	for _, err := range c {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func (c combined) As(target interface{}) bool {
	for _, err := range c {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// Combine converts a list of errors into one error.
//
// If any of the errors in errs are Permanent then the returned
// error will also be Permanent.
func Combine(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}
	result := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			result = append(result, err)
		}
	}
	return combined(result)
}
