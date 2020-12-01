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

package pdata

import (
	"go.opentelemetry.io/collector/internal/data"
)

// SpanID is an alias of OTLP SpanID data type.
type SpanID data.SpanID

func InvalidSpanID() SpanID {
	return SpanID(data.NewSpanID([8]byte{}))
}

func NewSpanID(bytes [8]byte) SpanID {
	return SpanID(data.NewSpanID(bytes))
}

// Bytes returns the byte array representation of the SpanID.
func (t SpanID) Bytes() [8]byte {
	return data.SpanID(t).Bytes()
}

// HexString returns hex representation of the SpanID.
func (t SpanID) HexString() string {
	return data.SpanID(t).HexString()
}

// IsValid returns true if id contains at leas one non-zero byte.
func (t SpanID) IsValid() bool {
	return data.SpanID(t).IsValid()
}
