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

package translator

import (
	"fmt"
)

// errIncompatibleType details a type conversion error during translation.
type errIncompatibleType struct {
	given    interface{}
	expected interface{}
}

func (i *errIncompatibleType) Error() string {
	return fmt.Sprintf("expected model type %T but given %T", i.expected, i.given)
}

// NewErrIncompatibleType returns errIncompatibleType instance
func NewErrIncompatibleType(expected, given interface{}) *errIncompatibleType {
	return &errIncompatibleType{
		given:    given,
		expected: expected,
	}
}
