// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request_test

import (
	"context"
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

func TestRequestsSizer(t *testing.T) {
	sz := request.RequestsSizer{}
	assert.EqualValues(t, 1, sz.Sizeof(&requesttest.FakeRequest{Items: 3}))
	assert.EqualValues(t, 4, sz.Sizeof(&requesttest.FakeRequest{Items: 3, Requests: 4}))
	assert.EqualValues(t, 1, sz.Sizeof(plainRequest{}))
}

type plainRequest struct{}

func (plainRequest) ItemsCount() int { return 0 }

func (plainRequest) BytesSize() int { return 0 }

func (plainRequest) MergeSplit(context.Context, int, request.SizerType, request.Request) ([]request.Request, error) {
	return []request.Request{plainRequest{}}, nil
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
