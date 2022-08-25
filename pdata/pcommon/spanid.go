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
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/data"
)

// SpanID is span identifier.
type SpanID internal.SpanID

func (ms SpanID) getOrig() data.SpanID {
	return internal.GetOrigSpanID(internal.SpanID(ms))
}

// InvalidSpanID returns an empty (all zero bytes) SpanID.
func InvalidSpanID() SpanID {
	return NewSpanID([8]byte{})
}

// NewSpanID returns a new SpanID from the given byte array.
func NewSpanID(bytes [8]byte) SpanID {
	return SpanID(internal.NewSpanID(data.NewSpanID(bytes)))
}

// Bytes returns the byte array representation of the SpanID.
func (ms SpanID) Bytes() [8]byte {
	return ms.getOrig().Bytes()
}

// HexString returns hex representation of the SpanID.
func (ms SpanID) HexString() string {
	return ms.getOrig().HexString()
}

// IsEmpty returns true if id doesn't contain at least one non-zero byte.
func (ms SpanID) IsEmpty() bool {
	return ms.getOrig().IsEmpty()
}
