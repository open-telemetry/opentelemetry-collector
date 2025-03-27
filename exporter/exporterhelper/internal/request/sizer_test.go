// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestItemsSizer(t *testing.T) {
	sz := request.NewItemsSizer()
	assert.EqualValues(t, 3, sz.Sizeof(&requesttest.FakeRequest{Items: 3}))
}

func TestSizeTypeUnmarshalText(t *testing.T) {
	var sizer request.SizerType
	require.NoError(t, sizer.UnmarshalText([]byte("bytes")))
	require.NoError(t, sizer.UnmarshalText([]byte("items")))
	require.NoError(t, sizer.UnmarshalText([]byte("requests")))
	require.Error(t, sizer.UnmarshalText([]byte("invalid")))
}

func TestSizeTypeMarshalText(t *testing.T) {
	val, err := request.SizerTypeBytes.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("bytes"), val)

	val, err = request.SizerTypeItems.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("items"), val)

	val, err = request.SizerTypeRequests.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("requests"), val)
}
