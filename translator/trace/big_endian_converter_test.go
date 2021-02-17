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

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestUInt64ToTraceIDConversion(t *testing.T) {
	assert.Equal(t,
		pdata.NewTraceID([16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
		UInt64ToTraceID(0, 0),
		"Failed 0 conversion:")
	assert.Equal(t,
		pdata.NewTraceID([16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01}),
		UInt64ToTraceID(256*256+256+1, 256+1),
		"Failed simple conversion:")
	assert.Equal(t,
		pdata.NewTraceID([16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}),
		UInt64ToTraceID(0, 5),
		"Failed to convert 0 high:")
	assert.Equal(t,
		UInt64ToTraceID(5, 0),
		pdata.NewTraceID([16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
		UInt64ToTraceID(5, 0),
		"Failed to convert 0 low:")
	assert.Equal(t,
		pdata.NewTraceID([16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}),
		UInt64ToTraceID(math.MaxUint64, 5),
		"Failed to convert MaxUint64:")
}

func TestUInt64ToSpanIDConversion(t *testing.T) {
	assert.Equal(t,
		pdata.NewSpanID([8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
		UInt64ToSpanID(0),
		"Failed 0 conversion:")
	assert.Equal(t,
		pdata.NewSpanID([8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01}),
		UInt64ToSpanID(256*256+256+1),
		"Failed simple conversion:")
	assert.Equal(t,
		pdata.NewSpanID([8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}),
		UInt64ToSpanID(math.MaxUint64),
		"Failed to convert MaxUint64:")
}

func TestTraceIDUInt64RoundTrip(t *testing.T) {
	wh := uint64(0x70605040302010FF)
	wl := uint64(0x0001020304050607)
	gh, gl := TraceIDToUInt64Pair(UInt64ToTraceID(wh, wl))
	assert.Equal(t, wl, gl)
	assert.Equal(t, wh, gh)
}

func TestSpanIdUInt64RoundTrip(t *testing.T) {
	w := uint64(0x0001020304050607)
	assert.Equal(t, w, SpanIDToUInt64(UInt64ToSpanID(w)))
}
