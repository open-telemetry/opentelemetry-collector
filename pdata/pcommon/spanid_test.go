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

func TestEmptySpanID(t *testing.T) {
	sid := EmptySpanID
	assert.Equal(t, [8]byte{}, sid.Bytes())
	assert.True(t, sid.IsEmpty())
}

func TestNewSpanID(t *testing.T) {
	sid := NewSpanID([8]byte{1, 2, 3, 4, 4, 3, 2, 1})
	assert.Equal(t, [8]byte{1, 2, 3, 4, 4, 3, 2, 1}, sid.Bytes())
	assert.False(t, sid.IsEmpty())
}

func TestSpanIDHexString(t *testing.T) {
	sid := NewSpanID([8]byte{})
	assert.Equal(t, "", sid.HexString())

	sid = NewSpanID([8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23})
	assert.Equal(t, "1223ad1223ad1223", sid.HexString())
}

func TestSpanIDImmutable(t *testing.T) {
	initialBytes := [8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}
	sid := NewSpanID(initialBytes)
	assert.Equal(t, initialBytes, sid.Bytes())

	// Get the bytes and try to mutate.
	bytes := sid.Bytes()
	bytes[4] = 0x89

	// Does not change the already created SpanID.
	assert.NotEqual(t, bytes, sid.Bytes())
	assert.Equal(t, initialBytes, sid.Bytes())
}
