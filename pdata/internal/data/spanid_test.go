// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
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

func TestSpanIDMarshalAndUnmarshalJSON(t *testing.T) {
	stream := json.BorrowStream(nil)
	defer json.ReturnStream(stream)
	src := SpanID([8]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78})
	src.MarshalJSONStream(stream)
	require.NoError(t, stream.Error())

	iter := json.BorrowIterator(stream.Buffer())
	defer json.ReturnIterator(iter)
	dest := SpanID{}
	dest.UnmarshalJSONIter(iter)
	require.NoError(t, iter.Error())

	assert.Equal(t, src, dest)
}
