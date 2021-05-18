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

package models

import (
	"fmt"
)

// ErrIncompatibleType details a type conversion error during translation.
type ErrIncompatibleType struct {
	given interface{}
	expected interface{}
}

func (i *ErrIncompatibleType) Error() string {
	return fmt.Sprintf("model type %T is expected but given %T ", i.expected, i.given)
}

// NewErrIncompatibleType returns ErrIncompatibleType instance
func NewErrIncompatibleType(expected, given interface{}) *ErrIncompatibleType {
	return &ErrIncompatibleType{
		given:    given,
		expected: expected,
	}
}