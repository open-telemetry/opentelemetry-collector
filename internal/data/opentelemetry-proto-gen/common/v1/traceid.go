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

// TraceID is a custom data type that is used for all trace_id fields in OTLP
// Protobuf messages.
type TraceID bytesID

// NewTraceID creates a TraceID from a byte slice.
func NewTraceID(bytes []byte) TraceID {
	return TraceID{id: bytes}
}

// Bytes returns the byte slice representation of the ID.
func (t TraceID) Bytes() []byte {
	return bytesID(t).Bytes()
}

// HexString returns hex representation of the ID.
func (t TraceID) HexString() string {
	return bytesID(t).HexString()
}

// Size returns the size of the data to serialize.
func (t *TraceID) Size() int {
	return (*bytesID)(t).Size()
}

// Equal returns true if ids are equal.
func (t TraceID) Equal(that TraceID) bool {
	return bytesID(t).Equal(bytesID(that))
}

// Compare the ids. See bytes.Compare for expected return values.
func (t TraceID) Compare(that TraceID) int {
	return bytesID(t).Compare(bytesID(that))
}

// MarshalTo converts trace ID into a binary representation. Called by Protobuf serialization.
func (t *TraceID) MarshalTo(data []byte) (n int, err error) {
	return (*bytesID)(t).MarshalTo(data)
}

// Unmarshal inflates this trace ID from binary representation. Called by Protobuf serialization.
func (t *TraceID) Unmarshal(data []byte) error {
	return (*bytesID)(t).Unmarshal(data)
}

// MarshalJSON converts trace id into a hex string enclosed in quotes.
func (t TraceID) MarshalJSON() ([]byte, error) {
	return bytesID(t).MarshalJSON()
}

// UnmarshalJSON inflates trace id from hex string, possibly enclosed in quotes.
// Called by Protobuf JSON deserialization.
func (t *TraceID) UnmarshalJSON(data []byte) error {
	return (*bytesID)(t).UnmarshalJSON(data)
}
