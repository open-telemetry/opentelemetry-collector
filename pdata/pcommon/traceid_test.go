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

func TestTraceID(t *testing.T) {
	tid := TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	assert.Equal(t, [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, [16]byte(tid))
	assert.False(t, tid.IsEmpty())
}

func TestNewTraceIDEmpty(t *testing.T) {
	tid := NewTraceIDEmpty()
	assert.Equal(t, [16]byte{}, [16]byte(tid))
	assert.True(t, tid.IsEmpty())
}

func TestTraceIDString(t *testing.T) {
	tid := TraceID([16]byte{})
	assert.Equal(t, "", tid.String())

	tid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	assert.Equal(t, "12345678123456781234567812345678", tid.String())
}

func TestTraceIDImmutable(t *testing.T) {
	initialBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	tid := TraceID(initialBytes)
	assert.Equal(t, TraceID(initialBytes), tid)

	// Get the bytes and try to mutate.
	tid[4] = 0x23

	// Does not change the already created TraceID.
	assert.NotEqual(t, TraceID(initialBytes), tid)
}
