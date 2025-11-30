// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestTraceID(t *testing.T) {
	tid := TraceID([traceIDSize]byte{})
	assert.EqualValues(t, [traceIDSize]byte{}, tid)
	assert.Equal(t, 0, tid.SizeProto())

	b := [traceIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	tid = b
	assert.EqualValues(t, b, tid)
	assert.Equal(t, traceIDSize, tid.SizeProto())
}

func TestTraceIDMarshal(t *testing.T) {
	buf := make([]byte, traceIDSize)

	tid := TraceID([traceIDSize]byte{})
	n := tid.MarshalProto(buf)
	assert.Equal(t, 0, n)

	tid = [traceIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	n = tid.MarshalProto(buf)
	assert.Equal(t, traceIDSize, n)
	assert.Equal(t, []byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, buf[0:traceIDSize])
}

func TestTraceIDUnmarshal(t *testing.T) {
	buf := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}

	tid := TraceID{}
	err := tid.UnmarshalProto(buf[:])
	require.NoError(t, err)
	assert.EqualValues(t, buf, tid)

	err = tid.UnmarshalProto(buf[0:0])
	require.NoError(t, err)
	assert.EqualValues(t, [traceIDSize]byte{}, tid)

	err = tid.UnmarshalProto(nil)
	require.NoError(t, err)
	assert.EqualValues(t, [traceIDSize]byte{}, tid)
}

func TestTraceIDMarshalAndUnmarshalJSON(t *testing.T) {
	stream := json.BorrowStream(nil)
	defer json.ReturnStream(stream)
	src := TraceID([traceIDSize]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78})
	src.MarshalJSON(stream)
	require.NoError(t, stream.Error())

	iter := json.BorrowIterator(stream.Buffer())
	defer json.ReturnIterator(iter)
	dest := TraceID{}
	dest.UnmarshalJSON(iter)
	require.NoError(t, iter.Error())

	assert.Equal(t, src, dest)
}
