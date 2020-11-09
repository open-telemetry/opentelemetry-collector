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

package tracetranslator

import (
	"encoding/binary"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// UInt64ToByteTraceID takes a two uint64 representation of a TraceID and
// converts it to a []byte representation.
func UInt64ToTraceID(high, low uint64) pdata.TraceID {
	traceID := [16]byte{}
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return pdata.NewTraceID(traceID)
}

// UInt64ToByteTraceID takes a two uint64 representation of a TraceID and
// converts it to a []byte representation.
func UInt64ToByteTraceID(high, low uint64) [16]byte {
	traceID := [16]byte{}
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return traceID
}

// Int64ToByteTraceID takes a two int64 representation of a TraceID and
// converts it to a []byte representation.
func Int64ToTraceID(high, low int64) pdata.TraceID {
	return UInt64ToTraceID(uint64(high), uint64(low))
}

// Int64ToByteTraceID takes a two int64 representation of a TraceID and
// converts it to a []byte representation.
func Int64ToByteTraceID(high, low int64) [16]byte {
	return UInt64ToByteTraceID(uint64(high), uint64(low))
}

// BytesToUInt64TraceID takes a []byte representation of a TraceID and
// converts it to a two uint64 representation.
func BytesToUInt64TraceID(traceID [16]byte) (uint64, uint64) {
	return binary.BigEndian.Uint64(traceID[:8]), binary.BigEndian.Uint64(traceID[8:])
}

// BytesToInt64TraceID takes a []byte representation of a TraceID and
// converts it to a two int64 representation.
func BytesToInt64TraceID(traceID [16]byte) (int64, int64) {
	traceIDHigh, traceIDLow := BytesToUInt64TraceID(traceID)
	return int64(traceIDHigh), int64(traceIDLow)
}

// TraceIDToUInt64Pair takes a pdata.TraceID and converts it to a pair of uint64 representation.
func TraceIDToUInt64Pair(traceID pdata.TraceID) (uint64, uint64) {
	return BytesToUInt64TraceID(traceID.Bytes())
}

// UInt64ToByteSpanID takes a uint64 representation of a SpanID and
// converts it to a []byte representation.
func UInt64ToByteSpanID(id uint64) [8]byte {
	spanID := [8]byte{}
	binary.BigEndian.PutUint64(spanID[:], id)
	return spanID
}

// UInt64ToSpanID takes a uint64 representation of a SpanID and
// converts it to a pdata.SpanID representation.
func UInt64ToSpanID(id uint64) pdata.SpanID {
	return pdata.NewSpanID(UInt64ToByteSpanID(id))
}

// Int64ToByteSpanID takes a int64 representation of a SpanID and
// converts it to a []byte representation.
func Int64ToByteSpanID(id int64) [8]byte {
	return UInt64ToByteSpanID(uint64(id))
}

// Int64ToByteSpanID takes a int64 representation of a SpanID and
// converts it to a []byte representation.
func Int64ToSpanID(id int64) pdata.SpanID {
	return UInt64ToSpanID(uint64(id))
}

// BytesToUInt64SpanID takes a []byte representation of a SpanID and
// converts it to a uint64 representation.
func BytesToUInt64SpanID(b [8]byte) uint64 {
	return binary.BigEndian.Uint64(b[:])
}

// BytesToInt64SpanID takes a []byte representation of a SpanID and
// converts it to a int64 representation.
func BytesToInt64SpanID(b [8]byte) int64 {
	return int64(BytesToUInt64SpanID(b))
}
