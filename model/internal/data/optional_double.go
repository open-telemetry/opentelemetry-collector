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

// OptionalDouble is a custom data type that is used for all optional double fields in OTLP
// Protobuf messages.
type OptionalDouble struct {
	orig float64
}

// Equal returns true if values are equal.
func (t OptionalDouble) Equal(that OptionalDouble) bool {
	return t.orig == that.orig
}

// NewOptionalDouble creates an OptionalDouble from a float64.
func NewOptionalDouble(d float64) *OptionalDouble {
	return &OptionalDouble{orig: d}
}

// TODO: implement all the methods below this line
func (t OptionalDouble) Marshal() ([]byte, error) {
	return []byte{}, nil
}
func (t *OptionalDouble) MarshalTo(data []byte) (n int, err error) {
	return 0, nil
}
func (t *OptionalDouble) Unmarshal(data []byte) error {
	return nil
}
func (t *OptionalDouble) Size() int {
	return 0
}

func (t OptionalDouble) MarshalJSON() ([]byte, error) {
	return []byte{}, nil
}
func (t *OptionalDouble) UnmarshalJSON(data []byte) error {
	return nil
}
