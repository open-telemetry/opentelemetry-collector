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

package v1

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// bytesID is a custom data type that is used for all trace_id/span_id fields in OTLP
// Protobuf messages.
type bytesID struct {
	// Just wrap a byte slice for now. In the future we may change to a fixed array
	// or a pair of 64 bit integers to reduce allocations.
	id []byte
}

// Bytes returns the byte slice representation of the ID.
func (t bytesID) Bytes() []byte {
	return t.id
}

// HexString returns hex representation of the ID.
func (t bytesID) HexString() string {
	return hex.EncodeToString(t.id)
}

// Size returns the size of the data to serialize.
func (t *bytesID) Size() int {
	return len(t.id)
}

// Equal returns true if ids are equal.
func (t bytesID) Equal(that bytesID) bool {
	return bytes.Equal(t.id, that.id)
}

// Compare the ids. See bytes.Compare for expected return values.
func (t bytesID) Compare(that bytesID) int {
	return bytes.Compare(t.id, that.id)
}

// MarshalTo converts trace ID into a binary representation. Called by Protobuf serialization.
func (t *bytesID) MarshalTo(data []byte) (n int, err error) {
	if len(data) < len(t.id) {
		return 0, fmt.Errorf("buffer is too short")
	}
	return copy(data, t.id), nil
}

// Unmarshal inflates this trace ID from binary representation. Called by Protobuf serialization.
func (t *bytesID) Unmarshal(data []byte) error {
	if t.id == nil {
		if len(data) == 0 {
			t.id = nil
			return nil
		}
		t.id = make([]byte, len(data))
	}
	copy(t.id, data)
	t.id = t.id[0:len(data)]
	return nil
}

// MarshalJSON converts trace id into a hex string enclosed in quotes.
func (t bytesID) MarshalJSON() ([]byte, error) {
	if t.id == nil {
		return []byte(`""`), nil
	}

	byteLen := len(t.id)

	// 2 chars per byte plus 2 quote chars at the start and end.
	hexLen := 2*byteLen + 2

	b := make([]byte, hexLen)
	hex.Encode(b[1:hexLen-1], t.id)
	b[0], b[hexLen-1] = '"', '"'

	return b, nil
}

// UnmarshalJSON inflates trace id from hex string, possibly enclosed in quotes.
// Called by Protobuf JSON deserialization.
func (t *bytesID) UnmarshalJSON(data []byte) error {
	l := len(data)
	if l < 2 || data[0] != '"' || data[l-1] != '"' {
		return fmt.Errorf("cannot unmarshal ID from string '%s'", string(data))
	}
	data = data[1 : l-1]
	s := string(data)
	if s == `""` {
		t.id = nil
		return nil
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return fmt.Errorf("cannot unmarshal ID from string '%s': %v", string(data), err)
	}
	return t.Unmarshal(b)
}
