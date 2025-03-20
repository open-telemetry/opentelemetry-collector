// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSizeTypeUnmarshalText(t *testing.T) {
	var sizer SizerType
	require.NoError(t, sizer.UnmarshalText([]byte("bytes")))
	require.NoError(t, sizer.UnmarshalText([]byte("items")))
	require.NoError(t, sizer.UnmarshalText([]byte("requests")))
	require.Error(t, sizer.UnmarshalText([]byte("invalid")))
}

func TestSizeTypeMarshalText(t *testing.T) {
	val, err := SizerTypeBytes.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("bytes"), val)

	val, err = SizerTypeItems.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("items"), val)

	val, err = SizerTypeRequests.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("requests"), val)
}
