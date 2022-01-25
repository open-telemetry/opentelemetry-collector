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
	encoding_binary "encoding/binary"
	"math"
)

type OptionalDouble struct {
	DoubleValue float64
	isSet       bool
}

func NewOptionalDouble(value float64) OptionalDouble {
	return OptionalDouble{isSet: true, DoubleValue: value}
}

func (o OptionalDouble) Size() (n int) {
	if o.IsEmpty() {
		return 0
	}
	var l int
	_ = l
	n += 9
	return n
}

// IsEmpty returns true if optional field is empty.
func (o OptionalDouble) IsEmpty() bool {
	return !o.isSet
}

// MarshalTo converts the optional double into a binary representation. Called by Protobuf serialization.
func (o *OptionalDouble) MarshalTo(dAtA []byte) (n int, err error) {
	if o.IsEmpty() {
		return 0, nil
	}
	size := o.Size()
	return o.marshalToSizedBuffer(dAtA[:size])
}

// TODO: implement MarshalJSON, UnmarshalJSON

func (o *OptionalDouble) marshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	i -= 8
	encoding_binary.LittleEndian.PutUint64(dAtA[i:], uint64(math.Float64bits(float64(o.DoubleValue))))
	i--
	return len(dAtA) - i, nil
}

func (o *OptionalDouble) Unmarshal(dAtA []byte) error {
	v := uint64(encoding_binary.LittleEndian.Uint64(dAtA))
	v2 := float64(math.Float64frombits(v))
	o.DoubleValue = v2
	o.isSet = true
	return nil
}
