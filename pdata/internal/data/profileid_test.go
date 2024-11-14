// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileID(t *testing.T) {
	tid := ProfileID([16]byte{})
	assert.EqualValues(t, [16]byte{}, tid)
	assert.EqualValues(t, 0, tid.Size())

	b := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	tid = b
	assert.EqualValues(t, b, tid)
	assert.EqualValues(t, 16, tid.Size())
}

func TestProfileIDMarshal(t *testing.T) {
	buf := make([]byte, 20)

	tid := ProfileID([16]byte{})
	n, err := tid.MarshalTo(buf)
	assert.EqualValues(t, 0, n)
	require.NoError(t, err)

	tid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	n, err = tid.MarshalTo(buf)
	assert.EqualValues(t, 16, n)
	assert.EqualValues(t, []byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, buf[0:16])
	require.NoError(t, err)

	_, err = tid.MarshalTo(buf[0:1])
	assert.Error(t, err)
}

func TestProfileIDMarshalJSON(t *testing.T) {
	tid := ProfileID([16]byte{})
	json, err := tid.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `""`, string(json))

	tid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	json, err = tid.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `"12345678123456781234567812345678"`, string(json))
}

func TestProfileIDUnmarshal(t *testing.T) {
	buf := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}

	tid := ProfileID{}
	err := tid.Unmarshal(buf[0:16])
	require.NoError(t, err)
	assert.EqualValues(t, buf, tid)

	err = tid.Unmarshal(buf[0:0])
	require.NoError(t, err)
	assert.EqualValues(t, [16]byte{}, tid)

	err = tid.Unmarshal(nil)
	require.NoError(t, err)
	assert.EqualValues(t, [16]byte{}, tid)
}

func TestProfileIDUnmarshalJSON(t *testing.T) {
	tid := ProfileID([16]byte{})
	err := tid.UnmarshalJSON([]byte(`""`))
	require.NoError(t, err)
	assert.EqualValues(t, [16]byte{}, tid)

	err = tid.UnmarshalJSON([]byte(`""""`))
	require.Error(t, err)

	tidBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	err = tid.UnmarshalJSON([]byte(`"12345678123456781234567812345678"`))
	require.NoError(t, err)
	assert.EqualValues(t, tidBytes, tid)

	err = tid.UnmarshalJSON([]byte(`12345678123456781234567812345678`))
	require.NoError(t, err)
	assert.EqualValues(t, tidBytes, tid)

	err = tid.UnmarshalJSON([]byte(`"nothex"`))
	require.Error(t, err)

	err = tid.UnmarshalJSON([]byte(`"1"`))
	require.Error(t, err)

	err = tid.UnmarshalJSON([]byte(`"123"`))
	require.Error(t, err)

	err = tid.UnmarshalJSON([]byte(`"`))
	assert.Error(t, err)
}
