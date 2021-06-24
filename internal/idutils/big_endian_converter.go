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

package idutils

import (
	"encoding/binary"

	"go.opentelemetry.io/collector/model/pdata"
)

// UInt64ToTraceID converts the pair of uint64 representation of a TraceID to pdata.TraceID.
func UInt64ToTraceID(high, low uint64) pdata.TraceID {
	traceID := [16]byte{}
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return pdata.NewTraceID(traceID)
}

// TraceIDToUInt64Pair converts the pdata.TraceID to a pair of uint64 representation.
func TraceIDToUInt64Pair(traceID pdata.TraceID) (uint64, uint64) {
	bytes := traceID.Bytes()
	return binary.BigEndian.Uint64(bytes[:8]), binary.BigEndian.Uint64(bytes[8:])
}

// UInt64ToSpanID converts the uint64 representation of a SpanID to pdata.SpanID.
func UInt64ToSpanID(id uint64) pdata.SpanID {
	spanID := [8]byte{}
	binary.BigEndian.PutUint64(spanID[:], id)
	return pdata.NewSpanID(spanID)
}

// SpanIDToUInt64 converts the pdata.SpanID to uint64 representation.
func SpanIDToUInt64(spanID pdata.SpanID) uint64 {
	bytes := spanID.Bytes()
	return binary.BigEndian.Uint64(bytes[:])
}
