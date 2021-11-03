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
	"strings"
)

type combined []error

func (c combined) Error() string {
	if len(c) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(c[0].Error())
	for _, err := range c[1:] {
		sb.WriteString("; ")
		sb.WriteString(err.Error())
	}
	sb.WriteString("]")
	return sb.String()
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
// Deprecated: Use alternative modules like "go.uber.org/multierr"
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
