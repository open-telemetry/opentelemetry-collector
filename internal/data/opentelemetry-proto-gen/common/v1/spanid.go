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

// SpanID is a custom data type that is used for all span_id fields in OTLP
// Protobuf messages.
type SpanID bytesID

// NewSpanID creates a SpanID from a byte slice.
func NewSpanID(bytes []byte) SpanID {
	return SpanID{id: bytes}
}

// Bytes returns the byte slice representation of the ID.
func (t SpanID) Bytes() []byte {
	return bytesID(t).Bytes()
}

// HexString returns hex representation of the ID.
func (t SpanID) HexString() string {
	return bytesID(t).HexString()
}

// Size returns the size of the data to serialize.
func (t *SpanID) Size() int {
	return (*bytesID)(t).Size()
}

// Equal returns true if ids are equal.
func (t SpanID) Equal(that SpanID) bool {
	return bytesID(t).Equal(bytesID(that))
}

// Compare the ids. See bytes.Compare for expected return values.
func (t SpanID) Compare(that SpanID) int {
	return bytesID(t).Compare(bytesID(that))
}

// MarshalTo converts trace ID into a binary representation. Called by Protobuf serialization.
func (t *SpanID) MarshalTo(data []byte) (n int, err error) {
	return (*bytesID)(t).MarshalTo(data)
}

// Unmarshal inflates this trace ID from binary representation. Called by Protobuf serialization.
func (t *SpanID) Unmarshal(data []byte) error {
	return (*bytesID)(t).Unmarshal(data)
}

// MarshalJSON converts trace id into a hex string enclosed in quotes.
func (t SpanID) MarshalJSON() ([]byte, error) {
	return bytesID(t).MarshalJSON()
}

// UnmarshalJSON inflates trace id from hex string, possibly enclosed in quotes.
// Called by Protobuf JSON deserialization.
func (t *SpanID) UnmarshalJSON(data []byte) error {
	return (*bytesID)(t).UnmarshalJSON(data)
}
