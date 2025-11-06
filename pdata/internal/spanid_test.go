// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestSpanID(t *testing.T) {
	sid := SpanID([spanIDSize]byte{})
	assert.EqualValues(t, [spanIDSize]byte{}, sid)
	assert.Equal(t, 0, sid.SizeProto())

	b := [spanIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8}
	sid = b
	assert.EqualValues(t, b, sid)
	assert.Equal(t, 8, sid.SizeProto())
}

func TestSpanIDMarshal(t *testing.T) {
	buf := make([]byte, spanIDSize)

	sid := SpanID([spanIDSize]byte{})
	n := sid.MarshalProto(buf)
	assert.Equal(t, 0, n)

	sid = [spanIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8}
	n = sid.MarshalProto(buf)
	assert.Equal(t, spanIDSize, n)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, buf[0:spanIDSize])
}

func TestSpanIDUnmarshal(t *testing.T) {
	buf := [spanIDSize]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}

	sid := SpanID{}
	err := sid.UnmarshalProto(buf[:])
	require.NoError(t, err)
	assert.EqualValues(t, [spanIDSize]byte{0x12, 0x23, 0xAD, 0x12, 0x23, 0xAD, 0x12, 0x23}, sid)

	err = sid.UnmarshalProto(buf[0:0])
	require.NoError(t, err)
	assert.EqualValues(t, [spanIDSize]byte{}, sid)

	err = sid.UnmarshalProto(nil)
	require.NoError(t, err)
	assert.EqualValues(t, [spanIDSize]byte{}, sid)

	err = sid.UnmarshalProto(buf[0:3])
	assert.Error(t, err)
}

func TestSpanIDMarshalAndUnmarshalJSON(t *testing.T) {
	stream := json.BorrowStream(nil)
	defer json.ReturnStream(stream)
	src := SpanID([spanIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78})
	src.MarshalJSON(stream)
	require.NoError(t, stream.Error())

	iter := json.BorrowIterator(stream.Buffer())
	defer json.ReturnIterator(iter)
	dest := SpanID{}
	dest.UnmarshalJSON(iter)
	require.NoError(t, iter.Error())

	assert.Equal(t, src, dest)
}
