// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestTraceID(t *testing.T) {
	tid := TraceID([16]byte{})
	assert.EqualValues(t, [16]byte{}, tid)
	assert.Equal(t, 0, tid.Size())

	b := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	tid = b
	assert.EqualValues(t, b, tid)
	assert.Equal(t, 16, tid.Size())
}

func TestTraceIDMarshal(t *testing.T) {
	buf := make([]byte, 20)

	tid := TraceID([16]byte{})
	n, err := tid.MarshalTo(buf)
	assert.Equal(t, 0, n)
	require.NoError(t, err)

	tid = [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	n, err = tid.MarshalTo(buf)
	assert.Equal(t, 16, n)
	assert.Equal(t, []byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, buf[0:16])
	require.NoError(t, err)

	_, err = tid.MarshalTo(buf[0:1])
	assert.Error(t, err)
}

func TestTraceIDUnmarshal(t *testing.T) {
	buf := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}

	tid := TraceID{}
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

func TestTraceIDMarshalAndUnmarshalJSON(t *testing.T) {
	stream := json.BorrowStream(nil)
	defer json.ReturnStream(stream)
	src := TraceID([16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78})
	src.MarshalJSONStream(stream)
	require.NoError(t, stream.Error())

	iter := json.BorrowIterator(stream.Buffer())
	defer json.ReturnIterator(iter)
	dest := TraceID{}
	dest.UnmarshalJSONIter(iter)
	require.NoError(t, iter.Error())

	assert.Equal(t, src, dest)
}
