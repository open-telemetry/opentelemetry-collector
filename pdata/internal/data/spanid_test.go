// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpanID(t *testing.T) {
	sid := SpanID([8]byte{})
	assert.EqualValues(t, [8]byte{}, sid)
	assert.Equal(t, 0, sid.Size())

	b := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	sid = b
	assert.EqualValues(t, b, sid)
	assert.Equal(t, 8, sid.Size())
}

func TestSpanIDMarshal(t *testing.T) {
	buf := make([]byte, 10)

	sid := SpanID([8]byte{})
	n, err := sid.MarshalTo(buf)
	assert.Equal(t, 0, n)
	require.NoError(t, err)

	sid = [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	n, err = sid.MarshalTo(buf)
	require.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, buf[0:8])

	_, err = sid.MarshalTo(buf[0:1])
	assert.Error(t, err)
}

func TestSpanIDMarshalJSON(t *testing.T) {
	sid := SpanID([8]byte{})
	json, err := sid.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `""`, string(json))

	sid = [8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}
	json, err = sid.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `"1223ad1223ad1223"`, string(json))
}

func TestSpanIDUnmarshal(t *testing.T) {
	buf := []byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}

	sid := SpanID{}
	err := sid.Unmarshal(buf[0:8])
	require.NoError(t, err)
	assert.EqualValues(t, [8]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}, sid)

	err = sid.Unmarshal(buf[0:0])
	require.NoError(t, err)
	assert.EqualValues(t, [8]byte{}, sid)

	err = sid.Unmarshal(nil)
	require.NoError(t, err)
	assert.EqualValues(t, [8]byte{}, sid)

	err = sid.Unmarshal(buf[0:3])
	assert.Error(t, err)
}

func TestSpanIDUnmarshalJSON(t *testing.T) {
	sid := SpanID{}
	err := sid.UnmarshalJSON([]byte(`""`))
	require.NoError(t, err)
	assert.EqualValues(t, [8]byte{}, sid)

	err = sid.UnmarshalJSON([]byte(`"1234567812345678"`))
	require.NoError(t, err)
	assert.EqualValues(t, [8]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, sid)

	err = sid.UnmarshalJSON([]byte(`1234567812345678`))
	require.NoError(t, err)
	assert.EqualValues(t, [8]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, sid)

	err = sid.UnmarshalJSON([]byte(`"nothex"`))
	require.Error(t, err)

	err = sid.UnmarshalJSON([]byte(`"1"`))
	require.Error(t, err)

	err = sid.UnmarshalJSON([]byte(`"123"`))
	require.Error(t, err)

	err = sid.UnmarshalJSON([]byte(`"`))
	assert.Error(t, err)
}
