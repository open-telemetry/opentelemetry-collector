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

// TraceID is an alias of OTLP TraceID data type.
type TraceID data.TraceID

func InvalidTraceID() TraceID {
	return TraceID(data.NewTraceID([16]byte{}))
}

func NewTraceID(bytes [16]byte) TraceID {
	return TraceID(data.NewTraceID(bytes))
}

// Bytes returns the byte array representation of the TraceID.
func (t TraceID) Bytes() [16]byte {
	return data.TraceID(t).Bytes()
}

// HexString returns hex representation of the TraceID.
func (t TraceID) HexString() string {
	return data.TraceID(t).HexString()
}

// IsValid returns true if id contains at leas one non-zero byte.
func (t TraceID) IsValid() bool {
	return data.TraceID(t).IsValid()
}
