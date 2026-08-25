// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestProfileID(t *testing.T) {
	tid := ProfileID([profileIDSize]byte{})
	assert.EqualValues(t, [profileIDSize]byte{}, tid)
	assert.Equal(t, 0, tid.SizeProto())

	b := [profileIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	tid = b
	assert.EqualValues(t, b, tid)
	assert.Equal(t, profileIDSize, tid.SizeProto())
}

func TestProfileIDMarshal(t *testing.T) {
	buf := make([]byte, profileIDSize)

	tid := ProfileID([profileIDSize]byte{})
	n := tid.MarshalProto(buf)
	assert.Equal(t, 0, n)

	tid = [profileIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	n = tid.MarshalProto(buf)
	assert.Equal(t, profileIDSize, n)
	assert.Equal(t, []byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, buf[0:profileIDSize])
}

func TestProfileIDUnmarshal(t *testing.T) {
	buf := [profileIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}

	tid := ProfileID{}
	err := tid.UnmarshalProto(buf[:])
	require.NoError(t, err)
	assert.EqualValues(t, buf, tid)

	err = tid.UnmarshalProto(buf[0:0])
	require.NoError(t, err)
	assert.EqualValues(t, [profileIDSize]byte{}, tid)

	err = tid.UnmarshalProto(nil)
	require.NoError(t, err)
	assert.EqualValues(t, [profileIDSize]byte{}, tid)
}

func TestProfileIDMarshalAndUnmarshalJSON(t *testing.T) {
	stream := json.BorrowStream(nil)
	defer json.ReturnStream(stream)
	src := ProfileID([profileIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78})
	src.MarshalJSON(stream)
	require.NoError(t, stream.Error())

	iter := json.BorrowIterator(stream.Buffer())
	defer json.ReturnIterator(iter)
	dest := ProfileID{}
	dest.UnmarshalJSON(iter)
	require.NoError(t, iter.Error())

	assert.Equal(t, src, dest)
}
