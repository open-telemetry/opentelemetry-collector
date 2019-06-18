// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracetranslator

import (
	"encoding/binary"
	"errors"
)

var (
	// ErrNilTraceID error returned when the TraceID is nil
	ErrNilTraceID = errors.New("TraceID is nil")
	// ErrWrongLenTraceID error returned when the TraceID does not have 16 bytes.
	ErrWrongLenTraceID = errors.New("TraceID does not have 16 bytes")
	// ErrNilSpanID error returned when the SpanID is nil
	ErrNilSpanID = errors.New("SpanID is nil")
	// ErrWrongLenSpanID error returned when the SpanID does not have 8 bytes.
	ErrWrongLenSpanID = errors.New("SpanID does not have 8 bytes")
)

// UInt64ToByteTraceID takes a two uint64 representation of a TraceID and
// converts it to a []byte representation.
func UInt64ToByteTraceID(high, low uint64) []byte {
	if high == 0 && low == 0 {
		return nil
	}
	traceID := make([]byte, 16)
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return traceID
}

// Int64ToByteTraceID takes a two int64 representation of a TraceID and
// converts it to a []byte representation.
func Int64ToByteTraceID(high, low int64) []byte {
	return UInt64ToByteTraceID(uint64(high), uint64(low))
}

// BytesToUInt64TraceID takes a []byte representation of a TraceID and
// converts it to a two uint64 representation.
func BytesToUInt64TraceID(traceID []byte) (uint64, uint64, error) {
	if traceID == nil {
		return 0, 0, ErrNilTraceID
	}
	if len(traceID) != 16 {
		return 0, 0, ErrWrongLenTraceID
	}
	return binary.BigEndian.Uint64(traceID[:8]), binary.BigEndian.Uint64(traceID[8:]), nil
}

// BytesToInt64TraceID takes a []byte representation of a TraceID and
// converts it to a two int64 representation.
func BytesToInt64TraceID(traceID []byte) (int64, int64, error) {
	traceIDHigh, traceIDLow, err := BytesToUInt64TraceID(traceID)
	return int64(traceIDHigh), int64(traceIDLow), err
}

// UInt64ToByteSpanID takes a uint64 representation of a SpanID and
// converts it to a []byte representation.
func UInt64ToByteSpanID(id uint64) []byte {
	if id == 0 {
		return nil
	}
	spanID := make([]byte, 8)
	binary.BigEndian.PutUint64(spanID, id)
	return spanID
}

// Int64ToByteSpanID takes a int64 representation of a SpanID and
// converts it to a []byte representation.
func Int64ToByteSpanID(id int64) []byte {
	return UInt64ToByteSpanID(uint64(id))
}

// BytesToUInt64SpanID takes a []byte representation of a SpanID and
// converts it to a uint64 representation.
func BytesToUInt64SpanID(b []byte) (uint64, error) {
	if b == nil {
		return 0, ErrNilSpanID
	}
	if len(b) != 8 {
		return 0, ErrWrongLenSpanID
	}
	return binary.BigEndian.Uint64(b), nil
}

// BytesToInt64SpanID takes a []byte representation of a SpanID and
// converts it to a int64 representation.
func BytesToInt64SpanID(b []byte) (int64, error) {
	id, err := BytesToUInt64SpanID(b)
	return int64(id), err
}
