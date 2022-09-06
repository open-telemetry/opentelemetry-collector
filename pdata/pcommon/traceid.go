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

var emptyTraceID = TraceID([16]byte{})

// Deprecated: [v0.60.0] use NewTraceIDEmpty.
var EmptyTraceID = TraceID([16]byte{})

// TraceID is a trace identifier.
type TraceID [16]byte

// NewTraceIDEmpty returns a new empty (all zero bytes) TraceID.
func NewTraceIDEmpty() TraceID {
	return emptyTraceID
}

// Deprecated: [v0.60.0] use TraceID(bytes).
func NewTraceID(bytes [16]byte) TraceID {
	return bytes
}

// Deprecated: [v0.60.0] use [16]byte(tid).
func (ms TraceID) Bytes() [16]byte {
	return ms
}

// HexString returns hex representation of the TraceID.
func (ms TraceID) HexString() string {
	if ms.IsEmpty() {
		return ""
	}
	return hex.EncodeToString(ms[:])
}

// IsEmpty returns true if id doesn't contain at least one non-zero byte.
func (ms TraceID) IsEmpty() bool {
	return data.TraceID(ms).IsEmpty()
}
