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

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanID(t *testing.T) {
	sid := SpanID([8]byte{})
	assert.EqualValues(t, [8]byte{}, sid)
	assert.EqualValues(t, 0, sid.Size())

	b := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	sid = b
	assert.EqualValues(t, b, sid)
	assert.EqualValues(t, 8, sid.Size())
}

func TestSpanIDMarshal(t *testing.T) {
	buf := make([]byte, 10)

	sid := SpanID([8]byte{})
	n, err := sid.MarshalTo(buf)
	assert.EqualValues(t, 0, n)
	assert.NoError(t, err)

	sid = [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	n, err = sid.MarshalTo(buf)
	assert.NoError(t, err)
	assert.EqualValues(t, 8, n)
	assert.EqualValues(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, buf[0:8])

	_, err = sid.MarshalTo(buf[0:1])
	assert.Error(t, err)
}

func TestSpanIDMarshalJSON(t *testing.T) {
	sid := SpanID([8]byte{})
	json, err := sid.MarshalJSON()
	assert.EqualValues(t, []byte(`""`), json)
	assert.NoError(t, err)

	sid = [8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}
	json, err = sid.MarshalJSON()
	assert.EqualValues(t, []byte(`"1223ad1223ad1223"`), json)
	assert.NoError(t, err)
}

func TestSpanIDUnmarshal(t *testing.T) {
	buf := []byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}

	sid := SpanID{}
	err := sid.Unmarshal(buf[0:8])
	assert.NoError(t, err)
	assert.EqualValues(t, [8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}, sid)

	err = sid.Unmarshal(buf[0:0])
	assert.NoError(t, err)
	assert.EqualValues(t, [8]byte{}, sid)

	err = sid.Unmarshal(nil)
	assert.NoError(t, err)
	assert.EqualValues(t, [8]byte{}, sid)

	err = sid.Unmarshal(buf[0:3])
	assert.Error(t, err)
}

func TestSpanIDUnmarshalJSON(t *testing.T) {
	sid := SpanID{}
	err := sid.UnmarshalJSON([]byte(`""`))
	assert.NoError(t, err)
	assert.EqualValues(t, [8]byte{}, sid)

	err = sid.UnmarshalJSON([]byte(`"1234567812345678"`))
	assert.NoError(t, err)
	assert.EqualValues(t, [8]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, sid)

	err = sid.UnmarshalJSON([]byte(`1234567812345678`))
	assert.NoError(t, err)
	assert.EqualValues(t, [8]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, sid)

	err = sid.UnmarshalJSON([]byte(`"nothex"`))
	assert.Error(t, err)

	err = sid.UnmarshalJSON([]byte(`"1"`))
	assert.Error(t, err)

	err = sid.UnmarshalJSON([]byte(`"123"`))
	assert.Error(t, err)

	err = sid.UnmarshalJSON([]byte(`"`))
	assert.Error(t, err)
}
