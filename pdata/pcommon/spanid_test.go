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

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanID(t *testing.T) {
	sid := SpanID([8]byte{1, 2, 3, 4, 4, 3, 2, 1})
	assert.Equal(t, [8]byte{1, 2, 3, 4, 4, 3, 2, 1}, [8]byte(sid))
	assert.False(t, sid.IsEmpty())
}

func TestNewSpanIDEmpty(t *testing.T) {
	sid := NewSpanIDEmpty()
	assert.Equal(t, [8]byte{}, [8]byte(sid))
	assert.True(t, sid.IsEmpty())
}

func TestSpanIDString(t *testing.T) {
	sid := SpanID([8]byte{})
	assert.Equal(t, "", sid.String())

	sid = SpanID([8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23})
	assert.Equal(t, "1223ad1223ad1223", sid.String())
}

func TestSpanIDImmutable(t *testing.T) {
	initialBytes := [8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}
	sid := SpanID(initialBytes)
	assert.Equal(t, SpanID(initialBytes), sid)

	// Get the bytes and try to mutate.
	sid[4] = 0x89

	// Does not change the already created SpanID.
	assert.NotEqual(t, SpanID(initialBytes), sid)
}
