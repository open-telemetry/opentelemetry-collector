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

package pcommon // import "go.opentelemetry.io/collector/pdata/pcommon"
import (
	"encoding/hex"

	"go.opentelemetry.io/collector/pdata/internal/data"
)

// EmptySpanID represents the empty (all zero bytes) SpanID.
var EmptySpanID = NewSpanID([8]byte{})

// SpanID is span identifier.
type SpanID struct {
	orig [8]byte
}

// Deprecated: [v0.59.0] use EmptySpanID.
func InvalidSpanID() SpanID {
	return EmptySpanID
}

// NewSpanID returns a new SpanID from the given byte array.
func NewSpanID(bytes [8]byte) SpanID {
	return SpanID{orig: bytes}
}

// Bytes returns the byte array representation of the SpanID.
func (ms SpanID) Bytes() [8]byte {
	return ms.orig
}

// HexString returns hex representation of the SpanID.
func (ms SpanID) HexString() string {
	if ms.IsEmpty() {
		return ""
	}
	return hex.EncodeToString(ms.orig[:])
}

// IsEmpty returns true if id doesn't contain at least one non-zero byte.
func (ms SpanID) IsEmpty() bool {
	return data.SpanID(ms.orig).IsEmpty()
}
