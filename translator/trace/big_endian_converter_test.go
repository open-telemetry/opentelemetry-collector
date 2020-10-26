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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUInt64ToBytesTraceIDConversion(t *testing.T) {
	assert.Equal(t,
		[16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		UInt64ToByteTraceID(0, 0),
		"Failed 0 conversion:")
	assert.Equal(t,
		[16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01},
		UInt64ToByteTraceID(256*256+256+1, 256+1),
		"Failed simple conversion:")
	assert.Equal(t,
		[16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		UInt64ToByteTraceID(0, 5),
		"Failed to convert 0 high:")
	assert.Equal(t,
		UInt64ToByteTraceID(5, 0),
		[16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		UInt64ToByteTraceID(5, 0),
		"Failed to convert 0 low:")
	assert.Equal(t,
		[16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		UInt64ToByteTraceID(math.MaxUint64, 5),
		"Failed to convert MaxUint64:")
}

func TestInt64ToBytesTraceIDConversion(t *testing.T) {
	assert.Equal(t,
		[16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Int64ToByteTraceID(0, 0),
		"Failed 0 conversion:")
	assert.Equal(t,
		[16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		Int64ToByteTraceID(0, -1),
		"Failed to convert negative low:")
	assert.Equal(t,
		[16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		Int64ToByteTraceID(-2, 5),
		"Failed to convert negative high:")
	assert.Equal(t,
		[16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Int64ToByteTraceID(5, math.MinInt64),
		"Failed to convert MinInt64:")
}

func TestUInt64ToBytesSpanIDConversion(t *testing.T) {
	assert.Equal(t,
		[8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		UInt64ToByteSpanID(0),
		"Failed 0 conversion:")
	assert.Equal(t,
		[8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01},
		UInt64ToByteSpanID(256*256+256+1),
		"Failed simple conversion:")
	assert.Equal(t,
		[8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		UInt64ToByteSpanID(math.MaxUint64),
		"Failed to convert MaxUint64:")
}

func TestInt64ToBytesSpanIDConversion(t *testing.T) {
	assert.Equal(t,
		[8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Int64ToByteSpanID(0),
		"Failed 0 conversion:")
	assert.Equal(t,
		[8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05},
		Int64ToByteSpanID(5),
		"Failed to convert positive id:")
	assert.Equal(t,
		[8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		Int64ToByteSpanID(-1),
		"Failed to convert negative id:")
	assert.Equal(t,
		[8]byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Int64ToByteSpanID(math.MinInt64),
		"Failed to convert MinInt64:")
}

func TestTraceIDInt64RoundTrip(t *testing.T) {
	wh := int64(0x70605040302010FF)
	wl := int64(0x0001020304050607)
	gh, gl := BytesToInt64TraceID(Int64ToByteTraceID(wh, wl))
	if gh != wh || gl != wl {
		t.Errorf("Round trip of TraceID failed:\n\tGot: (0x%0x, 0x%0x)\n\tWant: (0x%0x, 0x%0x)", gl, gh, wl, wh)
	}
}

func TestTraceIDUInt64RoundTrip(t *testing.T) {
	wh := uint64(0x70605040302010FF)
	wl := uint64(0x0001020304050607)
	gh, gl := BytesToUInt64TraceID(UInt64ToByteTraceID(wh, wl))
	assert.Equal(t, wl, gl)
	assert.Equal(t, wh, gh)
}

func TestSpanIdInt64RoundTrip(t *testing.T) {
	w := int64(0x0001020304050607)
	assert.Equal(t, w, BytesToInt64SpanID(Int64ToByteSpanID(w)))
}

func TestSpanIdUInt64RoundTrip(t *testing.T) {
	w := uint64(0x0001020304050607)
	assert.Equal(t, w, BytesToUInt64SpanID(UInt64ToByteSpanID(w)))
}
