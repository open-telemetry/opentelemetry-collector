// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceID(t *testing.T) {
	tid := TraceID([16]byte{})
	assert.EqualValues(t, [16]byte{}, tid)
	assert.EqualValues(t, 0, tid.Size())

	b := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	tid = b
	assert.EqualValues(t, b, tid)
	assert.EqualValues(t, 16, tid.Size())
}

func TestTraceIDMarshal(t *testing.T) {
	buf := make([]byte, 20)

	tid := TraceID([16]byte{})
	n, err := tid.MarshalTo(buf)
	assert.EqualValues(t, 0, n)
	assert.NoError(t, err)

	tid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	n, err = tid.MarshalTo(buf)
	assert.EqualValues(t, 16, n)
	assert.EqualValues(t, []byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, buf[0:16])
	assert.NoError(t, err)

	_, err = tid.MarshalTo(buf[0:1])
	assert.Error(t, err)
}

func TestTraceIDMarshalJSON(t *testing.T) {
	tid := TraceID([16]byte{})
	json, err := tid.MarshalJSON()
	assert.EqualValues(t, []byte(`""`), json)
	assert.NoError(t, err)

	tid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	json, err = tid.MarshalJSON()
	assert.EqualValues(t, []byte(`"12345678123456781234567812345678"`), json)
	assert.NoError(t, err)
}

func TestTraceIDUnmarshal(t *testing.T) {
	buf := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}

	tid := TraceID{}
	err := tid.Unmarshal(buf[0:16])
	assert.NoError(t, err)
	assert.EqualValues(t, buf, tid)

	err = tid.Unmarshal(buf[0:0])
	assert.NoError(t, err)
	assert.EqualValues(t, [16]byte{}, tid)

	err = tid.Unmarshal(nil)
	assert.NoError(t, err)
	assert.EqualValues(t, [16]byte{}, tid)
}

func TestTraceIDUnmarshalJSON(t *testing.T) {
	tid := TraceID([16]byte{})
	err := tid.UnmarshalJSON([]byte(`""`))
	assert.NoError(t, err)
	assert.EqualValues(t, [16]byte{}, tid)

	err = tid.UnmarshalJSON([]byte(`""""`))
	assert.Error(t, err)

	tidBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	err = tid.UnmarshalJSON([]byte(`"12345678123456781234567812345678"`))
	assert.NoError(t, err)
	assert.EqualValues(t, tidBytes, tid)

	err = tid.UnmarshalJSON([]byte(`12345678123456781234567812345678`))
	assert.NoError(t, err)
	assert.EqualValues(t, tidBytes, tid)

	err = tid.UnmarshalJSON([]byte(`"nothex"`))
	assert.Error(t, err)

	err = tid.UnmarshalJSON([]byte(`"1"`))
	assert.Error(t, err)

	err = tid.UnmarshalJSON([]byte(`"123"`))
	assert.Error(t, err)

	err = tid.UnmarshalJSON([]byte(`"`))
	assert.Error(t, err)
}
