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

package data // import "go.opentelemetry.io/collector/model/internal/data"

import (
	"github.com/gogo/protobuf/types"
)

// OptionalDouble is a custom data type that is used for all optional double fields in OTLP
// Protobuf messages.
type OptionalDouble struct {
	orig  *types.DoubleValue
	empty bool
}

// Equal returns true if values are equal.
func (m OptionalDouble) Equal(that OptionalDouble) bool {
	return m.orig.Equal(that.orig)
}

// NewOptionalDouble creates an OptionalDouble from a float64.
func NewOptionalDouble(d float64) OptionalDouble {
	return OptionalDouble{orig: &types.DoubleValue{Value: d}}
}

// MarshalTo converts OptionalDouble into a binary representation. Called by Protobuf serialization.
func (m *OptionalDouble) MarshalTo(data []byte) (n int, err error) {
	return m.orig.MarshalTo(data)
}

// Unmarshal inflates this OptionalDouble from binary representation. Called by Protobuf serialization.
func (m *OptionalDouble) Unmarshal(data []byte) error {
	m.orig = &types.DoubleValue{}
	if len(data) == 0 {
		m.empty = true
		return nil
	}
	return m.orig.Unmarshal(data)
}

// Size returns the size of the data to serialize.
func (m *OptionalDouble) Size() int {
	if m == nil {
		return 0
	}
	return m.orig.Size()
}

func (m *OptionalDouble) IsEmpty() bool {
	return m.empty
}

// TODO: implement
func (m OptionalDouble) MarshalJSON() ([]byte, error) {
	return []byte{}, nil
}

// TODO: implement
func (m *OptionalDouble) UnmarshalJSON(data []byte) error {
	return nil
}

func (m *OptionalDouble) Value() float64 {
	if m == nil || m.orig == nil {
		return 0
	}
	return m.orig.Value
}
