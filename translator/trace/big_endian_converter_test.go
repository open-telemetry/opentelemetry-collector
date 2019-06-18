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
	"math"
	"reflect"
	"testing"
)

func TestUInt64ToBytesTraceIDConversion(t *testing.T) {
	if nil != UInt64ToByteTraceID(0, 0) {
		t.Errorf("Failed to convert 0 TraceID:\n\tGot: %v\nWant: nil", UInt64ToByteTraceID(0, 0))
	}
	assertEqual(t, UInt64ToByteTraceID(256*256+256+1, 256+1),
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01},
		"Failed simple conversion:")
	assertEqual(t, UInt64ToByteTraceID(0, 5),
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		"Failed to convert 0 high:")
	assertEqual(t, UInt64ToByteTraceID(5, 0),
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		"Failed to convert 0 low:")
	assertEqual(t, UInt64ToByteTraceID(math.MaxUint64, 5),
		[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		"Failed to convert MaxUint64:")
}

func TestInt64ToBytesTraceIDConversion(t *testing.T) {
	if nil != Int64ToByteTraceID(0, 0) {
		t.Errorf("Failed to convert 0 TraceID:\n\tGot: %v\nWant: nil", Int64ToByteTraceID(0, 0))
	}
	assertEqual(t, Int64ToByteTraceID(0, -1),
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		"Failed to convert negative low:")
	assertEqual(t, Int64ToByteTraceID(-2, 5),
		[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		"Failed to convert negative high:")
	assertEqual(t, Int64ToByteTraceID(5, math.MinInt64),
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		"Failed to convert MinInt64:")
}

func TestBytesToUInt64TraceIDErrors(t *testing.T) {
	if _, _, err := BytesToUInt64TraceID(nil); err != ErrNilTraceID {
		t.Errorf("Got: %v\nWant: %v", err, ErrNilTraceID)
	}
	longTraceID := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00}
	if _, _, err := BytesToUInt64TraceID(longTraceID); err != ErrWrongLenTraceID {
		t.Errorf("Got: %v\nWant: %v", err, ErrWrongLenTraceID)
	}
}

func TestUInt64ToBytesSpanIDConversion(t *testing.T) {
	if nil != UInt64ToByteSpanID(0) {
		t.Errorf("Failed to convert 0 SpanID:\n\tGot: %v\nWant: nil", UInt64ToByteSpanID(0))
	}
	assertEqual(t, UInt64ToByteSpanID(256*256+256+1),
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01},
		"Failed simple conversion:")
	assertEqual(t, UInt64ToByteSpanID(math.MaxUint64),
		[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		"Failed to convert MaxUint64:")
}

func TestInt64ToBytesSpanIDConversion(t *testing.T) {
	if nil != Int64ToByteSpanID(0) {
		t.Errorf("Failed to convert 0 SpanID:\n\tGot: %v\nWant: nil", Int64ToByteSpanID(0))
	}
	assertEqual(t, Int64ToByteSpanID(5),
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		"Failed to convert positive id:")
	assertEqual(t, Int64ToByteSpanID(-1),
		[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		"Failed to convert negative id:")
	assertEqual(t, Int64ToByteSpanID(math.MinInt64),
		[]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		"Failed to convert MinInt64:")
}

func TestBytesToUInt64SpanIDErrors(t *testing.T) {
	if _, err := BytesToUInt64SpanID(nil); err != ErrNilSpanID {
		t.Errorf("Got: %v\nWant: %v", err, ErrNilSpanID)
	}
	if _, err := BytesToUInt64SpanID([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}); err != ErrWrongLenSpanID {
		t.Errorf("Got: %v\nWant: %v", err, ErrWrongLenSpanID)
	}
}

func TestTraceIDInt64RoundTrip(t *testing.T) {
	wh := int64(0x70605040302010FF)
	wl := int64(0x0001020304050607)
	gh, gl, err := BytesToInt64TraceID(Int64ToByteTraceID(wh, wl))
	if err != nil {
		t.Errorf("Error converting from bytes TraceID: %v", err)
	}
	if gh != wh || gl != wl {
		t.Errorf("Round trip of TraceID failed:\n\tGot: (0x%0x, 0x%0x)\n\tWant: (0x%0x, 0x%0x)", gl, gh, wl, wh)
	}
}

func TestTraceIDUInt64RoundTrip(t *testing.T) {
	wh := uint64(0x70605040302010FF)
	wl := uint64(0x0001020304050607)
	gh, gl, err := BytesToUInt64TraceID(UInt64ToByteTraceID(wh, wl))
	if err != nil {
		t.Errorf("Error converting from bytes TraceID: %v", err)
	}
	if gl != wl || gh != wh {
		t.Errorf("Round trip of TraceID failed:\n\tGot: (0x%0x, 0x%0x)\n\tWant: (0x%0x, 0x%0x)", gl, gh, wl, wh)
	}
}

func TestSpanIdInt64RoundTrip(t *testing.T) {
	w := int64(0x0001020304050607)
	g, err := BytesToInt64SpanID(Int64ToByteSpanID(w))
	if err != nil {
		t.Errorf("Error converting from OC span id: %v", err)
	}
	if g != w {
		t.Errorf("Round trip of SpanId failed:\n\tGot: 0x%0x\n\tWant: 0x%0x", g, w)
	}
}

func TestSpanIdUInt64RoundTrip(t *testing.T) {
	w := uint64(0x0001020304050607)
	g, err := BytesToUInt64SpanID(UInt64ToByteSpanID(w))
	if err != nil {
		t.Errorf("Error converting from OC span id: %v", err)
	}
	if g != w {
		t.Errorf("Round trip of SpanId failed:\n\tGot: 0x%0x\n\tWant: 0x%0x", g, w)
	}
}

func assertEqual(t *testing.T, got []byte, want []byte, em string) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s\n\tGot: %v\n\tWant: %v\n", em, got, want)
	}
}
