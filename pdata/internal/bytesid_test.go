// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/json"
)

func TestUnmarshalJSON(t *testing.T) {
	iter := json.BorrowIterator(nil)
	defer json.ReturnIterator(iter)

	id := [16]byte{}
	unmarshalJSON(id[:], iter.ResetBytes([]byte(`""`)))
	require.NoError(t, iter.Error())
	assert.Equal(t, [16]byte{}, id)

	idBytes := [16]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}
	unmarshalJSON(id[:], iter.ResetBytes([]byte(`"12345678123456781234567812345678"`)))
	require.NoError(t, iter.Error())
	assert.Equal(t, idBytes, id)

	unmarshalJSON(id[:], iter.ResetBytes([]byte(`"nothex"`)))
	require.Error(t, iter.Error())

	unmarshalJSON(id[:], iter.ResetBytes([]byte(`"1"`)))
	require.Error(t, iter.Error())

	unmarshalJSON(id[:], iter.ResetBytes([]byte(`"123"`)))
	require.Error(t, iter.Error())

	unmarshalJSON(id[:], iter.ResetBytes([]byte(`"`)))
	require.Error(t, iter.Error())
}
